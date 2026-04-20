# Accessibility Specification

Accessibility requirements for Helling WebUI.

## Commitment

Helling targets WCAG 2.1 AA for v1.0 and applies these rules for v0.x work by default.

## Core Requirements

- Full keyboard navigation for interactive flows.
- Visible focus indicators.
- Semantic HTML and ARIA where needed.
- Adequate color contrast for text and controls.
- Screen reader-friendly labels and announcements for dynamic content.

## Interaction Rules

- Every actionable UI element must be reachable and operable via keyboard.
- Modal dialogs must trap focus and restore focus on close.
- Async status updates should be announced via polite live regions when appropriate.

## Validation

- Run automated accessibility checks in CI where configured.
- Include manual keyboard and screen-reader smoke checks for major feature pages.
