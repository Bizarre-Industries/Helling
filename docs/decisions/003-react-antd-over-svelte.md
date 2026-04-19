# ADR-003: React + Ant Design Pro + refine over SvelteKit + DaisyUI

> Status: Accepted (updated 2026-04-15)

## Context

Initial dashboard was built with SvelteKit + TailwindCSS + DaisyUI. DaisyUI is consumer-app styled and fights admin UI density. Custom tables required ~200 lines each with TanStack Table + shadcn. Every page was monolithic with zero reusable components.

## Decision

Rewrite dashboard using React 19 + Vite + Ant Design Pro (ProTable, ProForm, StepsForm, ProLayout, Descriptions) + `@refinedev/core` + `@refinedev/antd`. No SSR — dashboard is behind login, no SEO needed.

`orval` generates React Query hooks and TypeScript types from the Helling OpenAPI spec (~25 Helling-owned endpoints). Proxied routes (Incus, Podman, Cloud Hypervisor) return native upstream response formats; the frontend consumes those directly via generated hooks or lightweight wrappers.

## Consequences

- ProTable: sort/filter/paginate/bulk-select in ~30 lines vs ~200
- StepsForm: multi-step wizards with validation for free
- Descriptions: key-value PropertyGrid for free
- refine CRUD framework eliminates boilerplate list/create/edit flows for all Helling-owned resources
- orval code generation covers ~25 Helling-owned endpoints (auth, users, schedules, webhooks, BMC, K8s, system, firewall); proxied routes are consumed via refine data provider or direct hooks
- Estimated 2x code reduction (8-12K lines vs 18-22K)
- Trade-off: larger bundle (~500KB gzipped vs ~86KB)
- React more verbose than Svelte for simple components
- Massive component ecosystem reduces custom code
- Net win for a ~17-page admin dashboard
