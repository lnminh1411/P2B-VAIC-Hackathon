import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react()],
  server: { port: 5173 },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/@supabase/')) return 'supabase-vendor'
          if (id.includes('/@tanstack/')) return 'query-vendor'
          if (id.includes('/motion/') || id.includes('/lucide-react/')) return 'ui-vendor'
          if (id.includes('/react/') || id.includes('/react-dom/')) return 'react-vendor'
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test-setup.ts',
  },
})
