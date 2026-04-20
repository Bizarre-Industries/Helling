# Design Tokens

Canonical design-token guidance for Helling WebUI.

## Token Categories

- Color tokens (surface, text, border, semantic states)
- Spacing scale
- Typography scale
- Radius and shadow tokens
- Z-index layers

## Rules

- Use tokenized values instead of ad-hoc literal values.
- Keep semantic naming (`color.text.primary`) over raw color naming.
- Avoid introducing one-off tokens without design review.

## Implementation

Token definitions should map cleanly to the chosen UI system and theme layer, and remain consistent across pages/components.
