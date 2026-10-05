import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In development the Go backend runs on :8080. Proxying keeps the browser
// on a single origin so no CORS setup is needed locally.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
  build: { chunkSizeWarningLimit: 900 },
});
