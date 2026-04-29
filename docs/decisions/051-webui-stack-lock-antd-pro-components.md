# ADR-051: WebUI stack locked to antd 6 + pro-components per webui-spec

> Status: Accepted (2026-04-27)

## Context

`docs/spec/webui-spec.md` §Stack (L47–L51) commits the WebUI to:

```text
antd 6                          → Core components
@ant-design/pro-components      → ProTable, ProForm, StepsForm, ProLayout, Descriptions
@ant-design/charts              → Charts (G2-based, antd theme integrated)
@tanstack/react-query           → Data fetching (via hey-api/openapi-ts generated hooks/options for Helling API)
```

`docs/design/tokens.md` L10 is explicit: _"antd first. Helling rides antd's theme token system. We override specific tokens; we don't build a parallel system."_

`docs/design/philosophy.md` inlines a `<ConfigProvider>` snippet as the canonical theme entry point.

**The implementation in `web/` walked away from this.** Audit `docs/audits/webui-2026-04-27.md` finding F-36 documents the divergence:

- `web/package.json` declares `lucide-react`, `@tanstack/react-query`, `@hey-api/client-fetch`, `react`, `react-dom`. **Zero antd-family dependencies.**
- `web/src/theme/` does not exist.
- `web/src/styles/app.css` defines a parallel CSS-custom-property token system — exactly what `tokens.md` forbids.
- Tables are `<div>` grids (F-26). Modals are hand-rolled with no a11y (F-25). Wizard is hand-rolled with no validation (F-14). Detail layouts hand-rolled (F-23).

The audit calls this out as the most expensive bug in the repo. Five other audit findings (F-06, F-14, F-23, F-25, F-26) are downstream of it: they prescribe extracting primitives that pro-components already ships.

The team has two paths. Either:

- **(a)** Commit to the spec. Port the WebUI onto antd 6 + pro-components. Land tokens via `<ConfigProvider>`.
- **(b)** Commit to the current hand-rolled stack. Update `docs/spec/webui-spec.md`, `docs/design/tokens.md`, `docs/design/philosophy.md` to remove antd references and document the hand-rolled foundation as canonical.

Per the project's standing memory rule (`feedback_docs_source_of_truth`: _"Always reference docs before implementing, never guess"_) and `feedback_plan_approach` (_"Docs first, ask before assuming, think beyond API layer"_), the spec wins by default unless the team makes an explicit informed decision to abandon it. No such decision has been made. The hand-rolled implementation is drift, not a decision.

Pro-components also brings concrete wins the audit catalogs:

- **F-25 (modal a11y)** → antd `Modal` ships focus trap, restore, `role="dialog"`, `aria-modal`, `aria-labelledby` natively.
- **F-26 (table semantics)** → ProTable ships real `<table>` markup with `aria-sort`, sort/filter/pagination/density/column-toggle for free.
- **F-14 (wizard validation)** → StepsForm ships per-step validation, gating, preview hooks.
- **F-23 (page header drift)** → ProLayout ships `PageContainer` with consistent header + breadcrumb + tabs.
- **F-06 (inline styles)** → ConfigProvider tokens collapse the 783 inline-style objects + 87 hardcoded color literals into theme tokens.
- **F-44 (prefers-reduced-motion)** → antd respects this natively.

The accessibility spec (`docs/spec/accessibility.md`) commits to WCAG 2.1 AA. The current hand-rolled foundation cannot meet that bar without rebuilding most of what antd already ships.

## Decision

**The WebUI commits to the spec'd stack: antd 6 + @ant-design/pro-components + @ant-design/charts + @tanstack/react-query.**

Path forward:

1. **Phase 0 (this ADR + Phase 1 in parallel).** Lock the decision. Audit doc captured. ADR accepted. No new pages added on the hand-rolled stack — every new page goes onto pro-components from day one.
2. **Phase 1 (parallel).** Ship stack-independent safety fixes: F-37 (auth-store memory-only), F-38 (wire login), F-39 (ErrorBoundary), R-03 (viewport meta), F-15 (confirmMatch), F-41 (fresh-clone build), F-44 (prefers-reduced-motion), F-45 (refetchOnWindowFocus default), F-50 (density persistence). These ship on the current code without depending on the migration outcome.
3. **Phase 2 (untangle foundation).** Split `pages.jsx` + `pages2.jsx` into per-route folders, drop `window.*` coupling, lazy-load per route, icon barrel, scaffold tests. Pure mechanical wins; reduces blast radius of Phase 3.
4. **Phase 3 (migration spike).** Port three representative pages to antd + pro-components: Instances list (ProTable), Instance Detail (ProDescriptions + Tabs), New Instance wizard (StepsForm). Validate the token bridge (`web/src/theme/tokens.ts` + `<ConfigProvider>`). Confirm bundle sizes are acceptable.
5. **Phase 4 (port).** Roll the remaining pages onto pro-components in batches grouped by section (Datacenter → Resources → Observability → Admin). Remove the parallel CSS token system from `app.css` once all consumers are migrated.

Add deps:

```bash
bun add antd@^6 @ant-design/pro-components @ant-design/charts dayjs
```

Token bridge file (introduced in Phase 3):

```ts
// web/src/theme/tokens.ts
import type { ThemeConfig } from "antd";
export const helling: ThemeConfig = {
  token: {
    colorPrimary: "var(--bzr-lime)",
    colorBgLayout: "var(--bzr-void)",
    colorBgContainer: "var(--bzr-ash)",
    fontFamily: "Hanken Grotesk, system-ui, sans-serif",
    fontFamilyCode: "JetBrains Mono, ui-monospace, monospace",
    borderRadius: 4,
    fontSize: 13
  },
  components: {
    Table: { rowHoverBg: "rgba(255,255,255,0.04)", cellPaddingBlock: 8 }
  }
};
```

App entry:

```tsx
// web/src/main.tsx
import { ConfigProvider } from "antd";
import { helling } from "./theme/tokens";
// ...
<ConfigProvider theme={helling}>{/* App */}</ConfigProvider>;
```

## Consequences

### Easier

- Modal a11y, table a11y, wizard validation, focus management, focus restore — all free from the framework.
- WCAG 2.1 AA compliance becomes achievable. Currently, it is not.
- Bundle: pro-components is tree-shakeable; the icon-barrel work in F-30 still applies for lucide leftovers, but most icons come via antd-icons.
- Light mode (F-20) becomes a `ConfigProvider` `algorithm: theme.defaultAlgorithm` swap. No more hand-flipped CSS variables.
- Token discipline (`tokens.md`) becomes enforceable: any color outside `helling` config is a lint target.
- Future contributors find a stack the spec describes, with a 10k+ component reference (antd docs).

### Harder

- ~3-4 sprint port. Phases 3 + 4 are the bulk of the cost.
- Every existing page will be touched. Diffs in Phases 3/4 will be large by file count, mostly mechanical.
- Bundle size of antd + pro-components is non-trivial. Mitigated by per-route lazy-loading (F-29) and tree-shaking. Initial budget: keep first-load JS under 400KB gzipped.
- Some custom shell affordances (Cmd-K palette, footer task drawer, bell inbox) stay hand-rolled — they're product-specific and have no pro-components equivalent. They keep their existing implementations.
- The Bizarre DS visual signature (stencil headings, lime accent, mono eyebrows, dot indicators, 1px borders, 4px radius) must be preserved through token configuration. Phase 3 spike validates this is achievable; if it isn't, ADR is reopened.

### Risk if not done

- Every WebUI PR compounds the divergence.
- Audit findings F-06, F-14, F-23, F-25, F-26 get fixed against the wrong target — wasted sprint when migration eventually happens.
- Spec ↔ code drift erodes the "docs as source of truth" rule that the rest of the project relies on. If the WebUI is allowed to walk away from its spec, every other spec becomes negotiable.

### Alternatives considered

- **Path (b): commit to the hand-rolled stack.** Rejected. Would require rewriting a11y, focus management, table semantics, wizard validation, theme system from scratch — re-implementing what pro-components ships, paid in salary instead of dependency cost. Also requires editing three normative spec docs to match a non-decision.
- **Stay on antd 5.** Rejected. antd 6 is GA, ships the React 19 compatibility layer, and `webui-spec.md` already names version 6.
- **Mantine, Chakra, shadcn.** Rejected. Spec names antd. Switching DS would require its own ADR with stronger justification than "we like it more"; no such case has been made. ProTable + StepsForm + ProLayout have no equivalent in shadcn.

## References

- `docs/spec/webui-spec.md` §Stack
- `docs/design/tokens.md` L10–L17
- `docs/design/philosophy.md` (ConfigProvider example)
- `docs/spec/accessibility.md` (WCAG 2.1 AA commitment)
- `docs/audits/webui-2026-04-27.md` finding F-36
- ADR-044 (hey-api WebUI codegen — unaffected; data fetching path stays the same)
- ADR-047 (dark mode scope — light mode now realizable via ConfigProvider algorithm swap)
- ADR-049 (vim mode and keymap surfacing — keymap layer stays hand-rolled, sits on top of antd)
