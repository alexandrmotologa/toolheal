import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:7171',
      '/v1': 'http://127.0.0.1:7171',
      '/rpc': 'http://127.0.0.1:7171',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
