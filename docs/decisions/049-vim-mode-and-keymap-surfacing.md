# ADR-049: Vim mode as an opt-in modal layer over the action registry

> Status: Accepted

## Context

Helling's target users run 200+ instances across 16 nodes and navigate by keyboard all day (see docs/design/philosophy.md). The existing keyboard layer (docs/design/keyboard.md) is action-based and configurable, but it is positional, not modal: `j` on a list page is undefined, and multi-row operations require repeated keystrokes (`d`, `d`, `d`, …) rather than composed verbs (`3dd`).

Zed's action-based keymap plus opt-in vim layer is the canonical reference for doing this in a modern UI without splitting the user base. Helping users who prefer modal navigation without punishing users who do not is a strict requirement — we cannot break users by enabling vim mode globally.

Two smaller sub-decisions belong here because they're often conflated with "add vim mode":

1. Whether `Cmd+P` and `Cmd+Shift+P` should be treated as one shortcut (as the previous version of keyboard.md implied by listing `Cmd+P` as "reserved — print") or as two distinct bindings. They are objectively two distinct keystrokes that live in separate browser-intercepted namespaces: `Cmd+P` is "print", `Cmd+Shift+P` is not claimed by any browser.
2. Whether keybinding hints belong on button surfaces, in tooltips, or in both. Putting hints on button surfaces competes with the label and adds visual noise; tooltips and menu `extra` slots are the right surface, matching Zed, VS Code, Sublime, macOS menus.

## Decision

### 1. Vim mode is an opt-in layer, not a default

- Disabled by default. Users enable it in Settings → Keyboard. The setting is a single toggle (`vim_mode: true | false`) persisted in user preferences alongside the existing keymap overrides.
- When enabled, `vim.mode` becomes a new context axis in the resolver. Bindings in the `VimNormal` / `VimVisual` / `VimVisualLine` / `VimCommand` contexts only fire when that axis matches.
- Vim-mode actions register in the same action registry as every other action. The command palette, `?` cheatsheet, and Settings → Keyboard editor list them when vim mode is on; they do not appear to users who have not enabled it.
- No user can be broken by enabling vim mode elsewhere: the mode state is local to the user's account (server-side) with a per-device local override.

### 2. Scope of vim emulation

In scope for v0.1:

- Modal navigation over ProTable rows and detail-view tabs.
- Normal-mode motions (`h j k l`, `gg`, `G`, `Ctrl-d`, `Ctrl-u`, `n`, `N`, `/`).
- Operators (`d`, `y`) composable with motions and counts (`5j`, `d5j`, `V5jy`).
- Visual and visual-line modes driving ProTable `rowSelection`.
- Insert-mode auto-switch when a text input is focused; `Esc` returns to normal and defocuses.
- Command-line mode (`:`) opens the command palette in a vim-prefixed filter state; small set of `:q`, `:w`, `:help`, `:set vim=off`, `:h <actionId>` aliases.
- Terminal passthrough inside xterm.js: vim-mode resolver disabled; only `Esc` intercepted to exit passthrough.

Out of scope for v0.1:

- Text editor vim mode — the code editor (@uiw/react-codemirror) owns its own vim layer via codemirror-vim. This ADR is about navigating the UI, not editing YAML inside it.
- Registers, macros, marks, jumplist. Any of these can come back in v0.5+ if users ask.
- Helix mode (object-verb order). Listed in docs/design/keyboard.md as v0.5+.

### 3. `Cmd+P` is NOT reserved; it is a first-class binding

`Cmd+P` and `Cmd+Shift+P` are two distinct keystrokes in two different namespaces. Browsers only intercept the former (print dialog). The latter is free for application use and is the convention for command-palette-on-Cmd-Shift-P across VS Code, Sublime, Obsidian, and others. Helling uses:

- `Cmd+K` — primary command palette (Zed convention)
- `Cmd+Shift+P` — command palette alias (VS Code muscle memory)
- `Cmd+P` — quick-open resource by name (Zed's file finder equivalent)

Users who want the browser print dialog still reach it through the browser menu; Helling intercepts `Cmd+P` and displays the quick-open resource panel. The keymap editor warns on any re-bind that overlaps a literally browser-reserved shortcut (`Cmd+T`, `Cmd+W`, `Cmd+N`, `Cmd+Tab`, `F5`, `Cmd+L`, `Cmd+S`), but does not warn on `Cmd+P` because the browser's claim is weaker and the Zed/VS Code convention is to override it.

### 4. Shortcut hints live in tooltips and menu `extra` slots, not on button surfaces

- Every `<ActionButton>` wraps its label and the formatted shortcut in an antd `<Tooltip>`. Hover/focus shows `"Create Schedule ⌘K"` or equivalent; the button surface remains clean.
- Dropdown / context menu items use antd `Menu`'s `extra` slot to render the shortcut as a right-aligned `<kbd>` chip.
- The command palette shows each action's binding on the right side of its row.
- The `?` cheatsheet modal shows bindings grouped by category.
- The Settings → Keyboard editor shows current and default bindings side by side.

This matches macOS native menus, Zed, VS Code, Sublime, and Obsidian. The alternative — showing the chip directly on the button surface — was rejected because it competes with the label, duplicates the primary affordance, and generally produces the "everything shouts" feel we deliberately avoid (philosophy rule 1: information density over decoration).

Visual spec for the `<kbd>` chip lives in docs/design/tokens.md. It is the only place the chip is styled.

## Consequences

**Easier:**

- Solo power users can navigate 200+ VMs without moving off home row — `5j d` deletes 5 rows in the time a mouse user is still clicking the first checkbox.
- Existing users are unaffected. The default keymap is unchanged. Opt-in means opt-in.
- The command palette already exists (keyboard.md Architecture section); wiring `:` to it is trivial. No new primitive systems.
- `Cmd+Shift+P` adoption pulls in VS Code users with zero learning curve. The palette behaves identically, just bound to two keys.
- Button-tooltip integration already exists (keyboard.md `ActionButton` spec); the `<kbd>` chip in tokens.md is the final visual polish.

**Harder:**

- Resolver now has a two-keystroke lookahead for operator + motion composition (`d` + `j`). Needs a timeout (1 s) and count-accumulation state. Non-trivial state machine, but well-understood (every vim emulator does this).
- Test surface grows: every vim-mode action needs a unit test for composition behavior. E2E vim tests per page.
- Documentation burden: vim mode has to be kept in sync with non-vim mode as new actions land. Mitigation: single registry is the source of truth; adding a `vim_normal` default binding to an action is a two-line change, not a parallel doc.
- Conflict surface with Zed's `:` command set — we support a strict subset (`:q`, `:w`, `:help`, `:set vim=off`, `:h <actionId>`). Users with deep Zed muscle memory will find gaps; we ship docs telling them the palette is the escape hatch.
- `Cmd+P` intercept overrides a browser-native shortcut. Users who want the browser print dialog inside Helling (rare) must go through the browser menu. Documented in the keymap editor.

**Neutral:**

- Scope creep risk exists (registers, macros, marks). Contained by the ADR: none of those ship in v0.1.
- Helix mode defer is explicit. If demand emerges, helix mode adds a new value to the mode axis and a parallel keymap table; no further architectural change.

## Alternatives considered

**Rejected: ship vim mode as the default.**
Breaks every existing user on upgrade. Violates the "opt-in" commitment in keyboard.md principle #1. No.

**Rejected: fork the action system for vim users.**
Two registries to keep in sync. Doubles test surface. Abandons the "single source of truth" the keyboard architecture is built on.

**Rejected: embed a mature library (codemirror-vim outside the editor, or a port of Monaco's vim).**
These libraries are text-editor-centric. Driving them against ProTable rows and tabs is a worse fit than building a small, UI-navigation-focused layer on top of the existing resolver. Measured: ~600 LOC for the full v0.1 scope above.

**Rejected: put shortcut chips on button surfaces.**
Compared side-by-side in prototype. Chip-on-button competes with the label, produces a busy UI at ProTable density (a row with 3 action icons × 3 chip strings = visual noise). Tooltip + menu-extra matches macOS / Zed / VS Code idioms and is the 20-year-stable answer.

**Rejected: treat `Cmd+P` as off-limits.**
The previous keyboard.md did this. It's factually wrong — `Cmd+P` and `Cmd+Shift+P` are distinct keystrokes and every major editor overrides `Cmd+P`. Leaving `Cmd+P` to the browser means Helling users trying to print a PDF rendered by the app get a print dialog, which is not what they want in an admin UI.

## References

- docs/design/keyboard.md (Vim mode section; Default keymap Global table)
- docs/design/tokens.md (`<kbd>` chip visual spec)
- docs/spec/webui-spec.md (command palette referenced in layout)
- docs/design/magic.md #10 (command palette as a magic feature — now promoted to core)
- Zed's action-based keymap architecture: <https://zed.dev/docs/key-bindings>
- Zed's vim mode: <https://zed.dev/docs/vim>
