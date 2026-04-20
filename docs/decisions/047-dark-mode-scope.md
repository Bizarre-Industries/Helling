# ADR-047: Dark Mode Scope for v0.1

> Status: Accepted (2026-04-21)

## Context

`docs/design/tokens.md` and `docs/design/pages/settings.md` already treat v0.1 as light-only with dark mode deferred. The design tokens are structured so a dark palette is a token-swap, not a rewrite. The ADR needs to formalize what those docs already assume.

## Decision

**v0.1 ships light-only.** Dark mode lands in v0.5+ as a theme-toggle feature on the settings page.

Scope for v0.1:

- Single light theme defined in `docs/design/tokens.md`
- antd `ConfigProvider` configured with the light token set
- No user-facing theme toggle
- No `prefers-color-scheme` detection
- Design tokens structured to make a future dark palette a token-swap (already done; see tokens.md §4)

Scope deferred to v0.5+:

- Dark palette values filled into the Dark column placeholder in tokens.md
- Settings page theme toggle (currently deferred per settings.md)
- `prefers-color-scheme` OS preference detection
- Per-user persistence of theme choice

## Rationale

- Halves the QA matrix for v0.1 (visual regression, contrast validation, component state variants in both themes = 2× work)
- Design tokens are already structured for the token-swap migration — no rewrite cost when dark mode lands
- `function over beauty` (ADR-008) pushes back on polish features before product is shipped
- Operators who want dark styling at v0.1 can use browser extensions (Dark Reader, etc.) — this is not ideal but it's a known workaround

## Acceptance criteria for v0.1

- Every page passes WCAG 2.1 AA contrast in the single light theme
- No code paths read `prefers-color-scheme` or branch on theme
- tokens.md's Dark column remains a placeholder — no dead code from half-implemented dark support

## Acceptance criteria for v0.5+

- Dark palette values defined in tokens.md
- Settings page exposes a theme selector (Light / Dark / System)
- Per-user preference stored in the config domain (`ui.theme`)
- All pages pass WCAG 2.1 AA contrast in both themes
- Visual regression suite runs for both themes

## Consequences

**Easier (for v0.1):**

- Smaller QA matrix
- No dark-mode-specific component bugs to chase
- Clearer visual design baseline

**Harder (for v0.1):**

- Some operators running homelabs at night may be mildly annoyed
- Browser-extension workarounds have edge cases (contrast in extensions is imperfect)

## References

- `docs/design/tokens.md` §4 (Dark column placeholder, token-swap migration)
- `docs/design/pages/settings.md` (preset themes already deferred to v0.5+)
- ADR-008 (function over beauty)
