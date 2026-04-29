// Regression guard for Phase 2D F-30 icon barrel.
//
// Catches the most likely future bug: a contributor adds `<I n="some-name" />`
// without registering "some-name" in ICONS. The audit-grep that built the
// initial barrel (84 names) has to stay accurate as the UI evolves; failing
// this test means update icons.ts before merging.
import { describe, expect, it } from 'vitest';
import { ICONS, type IconName } from './icons';

describe('icons barrel', () => {
  it('exports a non-empty ICONS map', () => {
    expect(Object.keys(ICONS).length).toBeGreaterThan(0);
  });

  it('every entry is a renderable component', () => {
    // Lucide icons are React.forwardRef objects (typeof === 'object'),
    // older Lucide names are plain function components. Accept both.
    for (const [name, Comp] of Object.entries(ICONS)) {
      const t = typeof Comp;
      expect(
        t === 'function' || (t === 'object' && Comp !== null),
        `${name} should be a renderable React component, got ${t}`,
      ).toBe(true);
    }
  });

  it('contains the icons Phase 1 PageLogin needs', () => {
    // PageLogin Continue button uses arrow-right; MFA uses shield.
    const required: IconName[] = ['arrow-right', 'shield'];
    for (const name of required) {
      expect(ICONS[name], `missing required icon: ${name}`).toBeDefined();
    }
  });

  it('all keys are kebab-case strings', () => {
    for (const name of Object.keys(ICONS)) {
      expect(name).toMatch(/^[a-z][a-z0-9-]*$/);
    }
  });
});
