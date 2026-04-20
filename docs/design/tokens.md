# Design Tokens

<!-- markdownlint-disable MD040 -->

> Canonical source for Helling WebUI design tokens. Every color, spacing, radius, and typography decision lives here. Components and pages reference tokens by name, not by literal value.

## Principles

1. **Semantic over literal.** `color.text.secondary` — not `#7a7a7a`. Refactoring a palette is a token change, not a find/replace.
2. **antd first.** Helling rides antd's theme token system (`antd/theme`). We override specific tokens; we don't build a parallel system.
3. **Compact by default.** All sizing tokens reflect ADR-008 ("function over beauty"). antd's default sizes are too airy for admin-density.
4. **Dark mode is deferred but structured.** Per ADR-047, v0.1 is light-only. Tokens are structured so a dark palette is a token-swap, not a rewrite.
5. **No one-off tokens.** If a value is used in one place, inline it. Promote to a token once it's used ≥2 places OR carries semantic meaning.

## antd ConfigProvider mapping

These tokens plug directly into `<ConfigProvider theme={{ token, components }}>` at the app root.

### Global tokens

| Token              | Value                                                                 | Semantic use                                                                |
| ------------------ | --------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `borderRadius`     | `4`                                                                   | All radius corners (buttons, cards, inputs) — quarter of antd's default `8` |
| `fontSize`         | `13`                                                                  | Body text — denser than antd default `14`                                   |
| `fontSizeHeading1` | `20`                                                                  | Page title                                                                  |
| `fontSizeHeading2` | `16`                                                                  | Section header                                                              |
| `fontSizeHeading3` | `14`                                                                  | Subsection header                                                           |
| `controlHeight`    | `32`                                                                  | All interactive controls (buttons, inputs, selects)                         |
| `controlHeightSM`  | `28`                                                                  | Compact variant (used inside tables)                                        |
| `motion`           | `false`                                                               | No transitions. See philosophy.md rule 6                                    |
| `fontFamily`       | `system-ui, -apple-system, "Segoe UI", Roboto, sans-serif`            | Default body                                                                |
| `fontFamilyCode`   | `"SF Mono", "JetBrains Mono", Consolas, "Liberation Mono", monospace` | IPs, MACs, UUIDs, paths, log lines                                          |
| `wireframe`        | `false`                                                               | Filled backgrounds (flat but not outline-only)                              |

### Spacing scale

antd uses `padding` and `margin` tokens derived from a base unit. We compress:

| Token        | Value | Use                                          |
| ------------ | ----- | -------------------------------------------- |
| `paddingXXS` | `2`   | Tightest gaps (inline row actions)           |
| `paddingXS`  | `4`   | Dense rows, tag padding                      |
| `paddingSM`  | `8`   | Standard component padding                   |
| `padding`    | `12`  | Default surface padding                      |
| `paddingLG`  | `16`  | Section separator (Rows `gutter={[16, 16]}`) |
| `paddingXL`  | `24`  | Page-level outer gutter                      |

### Component overrides

`components` inside ConfigProvider:

```tsx
Table: {
  cellPaddingBlockSM: 4,
  cellPaddingInlineSM: 8,
  fontSize: 13,
  headerBg: 'var(--color-surface-secondary)',
}

Tabs: {
  horizontalItemPaddingSM: '8px 12px',
  horizontalItemGutter: 16,
  titleFontSizeSM: 13,
}

Badge: {
  fontSize: 11,
  dotSize: 6,
}

Card: {
  boxShadow: 'none',              // flat — use border, not shadow
  boxShadowTertiary: 'none',
}

Layout: {
  headerBg: 'var(--color-surface-primary)',
  siderBg: 'var(--color-surface-secondary)',
}
```

## Color tokens

Semantic names. v0.1 values listed; v0.5+ dark palette is a token-swap (placeholder column documents intent).

| Token                     | Light (v0.1) | Dark (v0.5+, deferred) | Use                      |
| ------------------------- | ------------ | ---------------------- | ------------------------ |
| `color.text.primary`      | `#1f1f1f`    | (tbd)                  | Body text, table cells   |
| `color.text.secondary`    | `#595959`    | (tbd)                  | Help text, timestamps    |
| `color.text.tertiary`     | `#8c8c8c`    | (tbd)                  | Placeholder, disabled    |
| `color.text.inverse`      | `#ffffff`    | (tbd)                  | On primary button        |
| `color.surface.primary`   | `#ffffff`    | (tbd)                  | Main content background  |
| `color.surface.secondary` | `#fafafa`    | (tbd)                  | Sidebar, table header    |
| `color.surface.tertiary`  | `#f0f0f0`    | (tbd)                  | Inline code, field hover |
| `color.border.default`    | `#d9d9d9`    | (tbd)                  | Card / input border      |
| `color.border.strong`     | `#8c8c8c`    | (tbd)                  | Focused input            |
| `color.status.success`    | `#52c41a`    | `#73d13d`              | Running, success badge   |
| `color.status.warning`    | `#faad14`    | `#ffc53d`              | Degraded, idle warning   |
| `color.status.error`      | `#ff4d4f`    | `#ff7875`              | Stopped, failed, denied  |
| `color.status.info`       | `#1677ff`    | `#4096ff`              | Primary action, link     |
| `color.status.neutral`    | `#8c8c8c`    | `#bfbfbf`              | Pending, unknown         |

Severity-to-color mapping for log/audit pages matches `color.status.*`:

- Emergency / Alert / Critical / Error → `color.status.error`
- Warning → `color.status.warning`
- Notice / Info → `color.status.info`
- Debug → `color.status.neutral`

## Typography scale

| Usage               | Size   | Weight | Token reference      |
| ------------------- | ------ | ------ | -------------------- |
| Page title (H1)     | `20px` | 600    | `fontSizeHeading1`   |
| Section header (H2) | `16px` | 600    | `fontSizeHeading2`   |
| Subsection (H3)     | `14px` | 600    | `fontSizeHeading3`   |
| Body / table cell   | `13px` | 400    | `fontSize`           |
| Helper text         | `12px` | 400    | `fontSizeSM`         |
| Badge / caption     | `11px` | 400    | (component override) |
| Code / monospace    | `13px` | 400    | `fontFamilyCode`     |

## Z-index layers

| Token                   | Value  | Use                                       |
| ----------------------- | ------ | ----------------------------------------- |
| `zIndex.base`           | `0`    | Normal flow                               |
| `zIndex.dropdown`       | `1050` | `Select`, `Dropdown` menus (antd default) |
| `zIndex.sticky`         | `1020` | Sticky table header                       |
| `zIndex.drawer`         | `1060` | `Drawer` (task log, detail)               |
| `zIndex.modal`          | `1070` | `Modal`, `ModalForm`                      |
| `zIndex.notification`   | `1090` | `notification`, `message` toasts          |
| `zIndex.commandPalette` | `1100` | Cmd+K palette (above everything)          |

## Icon conventions

- Library: `lucide-react` (single source; never mix with antd icons except inside antd components like `Badge status`).
- Stroke width: `1.75` (antd default is `2`; slightly finer reads better at 13px body size).
- Size: `14` for inline row actions, `16` for toolbar buttons, `20` for empty-state illustrations, `12` for badges.
- Color: inherits from parent text color via `currentColor`; never hardcode.

## `<kbd>` — keyboard shortcut chips

Keyboard shortcuts appear in three places in the UI: button tooltips, dropdown menu items (`extra` slot), and the command palette. All three render shortcuts with `<kbd>` chips styled uniformly, generated from the shared `Kbd` component.

**Visual spec:**

| Property                        | Value                            | Notes                                                    |
| ------------------------------- | -------------------------------- | -------------------------------------------------------- |
| Font family                     | `fontFamilyCode`                 | Same monospace as IPs and paths                          |
| Font size                       | `11px`                           | Same as `Badge`                                          |
| Padding                         | `2px 5px`                        | Tight; chip is a shortcut hint, not a button             |
| Background                      | `color.surface.tertiary`         | Light fill — visible but not competing with text         |
| Border                          | `1px solid color.border.default` | One pixel, no radius above 4                             |
| Border radius                   | `borderRadius` (4)               | Matches buttons and inputs                               |
| Color                           | `color.text.secondary`           | Readable but not louder than the action label next to it |
| Margin between chips in a chord | `2px`                            | `g` + `s` renders as `[g] [s]` with a tight gap          |
| Separator for sequences         | U+00A0 (non-breaking space)      | Prevents awkward wraps between a label and its chord     |

**Rendering rules:**

- macOS: modifier glyphs (`⌘`, `⌥`, `⌃`, `⇧`) with no separator — `⌘K`, `⌘⇧P`
- Windows / Linux: word modifiers joined with `+` — `Ctrl+K`, `Ctrl+Shift+P`
- Chords (space-separated in the stored format): each keystroke as its own chip, rendered with a tight gap — `g` `s` for "go to storage"
- The `Kbd` component accepts a stored keystroke string (`"cmd-k"`, `"g s"`, `"cmd-shift-p"`) and handles platform formatting internally

**Where `<kbd>` chips appear:**

1. **Next to button labels in tooltips.** Every `<ActionButton>` wraps its label and chip in a tooltip — hover shows `"Create Schedule ⌘K"` or similar. The button surface itself stays clean.
2. **In dropdown / context menu items.** antd `Menu` `extra` slot renders the chip right-aligned — matches the affordance you see in macOS menus, VS Code, and Zed.
3. **In the command palette.** Each row shows the action label on the left and the current binding as chips on the right.
4. **In the `?` cheatsheet modal.** Action rows grouped by category; chip on the right.
5. **In Settings → Keyboard** (the keymap editor). Current and default bindings both rendered as chips for visual diff.

**Where chips do NOT appear:**

- On button surfaces themselves. This would compete with the label and add visual noise. The tooltip is where hints live.
- Inside modal titles or breadcrumbs.
- Inside table cells (except the keymap editor's own current-binding column).

```tsx
// web/src/components/Kbd.tsx
import { tokens } from "@/theme/tokens";

export function Kbd({ keystroke }: { keystroke: string }) {
  const parts = formatKeystrokeForPlatform(keystroke); // ["⌘", "K"] or ["Ctrl", "Shift", "P"]
  return (
    <span style={{ display: "inline-flex", gap: 2 }}>
      {parts.map((p, i) => (
        <kbd key={i} style={kbdStyle}>
          {p}
        </kbd>
      ))}
    </span>
  );
}
```

Token values live in `web/src/theme/tokens.ts` exported as a strongly-typed object. `web/src/theme/index.tsx` consumes this object and passes it to `ConfigProvider`. CSS variables (for consumption in `@uiw/react-codemirror` themes, `xterm.js` themes, custom SVGs) are emitted by a small generator in the same module.

```tsx
// web/src/theme/tokens.ts
export const tokens = {
  borderRadius: 4,
  fontSize: 13
  // ...
} as const;

// web/src/theme/index.tsx
import { tokens } from "./tokens";
export const HellingThemeProvider = ({ children }) => (
  <ConfigProvider theme={{ token: tokens, components: componentOverrides }}>
    {children}
  </ConfigProvider>
);
```

## Cross-references

- docs/design/philosophy.md (why these values — rule 1 information density, rule 6 no motion)
- docs/design/keyboard.md (Kbd chips are the visible surface of that doc's bindings)
- docs/spec/webui-spec.md (Theme section inlines the antd ConfigProvider snippet)
- docs/spec/accessibility.md (color contrast minimums for `color.text.*` over `color.surface.*`)
- docs/decisions/008-function-over-beauty.md
- docs/decisions/047-dark-mode-scope.md (why the Dark column is deferred)
