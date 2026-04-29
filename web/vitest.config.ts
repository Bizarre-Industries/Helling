import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// Audit F-40: vitest scaffold. Three smoke tests guard Phase 1 work.
// Real coverage expansion follows Phase 2A monolith split.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    css: false,
  },
});
