import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.API_PORT ?? '8080'}`,
        changeOrigin: false,
      },
    },
  },
  build: {
    outDir: 'dist',
  },
})
