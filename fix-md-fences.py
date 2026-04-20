#!/usr/bin/env python3
"""
scripts/fix-md-fences.py

Auto-labels bare markdown code fences (MD040 violations) by inspecting
the fence content and inferring the language.

Usage:
    python3 scripts/fix-md-fences.py                     # fix every *.md in repo (excluding refs/)
    python3 scripts/fix-md-fences.py --dry-run           # show what would change, don't write
    python3 scripts/fix-md-fences.py path/to/file.md ... # only fix specified files

Heuristic labels, in priority order:
    bash    — shebang, sudo, common unix commands, $ prompts
    yaml    — starts with `key: value` or `- key: ...`
    json    — starts with `{` or `[`
    toml    — starts with `[section]` or `key = value`
    go      — starts with `package`, `import`, `func`, `var`, `const`
    sql     — starts with SELECT/INSERT/CREATE/etc (case-insensitive)
    ini     — has `[section]` headers with `key=value` bodies
    text    — fallback (file trees, plain output, ASCII diagrams)

This is deliberately conservative: when in doubt, it picks `text`, which
satisfies MD040 without falsely claiming a syntax.
"""

from __future__ import annotations

import argparse
import pathlib
import re
import sys

# Commands that strongly suggest bash content.
BASH_COMMANDS = {
    "sudo", "cd", "ls", "mkdir", "rm", "mv", "cp", "chmod", "chown",
    "find", "grep", "sed", "awk", "cat", "echo", "export", "tar",
    "curl", "wget", "git", "go", "npm", "bun", "yarn", "task",
    "make", "docker", "podman", "incus", "systemctl", "journalctl",
    "apt", "apt-get", "dnf", "yum", "pacman", "brew", "pip", "pipx",
    "cargo", "rustc", "python3", "python", "node", "ssh", "scp",
    "helling", "hellingd", "hellingctl", "lefthook", "goose", "sqlc",
    "vacuum", "gitleaks", "shellcheck", "shfmt", "markdownlint-cli2",
    "prettier", "typos", "lychee", "yamllint", "biome", "bunx",
    "golangci-lint", "gofumpt", "goimports", "oapi-codegen", "govulncheck",
    "kubectl", "helm", "terraform", "ansible",
}

# Go keywords that typically appear at the top of a code block.
GO_PREFIXES = ("package ", "import ", "func ", "type ", "var ", "const ")

# SQL keywords (uppercase-first check).
SQL_KEYWORDS = {"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER",
                "DROP", "WITH", "BEGIN", "COMMIT", "PRAGMA", "EXPLAIN"}


def classify(content: str) -> str:
    """Inspect block content and return a best-guess language label."""
    if not content.strip():
        return "text"

    stripped = content.strip()
    lines = [ln for ln in stripped.splitlines() if ln.strip()]
    if not lines:
        return "text"
    first = lines[0].lstrip()

    # Shebang → bash (even if content is Python; the fence label is about
    # the surrounding shell context, not the script it's invoking).
    if stripped.startswith("#!"):
        if "python" in lines[0]:
            return "python"
        if "node" in lines[0]:
            return "javascript"
        return "bash"

    # JSON — must look like a document, not just a JSON-ish fragment.
    if stripped.startswith(("{", "[")) and stripped.rstrip().endswith(("}", "]")):
        # Distinguish JSON from JavaScript by absence of function/const/etc.
        if not re.search(r"\b(function|const|let|var|=>)\b", stripped):
            return "json"

    # TOML / INI — [section] header at start.
    if re.match(r"^\[[\w.-]+\]\s*$", first):
        # TOML allows `key = value`; INI typically uses `key=value` without spaces.
        if re.search(r"^[\w.-]+\s*=\s*", stripped, re.MULTILINE):
            return "toml"
        return "ini"

    # Go — starts with package/import/func/etc.
    if any(first.startswith(p) for p in GO_PREFIXES):
        return "go"

    # SQL — first word is a SQL keyword (case-sensitive uppercase check first,
    # then case-insensitive as fallback).
    first_word = first.split()[0] if first.split() else ""
    if first_word in SQL_KEYWORDS:
        return "sql"
    if first_word.upper() in SQL_KEYWORDS and len(lines) > 1:
        # Multi-line with SQL keyword — lowercase SQL.
        return "sql"

    # YAML — `key: value` or `- key: value` pattern.
    # Must have colon-space pattern as dominant shape.
    yaml_like_lines = sum(1 for ln in lines if re.match(r"^\s*(-\s+)?[\w.-]+:\s*", ln))
    if yaml_like_lines >= max(1, len(lines) // 2):
        # Make sure it's not actually a file listing with `foo:` style headers.
        if not any(ln.rstrip().endswith("/") for ln in lines[:5]):
            return "yaml"

    # Bash — multi-criterion check.
    #  - $ prompt
    #  - `#` comment at start (but not a shebang, handled above)
    #  - first token is a known bash command
    if first.startswith("$ ") or first.startswith("# ") and len(lines) > 1:
        return "bash"

    first_token = first.split()[0] if first.split() else ""
    # Strip trailing ; or & from token.
    first_token = first_token.rstrip(";&|")
    if first_token in BASH_COMMANDS:
        return "bash"

    # Multi-line blocks that LOOK like shell (has && or | or > redirect).
    if "&&" in stripped or " | " in stripped or ">>" in stripped:
        if any(ln.lstrip().split()[0].rstrip(";&|") in BASH_COMMANDS
               for ln in lines if ln.strip()):
            return "bash"

    # File tree / ASCII box — lots of box-drawing or slash-separators.
    if re.search(r"[├└│─┌┐┘┬┴┼]", stripped):
        return "text"
    if sum(1 for ln in lines if "/" in ln and not ln.lstrip().startswith("#")) >= len(lines) // 2:
        return "text"

    # Default fallback.
    return "text"


def fix_file(path: pathlib.Path, dry_run: bool = False) -> tuple[int, list[tuple[int, str]]]:
    """Scan file, label bare fences. Returns (count_changed, list_of_changes)."""
    text = path.read_text()
    lines = text.split("\n")
    out: list[str] = []
    changes: list[tuple[int, str]] = []

    # Regex definitions:
    # - bare_fence_re:    ```   or ~~~  (nothing after)  — needs labeling
    # - labeled_fence_re: ```<lang>  or ~~~<lang>       — already labeled; passthrough
    bare_fence_re = re.compile(r"^(\s*)(```|~~~)\s*$")
    labeled_fence_re = re.compile(r"^(\s*)(```|~~~)[a-zA-Z0-9_+.-]+\s*$")

    i = 0
    while i < len(lines):
        line = lines[i]

        # Already-labeled opening fence: passthrough entire block.
        lm = labeled_fence_re.match(line)
        if lm:
            marker = lm.group(2)
            out.append(line)
            i += 1
            # Copy body and close fence verbatim.
            close_re = re.compile(rf"^\s*{re.escape(marker)}\s*$")
            while i < len(lines):
                out.append(lines[i])
                if close_re.match(lines[i]):
                    i += 1
                    break
                i += 1
            continue

        # Bare opening fence: needs labeling.
        bm = bare_fence_re.match(line)
        if bm:
            indent = bm.group(1)
            marker = bm.group(2)
            close_re = re.compile(rf"^\s*{re.escape(marker)}\s*$")

            # Scan ahead to find the close fence, collecting body lines.
            j = i + 1
            body_lines: list[str] = []
            while j < len(lines) and not close_re.match(lines[j]):
                body_lines.append(lines[j])
                j += 1

            # Classify and emit labeled opening fence.
            lang = classify("\n".join(body_lines))
            out.append(f"{indent}{marker}{lang}")
            changes.append((i + 1, lang))

            # Emit body verbatim.
            out.extend(body_lines)

            # Emit close fence and advance past it.
            if j < len(lines):
                out.append(lines[j])
                i = j + 1
            else:
                # Unclosed fence — malformed markdown. Bail gracefully.
                i = j
            continue

        # Not a fence line: copy verbatim.
        out.append(line)
        i += 1

    new_text = "\n".join(out)
    if new_text != text and not dry_run:
        path.write_text(new_text)

    return len(changes), changes


def find_target_files(args_paths: list[str]) -> list[pathlib.Path]:
    """Return the list of files to process."""
    if args_paths:
        return [pathlib.Path(p) for p in args_paths if pathlib.Path(p).is_file()]

    # Default: all .md in repo except refs/, node_modules/, etc.
    excluded_parts = {"refs", "node_modules", ".task", "dist", "bin", ".git"}
    results: list[pathlib.Path] = []
    for path in pathlib.Path(".").rglob("*.md"):
        if any(part in excluded_parts for part in path.parts):
            continue
        results.append(path)
    return sorted(results)


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Auto-label bare markdown code fences (fix MD040 violations).",
    )
    parser.add_argument("paths", nargs="*", help="Specific files to process (default: all .md)")
    parser.add_argument("--dry-run", action="store_true",
                        help="Show proposed changes; don't write files")
    parser.add_argument("--verbose", "-v", action="store_true",
                        help="Show per-fence classification decisions")
    args = parser.parse_args()

    files = find_target_files(args.paths)
    if not files:
        print("No markdown files found.", file=sys.stderr)
        return 1

    total_files_changed = 0
    total_fences_labeled = 0
    lang_counts: dict[str, int] = {}

    for path in files:
        count, changes = fix_file(path, dry_run=args.dry_run)
        if count > 0:
            total_files_changed += 1
            total_fences_labeled += count
            print(f"{'would fix' if args.dry_run else 'fixed'}: {path} ({count} fences)")
            if args.verbose:
                for lineno, lang in changes:
                    print(f"    line {lineno}: → {lang}")
            for _, lang in changes:
                lang_counts[lang] = lang_counts.get(lang, 0) + 1

    if total_fences_labeled == 0:
        print("No bare fences found. Nothing to fix.")
        return 0

    print()
    print(f"{'(dry-run) ' if args.dry_run else ''}"
          f"{total_fences_labeled} fences labeled across {total_files_changed} files.")
    print("Language breakdown:")
    for lang, n in sorted(lang_counts.items(), key=lambda kv: -kv[1]):
        print(f"  {lang:<12} {n:>4}")

    if args.dry_run:
        print()
        print("Re-run without --dry-run to apply.")

    return 0


if __name__ == "__main__":
    sys.exit(main())
