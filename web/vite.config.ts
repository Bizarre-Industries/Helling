import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      // Dev-time: forward /api/* to the local hellingd daemon so the
      // generated hey-api SDK can be configured with same-origin baseUrl.
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    target: 'es2022',
    modulePreload: {
      resolveDependencies(_filename, deps) {
        return deps.filter((dep) => !dep.includes('scalar-'));
      },
    },
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/node_modules/@scalar/')) {
            const match = id.match(/\/node_modules\/@scalar\/([^/]+)/);
            return match ? `scalar-${match[1]}` : 'scalar';
          }
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
});
