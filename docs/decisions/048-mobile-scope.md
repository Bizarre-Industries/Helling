# ADR-048: Mobile and Responsive Scope for v0.1

> Status: Accepted (2026-04-21)

## Context

`docs/design/philosophy.md` Rule 8 ("Responsive ≠ Mobile-First") already defines the responsive layout contract across three breakpoints. This ADR formalizes which breakpoints are QA-gated for v0.1 and what "responsive" means per form factor — so scope isn't ambiguous when shipping.

## Decision

**v0.1 ships responsive per philosophy.md Rule 8**, with asymmetric QA coverage:

| Form factor | Breakpoint | Layout                               | v0.1 QA gate               |
| ----------- | ---------- | ------------------------------------ | -------------------------- |
| Desktop     | ≥1280px    | Full three-panel, maximum density    | ✅ Full manual + automated |
| Tablet      | 768-1279px | Tree collapses to icons, two-panel   | ✅ Manual + Axe CI         |
| Phone       | <768px     | Single-panel, drawer nav, large taps | 🟡 Best-effort, no gate    |

Full functionality is available at all three breakpoints — VM create wizards, console access, backup flows, etc. Rule 8 calls out that phone is for "emergency use from a phone" — not a second-class read-only view.

## Rationale

- Rule 8 is already in philosophy.md as a binding rule; this ADR formalizes it so the v0.1 release gate isn't ambiguous
- antd ProLayout handles most responsive behavior out of the box (side panel collapse, responsive Pro components). The v0.1 delta is mostly "don't break it" rather than "build it"
- Operators check on alerts from phones; restricting phone to read-only would cover that use case but would require an entirely separate UI path (anti-ADR-008)
- Desktop ≥1280 is the 95% use case and gets the strictest QA
- Phone <768 is <5% use. Best-effort means "if reported, fix it; don't gate release on exhaustive phone testing"

## Acceptance criteria for v0.1

- Desktop layout (1920×1080) passes full manual QA + automated visual regression across all pages
- Tablet layout (1024×768) passes manual QA on all critical paths (VM create, console, backup, auth, cluster view) and automated Axe accessibility checks on dashboard + alerts page
- Phone layout (375×667 representative) renders without overflow and without broken navigation on dashboard, alerts, and VM detail pages. No full QA matrix.
- No mobile-specific code paths or runtime detection — CSS-only responsive via antd breakpoints and Tailwind-style utility classes

## Deferred to post-v0.1

- Dedicated phone-optimized workflows (one-handed operation, swipe gestures, pull-to-refresh)
- Mobile PWA install prompt
- Explicit iOS/Android testing devices in CI
- Offline mode / service worker

## Consequences

**Easier:**

- Clear single layout system (Rule 8 breakpoints) rather than divergent desktop-only and mobile-only codebases
- antd ProLayout provides most of the responsive behavior free
- Phone best-effort policy keeps v0.1 release cadence from getting blocked on phone QA

**Harder:**

- Phone edge cases may surface post-release — Rule 8 sets expectations correctly (emergency use) but users will still report issues
- Tablet QA cost is real and not optional — approximately +25% to UI QA time per release

## References

- `docs/design/philosophy.md` Rule 8 (normative breakpoint contract)
- `docs/spec/accessibility.md` (WCAG 2.1 AA targets)
- ADR-008 (function over beauty)
- ADR-047 (dark mode — same deferred-feature pattern)
