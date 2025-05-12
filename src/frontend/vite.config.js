import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // Pengaturan dev server
    proxy: {
      // Proxy API requests untuk development lokal
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      }
    }
  },
})