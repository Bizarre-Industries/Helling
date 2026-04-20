# Keyboard Navigation

<!-- markdownlint-disable MD040 -->

> Every action in Helling is reachable from the keyboard, bound to a named action (not a key), scoped to a context, configurable by the user, and surfaced in the UI as a hint next to the button or menu item that triggers it. Inspired by Zed's action-based keymap architecture, adapted for the browser.

## Design principles

1. **Actions, not keys, are the primitive.** `schedules.runNow` is the thing. Which key triggers it is secondary, configurable, and context-dependent. Same rule as Zed: the UI calls `dispatch('schedules.runNow')`, never a keystroke.
2. **Every action appears in the command palette.** `Cmd/Ctrl+K` opens it. Whatever you can do with the mouse, you can do by typing a fragment of the action label. Actions that have keybindings show their binding on the right side of the palette row.
3. **Every button in the UI shows its binding.** Hover (or focus) shows a tooltip: action label + current binding. Users learn shortcuts by using the UI, not by reading a cheat sheet.
4. **Context tree matches the DOM focus tree.** A binding only fires when its context is active. `Workspace > Pane > ProTable > Row` — bindings scoped to `ProTable` don't fire when a `Modal` is open on top.
5. **Configurable per-user.** Bindings persist under the user's account settings (server-side, same store as theme + layout preferences). A local-override layer allows per-device tweaks without pushing to the server.
6. **Avoid collisions with browser / OS shortcuts, but use the distinct namespace.** `Cmd+T`, `Cmd+W`, `Cmd+N`, `Cmd+Tab`, `F5`, `Cmd+L` (address bar focus), `Cmd+S` (save page) are off-limits — these literally cannot be intercepted on most browsers. But `Cmd+P` and `Cmd+Shift+P` are **different keystrokes in separate namespaces** — browsers only claim `Cmd+P` (print). `Cmd+Shift+P` is free for our use (this is the exact convention VS Code, Zed, Sublime, and Obsidian all follow — command palette on `Cmd+Shift+P`, and nothing on `Cmd+P`). Helling binds `Cmd+K` to palette as primary (Zed's convention) and `Cmd+Shift+P` as a secondary alias for muscle memory from VS Code users. `Cmd+P` alone is bound to "quick open resource" (Zed's file finder). Users who re-bind to literally reserved shortcuts (`Cmd+T`, `Cmd+W`) get a warning in the keymap editor.
7. **Discoverability beats memorization.** The `?` key opens the shortcut reference modal, filtered to the current context. The command palette finds actions by text. Button tooltips show bindings. A user should never need to read this doc — except as a contributor implementing new actions.

## Architecture

### Action registry

Every action the UI can perform is a typed entry in a single registry at `web/src/keyboard/registry.ts`:

```ts
export interface ActionDef {
  id: string; // "schedules.runNow"
  label: string; // "Run schedule now" (shown in palette + tooltip)
  description?: string; // Long-form help for the keymap editor
  category: ActionCategory; // "Schedules" | "Instances" | "Global" | ...
  defaultBinding?: Keystroke | Keystroke[]; // e.g. "r" or ["g", "s"]
  context: ContextExpression; // when the action is available
  handler: (dispatchArgs?: unknown) => void; // what it does
  hidden?: boolean; // if true, excluded from palette / cheatsheet
}
```

Actions are registered from the component that owns them (a hook, `useAction()`), not from a central list that grows stale. The registry is the single source of truth for:

- The command palette result list
- The `?` modal content
- The keymap editor
- Button tooltip hints (a component asks the registry "what binding currently fires action X?")
- Automated tests that verify every primary button has a matching action

### Context tree

Mirrors Zed's context model, simplified for a web admin UI. Contexts are arranged in a tree rooted at `Workspace`:

```
Workspace
├── CommandPalette           (top-most when open)
├── Modal                    (when any modal is open)
├── Drawer
│   └── TaskLogDrawer
├── Sidebar
│   └── ResourceTree
├── Page                     (whichever route is active; tag = route name)
│   ├── ProTable
│   │   └── Row              (row focused)
│   ├── Tabs
│   │   └── Tab              (tab focused)
│   ├── Terminal             (xterm.js: swallows all keystrokes except escape hatches)
│   └── CodeEditor           (@uiw/react-codemirror)
```

A binding can target a specific context expression (`Page=schedules && Row` — fires on row of schedules page), or a broader one (`ProTable` — fires on any row of any table). More specific contexts win on conflict.

### Binding resolver

One pass per keystroke:

1. Capture keystroke, normalize (`Cmd` / `Meta` on macOS; `Ctrl` on others by default; user can swap).
2. Walk the active context chain from leaf to root.
3. At each level, look up user overrides first, then defaults.
4. First match wins. Chord-in-progress (`g` waiting for second key) holds state for 500ms then resets.
5. If a keystroke would pass through to the browser or terminal AND has a binding, the binding fires and the event is `preventDefault()`ed.
6. Terminal and code-editor contexts suppress the resolver except for an allowlist (`Escape`, `Cmd/Ctrl+Shift+C/V`, `Cmd/Ctrl+K` for palette).

Implementation: own lightweight resolver over `react-hotkeys-hook` primitives. Zed's full context-expression grammar (`!`, `&&`, `||`, `os=macos`) is overkill for our needs; we use a simpler subset — a leaf context tag plus optional parent requirement.

### Button tooltip integration

Every actionable antd component (`Button`, `Dropdown.Item`, `Menu.Item`, row-action icons) is wrapped in an `ActionButton` / `ActionMenuItem` / `ActionIcon` component that takes an `action` prop instead of an `onClick` prop:

```tsx
<ActionButton type="primary" action="schedules.create" icon={<Plus />}>
  Create Schedule
</ActionButton>
```

The component:

- Resolves the action from the registry at render time
- Calls `action.handler` on click
- Shows the label from the registry (unless overridden by children)
- Wraps in antd `<Tooltip>` showing `${label} · ${formattedBinding}` on hover/focus
- Becomes disabled if the action's context expression isn't currently satisfied

This is the mechanism that makes shortcuts visible in the UI — you never declare a shortcut hint separately from the button. The tooltip is generated from the same data the resolver uses. A lint rule enforces `<ActionButton>` over raw `<Button onClick>` for actions that have a binding.

### Shortcut indicator in menus

Dropdown menus and context menus also render the binding. antd's `Menu` component accepts an `extra` slot on each item; we populate it with the formatted binding:

```tsx
<Menu items={[
  { key: 'start', label: 'Start', extra: kbd('S'), onClick: ... },
  { key: 'stop',  label: 'Stop',  extra: kbd('Shift+S'), onClick: ... },
]} />
```

`kbd()` renders `<kbd>` elements styled per the token spec — small, monospace, framed — matching the way macOS and Zed render binding hints.

### Command palette

`Cmd/Ctrl+K` opens an antd `<Modal>` with a list of actions filtered by fuzzy match on label and category. Each row:

- Left: action icon (from registry) + label + category
- Right: current binding as styled `<kbd>`s, or `—` if unbound

Palette actions:

- `↑` / `↓`: move selection
- `Enter`: dispatch selected action
- `Cmd/Ctrl+K` again (or `Escape`): close

The palette is context-aware — only actions whose context is currently satisfied appear, unless the user types `>` (Zed convention) to see all actions including disabled ones.

### `?` cheatsheet modal

`?` (when no input is focused) opens a modal listing every action available in the current context, grouped by category, with binding on the right. This is a read-only view; the "Edit" button jumps to Settings → Keyboard.

### Settings → Keyboard (the keymap editor)

A dedicated page under `/settings/keyboard` (or a tab in Settings). Antd `ProTable` with:

- Action ID (monospace, copyable)
- Label
- Category filter
- Context (monospace tag)
- Default binding (shown for reference)
- Current binding (editable via an inline "Record keystroke" input)
- Reset-to-default button per row

Bulk actions: Export as JSON, Import from JSON, Reset all to default.

Conflict detection on save: if two actions in overlapping contexts share a binding, the editor highlights both rows red and refuses to save until resolved. A "why?" tooltip explains which context resolution rule would pick the winner.

### Persistence

Two layers:

1. **Server-side** — stored per-user via a new `/api/v1/users/{id}/preferences/keymap` endpoint (v0.5+; not in v0.1 openapi yet). JSON schema: `{ overrides: { [actionId]: Keystroke | Keystroke[] | null } }`. `null` explicitly unbinds.
2. **Local-override** — `localStorage` layer for per-device quirks (laptop vs external keyboard). Users can toggle "Use local overrides on this device" in the keymap editor.

Server-side is the source of truth across sessions/devices. Local wins on the device it was set.

### Keystroke format

Same normalized string format as Zed, simplified:

- Modifiers: `cmd`, `ctrl`, `alt`, `shift` (lowercase, hyphen-joined, in that canonical order)
- Keys: lowercase letter/digit/symbol, or named (`enter`, `escape`, `tab`, `space`, `up`, `down`, `left`, `right`, `home`, `end`, `pageup`, `pagedown`, `backspace`, `delete`, `f1`..`f12`)
- Chords: space-separated (`g s` means `g` then `s` within the chord timeout)
- Cross-platform: in config, `cmd` means `Cmd` on macOS and `Ctrl` on others by default. Users can pin to one or the other with `meta` or `ctrl` explicitly.

Examples:

```
"cmd-k"               → Cmd+K on macOS, Ctrl+K on others
"ctrl-k"              → Ctrl+K always
"g s"                 → press g, then s within 500ms
"shift-s"             → Shift+S
"cmd-shift-p"         → Cmd+Shift+P
```

## Default keymap

These are the defaults — every binding here can be overridden in Settings → Keyboard.

### Global

| Action ID            | Default binding | Label                                                          |
| -------------------- | --------------- | -------------------------------------------------------------- |
| `palette.open`       | `cmd-k`         | Open command palette                                           |
| `palette.open.alt`   | `cmd-shift-p`   | Open command palette (VS Code alias)                           |
| `resource.quickOpen` | `cmd-p`         | Quick open resource by name (instances, containers, templates) |
| `search.focus`       | `/`             | Focus current page filter                                      |
| `help.shortcuts`     | `?`             | Show shortcuts for this context                                |
| `nav.dashboard`      | `g d`           | Go to Dashboard                                                |
| `nav.instances`      | `g i`           | Go to Instances                                                |
| `nav.containers`     | `g c`           | Go to Containers                                               |
| `nav.kubernetes`     | `g k`           | Go to Kubernetes                                               |
| `nav.storage`        | `g s`           | Go to Storage                                                  |
| `nav.networking`     | `g n`           | Go to Networking                                               |
| `nav.firewall`       | `g f`           | Go to Firewall                                                 |
| `nav.backups`        | `g b`           | Go to Backups                                                  |
| `nav.logs`           | `g l`           | Go to Logs                                                     |
| `nav.audit`          | `g a`           | Go to Audit                                                    |
| `nav.users`          | `g u`           | Go to Users (admin only)                                       |
| `nav.settings`       | `g ,`           | Go to Settings                                                 |
| `nav.back`           | `cmd-left`      | Back to previous view                                          |
| `dismiss`            | `escape`        | Close topmost modal / drawer / popconfirm                      |

### List views (context: `ProTable`)

| Action ID               | Default binding | Label                             |
| ----------------------- | --------------- | --------------------------------- |
| `list.moveUp`           | `up`            | Move row focus up                 |
| `list.moveDown`         | `down`          | Move row focus down               |
| `list.openRow`          | `enter`         | Open focused row                  |
| `list.toggleSelect`     | `space`         | Toggle row selection              |
| `list.extendSelectUp`   | `shift-up`      | Extend selection up               |
| `list.extendSelectDown` | `shift-down`    | Extend selection down             |
| `list.selectAllOnPage`  | `cmd-a`         | Select all rows on page           |
| `list.create`           | `c`             | Create new item                   |
| `list.editFocused`      | `e`             | Edit focused row                  |
| `list.deleteFocused`    | `d`             | Delete focused row (with confirm) |
| `list.reload`           | `cmd-r`         | Reload list                       |

### Detail views (context: `Page > Tabs`)

| Action ID                    | Default binding | Label         |
| ---------------------------- | --------------- | ------------- |
| `tabs.first` .. `tabs.ninth` | `1` .. `9`      | Jump to tab N |
| `tabs.next`                  | `]`             | Next tab      |
| `tabs.prev`                  | `[`             | Previous tab  |

### Per-page overrides

Each page registers its own actions. The palette and `?` modal filter by context automatically.

**Instance detail (`Page=instance`):** `instance.start` (`s`), `instance.stop` (`shift-s`), `instance.restart` (`r`), `instance.snapshot` (`p`), `instance.backupNow` (`b`), `instance.openConsole` (`o`).

**Container detail (`Page=container`):** `container.start` (`s`), `container.stop` (`shift-s`), `container.restart` (`r`), `container.gotoLogs` (`l`), `container.gotoExec` (`x`).

**Workspaces (`Page=workspaces`):** `workspaces.launch` (`l`), `workspaces.openConsole` (`enter`), `workspaces.destroy` (`d`).

**Logs (`Page=logs`):** `logs.toggleFollow` (`f`), `logs.toggleTimestamps` (`t`), `logs.jumpToTail` (`cmd-end`), `logs.jumpToHead` (`cmd-home`).

**Schedules (`Page=schedules`):** `schedules.runNow` (`r`), `schedules.toggleEnable` (`space`).

**Audit (`Page=audit`):** `audit.export` (`e`), `audit.clearFilters` (`c`).

**Kubernetes cluster (`Page=cluster`):** `cluster.gotoKubectl` (`k`), `cluster.downloadKubeconfig` (`shift-k`), `cluster.upgrade` (`u`).

Full canonical list lives in `web/src/keyboard/registry.ts`. The doc you're reading is kept in sync manually; the registry is the source of truth.

## Vim mode

An opt-in modal navigation layer. Disabled by default. When enabled (user toggle in Settings → Keyboard), the resolver gates a second keymap layer on `vim.mode` state.

### Why vim mode

Power users running 200+ instances across 16 nodes navigate with the keyboard all day. Modal editing collapses long chords into short, composable motions — `5dd` is five row deletions, not five `d` presses. The same users who drove the "tables, not cards" philosophy (docs/design/philosophy.md) want `j`/`k` to move rows and `:` to run commands. This is who Helling is for.

Zed's vim layer is the reference: actions stay the primitive, modes and motions dispatch actions, and non-vim users are unaffected because the feature is opt-in. Same model here.

### What vim mode IS

- Modal navigation over the existing action registry. Modes gate which actions are available.
- Normal / visual / insert / command-line modes, scoped per surface (ProTable rows, detail views, code editor, terminal passthrough).
- Composable counts and operators where they make sense (`5j` to move 5 rows, `d5j` to select 5 rows down and delete).
- A command-line mode (`:`) that opens the command palette pre-filtered, matching Zed's `:` → palette convention.

### What vim mode is NOT

- Not a full vim emulator. No macros, no marks (yet), no registers.
- Not a text editor vim mode — the code editor (`@uiw/react-codemirror`) has its own vim layer (codemirror-vim). This section is about navigating the UI, not editing YAML.
- Not the default. Users opt in via `Settings → Keyboard → Vim mode: enabled`.

### Mode state

`vim.mode` is a new context axis the resolver reads, alongside the DOM-focus context tree. Values:

| Mode          | When active                                                      | What it affects                                                       |
| ------------- | ---------------------------------------------------------------- | --------------------------------------------------------------------- |
| `normal`      | Default when vim mode is on and no text input is focused         | Motions (`hjkl`, `gg`, `G`), operators (`d`, `y`), counts (`5`, `10`) |
| `visual`      | After `v` from normal, or from a `rowSelection` state            | Motion extends selection; operator acts on selection                  |
| `visual-line` | After `V` from normal                                            | Whole-row selection                                                   |
| `insert`      | When any text input, textarea, or CodeMirror instance is focused | All vim bindings disabled; keys pass through to the input             |
| `command`     | After `:` from normal                                            | Command palette open, filter string editable                          |

Mode indicator: a compact status element in the bottom-right of the ProLayout footer, styled like Zed's mode pill. Colors per `color.status.*` from `docs/design/tokens.md`. Hidden when vim mode is off.

### Normal-mode keymap (context: `VimNormal`)

These bindings exist ONLY when `vim.mode == normal` AND no text input is focused. They don't override non-vim users because `VimNormal` is not in the default context tree.

**Motions (within a ProTable):**

| Binding   | Action ID              | Behavior                                     |
| --------- | ---------------------- | -------------------------------------------- |
| `h`       | `vim.list.columnLeft`  | (reserved; columns not scrollable in v0.1)   |
| `j`       | `list.moveDown`        | Move row focus down                          |
| `k`       | `list.moveUp`          | Move row focus up                            |
| `l`       | `list.openRow`         | Open focused row (= `enter`)                 |
| `g g`     | `list.moveToFirst`     | Jump to first row                            |
| `shift-g` | `list.moveToLast`      | Jump to last row                             |
| `ctrl-d`  | `list.halfPageDown`    | Half-page down                               |
| `ctrl-u`  | `list.halfPageUp`      | Half-page up                                 |
| `n`       | `list.nextSearchMatch` | Next search match                            |
| `shift-n` | `list.prevSearchMatch` | Previous search match                        |
| `/`       | `search.focus`         | Focus filter input → switches to insert mode |

**Counts:** Digits `1`-`9` accumulate a count, consumed by the next motion or operator. `5j` moves 5 rows down. `10shift-g` jumps to row 10. Count + operator + motion also works: `d5j` selects 5 rows down and deletes.

**Operators:**

| Binding   | Action ID             | Behavior                                                                       |
| --------- | --------------------- | ------------------------------------------------------------------------------ |
| `d`       | `vim.operator.delete` | With visual selection or motion: delete focused rows (with confirm)            |
| `y`       | `vim.operator.yank`   | Yank focused row(s) to clipboard as JSON (pattern: serialize resource summary) |
| `p`       | `vim.operator.paste`  | (future; context-dependent)                                                    |
| `v`       | `vim.mode.visual`     | Enter visual mode; motions extend selection                                    |
| `shift-v` | `vim.mode.visualLine` | Enter visual-line mode                                                         |
| `escape`  | `vim.mode.normal`     | Return to normal mode from any other                                           |

**Navigation between pages (normal mode):** Works identically to non-vim users — `g d` dashboard, `g i` instances, etc. Existing `g`-prefix global navigation is a subset of vim's native `g` operator space, designed so both populations use the same bindings.

**Page-specific actions in normal mode:** Single-letter page bindings (`s` start, `shift-s` stop, `r` restart, etc.) still fire in normal mode. Vim users wanted modal control, not a different action vocabulary.

**Tabs in detail views:** `gt` next tab (vim's native), `shift-g shift-t` previous tab, or numeric `1`-`9` via the existing `tabs.first..ninth` actions.

### Command-line mode (`:`)

Press `:` in normal mode to open the command palette in a special mode:

- Palette input pre-filled with `:` prefix
- Typing filters as usual
- `Enter` dispatches the action
- Supports a small set of vim-flavored aliases:
  - `:q` / `:quit` → `nav.back`
  - `:w` → no-op (no unsaved state in a REST UI; noop with "nothing to save" toast)
  - `:help` → open `?` cheatsheet
  - `:set vim=off` → disable vim mode (toggles the setting)
  - `:h {actionId}` → jump to Settings → Keyboard with that action highlighted
- `Esc` closes palette, returns to normal mode
- `Cmd+K` and `Cmd+Shift+P` open the same palette without the vim prefix (regular palette)

### Visual mode

Visual mode drives `rowSelection` in ProTable. Entering visual mode starts selection on the focused row. Motions extend it. Operators act on it:

- `v` then `j j j` then `d` → select 4 rows, delete them (with confirm)
- `shift-v` then `5 j` then `y` → select 6 rows line-wise, yank as JSON to clipboard
- `escape` → clear selection, back to normal

### Insert mode

Automatic. Any focused text input (antd `Input`, `Input.Search`, `Select` search, `DatePicker`, CodeMirror) switches `vim.mode` to `insert`. Leaving focus returns to `normal`. `Esc` inside an input also returns to normal mode (mimicking vim's insert → normal transition), and defocuses the input.

### Terminal passthrough

Inside xterm.js (console, exec, kubectl tabs), vim mode is disabled — the terminal itself may be running vim or neovim, and double-intercepting would break keystrokes. When focus enters a terminal, `vim.mode` becomes `passthrough`; only `Esc` (to exit passthrough) is intercepted.

### Helix mode

Not in v0.1. Helix reverses vim's verb-object order to object-verb. Zed ships both (docs/spec/webui-spec.md → keybindings → Helix Mode). If user demand emerges, we ship helix mode as a third value for `vim.mode` — same registry, different keymap. Tracked for v0.5+.

### Conflicts with the default keymap

Some bindings already in the default keymap are vim-native: `/`, `?`, `Esc`. These work identically in both modes — they're not "vim bindings," they're universal conventions the default keymap adopted from vim.

New conflicts introduced by vim mode:

- `j` / `k` in non-vim mode do nothing on rows (arrow keys handle this); in normal mode, `j` / `k` move row focus
- `d` in non-vim mode deletes focused row directly; in normal mode, `d` is an operator requiring a motion or selection
- `v` in non-vim mode does nothing; in normal mode, enters visual mode

Resolution: vim bindings live in the `VimNormal` / `VimVisual` context. When vim mode is off, that context is never active, and the non-vim defaults apply unchanged. No migration needed for existing users.

### Implementation notes

- One context layer (`vim.mode`) added to the resolver. No new resolver logic — the existing "more specific context wins" rule handles it.
- Motion + operator composition is a two-keystroke lookahead in the resolver: after `d`, wait up to 1s for a motion key (`j`, `k`, `5j`, `gg`, `G`). Timeout reverts to normal and fires no action.
- Count accumulation (`5`, `10`) is transient state in the resolver, not context. Cleared after any non-digit key.
- Register every vim-mode action in the same registry as non-vim actions. The command palette and keymap editor list them naturally when vim mode is enabled.
- Mode-indicator component subscribes to `vim.mode` changes. Zero re-renders in non-vim users because the component isn't mounted.

- Every binding is discoverable without the keyboard — tooltips, command palette, cheatsheet modal, keymap editor all work via mouse.
- Screen readers announce the action label, not just the keybinding, on focus.
- Focus indicators are always visible (per `docs/spec/accessibility.md`); keyboard users can see where they are.
- No binding traps focus in a way that prevents `Tab` escape; `Escape` always unwinds one level.
- Terminal / code-editor contexts announce keystroke passthrough mode to assistive tech when entered.

## Testing

- Every action registration generates a unit test asserting: `ActionButton` renders with correct tooltip, palette includes the action, `?` modal includes the action in the right category.
- Snapshot test: exported default keymap JSON matches a golden file. Changes force a conscious update.
- E2E (Playwright): for each page, simulate every default binding and assert the correct API call / navigation occurred.

## Implementation checklist (for the engineer building this)

- [ ] `web/src/keyboard/registry.ts` — typed action registry + helpers
- [ ] `web/src/keyboard/resolver.ts` — context-aware binding resolver
- [ ] `web/src/keyboard/provider.tsx` — React provider + `useAction()` hook
- [ ] `web/src/components/ActionButton.tsx`, `ActionIcon.tsx`, `ActionMenuItem.tsx` — tooltip-aware wrappers
- [ ] `web/src/components/CommandPalette.tsx` — fuzzy search over registry
- [ ] `web/src/components/ShortcutCheatsheet.tsx` — `?` modal
- [ ] `web/src/pages/settings/Keyboard.tsx` — editor with record-keystroke input, conflict detection, import/export
- [ ] `web/src/keyboard/vim/` — vim mode layer: mode state, operator composition, count accumulation, mode indicator (opt-in, off by default)
- [ ] `web/src/components/VimModeIndicator.tsx` — status pill in ProLayout footer
- [ ] `/api/v1/users/{id}/preferences/keymap` endpoint (v0.5+; out of v0.1 scope — see docs/spec/auth-v0.5.md for the broader preferences surface)
- [ ] `Kbd` token in `docs/design/tokens.md` + styling for `<kbd>` — already referenced from here; tokens doc owns the visual

## Cross-references

- docs/design/magic.md #10 (command palette is a first-class feature, not a magic extra)
- docs/design/tokens.md (Kbd / button / tooltip tokens — add a `Kbd` subsection)
- docs/spec/accessibility.md (WCAG 2.1 AA — keyboard reachability)
- docs/spec/webui-spec.md ("Related docs" index)
- docs/design/pages/\*.md (each page lists its actions — those IDs match this registry)
- Zed's action-based keymap model: <https://zed.dev/docs/key-bindings>
