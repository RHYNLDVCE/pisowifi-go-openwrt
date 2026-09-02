import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: 'auto',
      useCredentials: true,
      manifest: {
        name: 'PisoWifi Admin',
        short_name: 'PisoAdmin',
        description: 'PisoWifi Management Dashboard',
        theme_color: '#000000',
        background_color: '#ffffff',
        display: 'standalone',
        icons: [
          {
            src: 'pwa-192x192.png',
            sizes: '192x192',
            type: 'image/png'
          },
          {
            src: 'pwa-512x512.png',
            sizes: '512x512',
            type: 'image/png'
          }
        ]
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff,woff2}']
      }
    }),
  ],
  base: '/admin/',
  build: {
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      treeshake: {
        moduleSideEffects: (id, external) => {
          if (id.includes('registry.js') || id.includes('.bones.json') || id.includes('boneyard-js')) {
            return true;
          }
          return true;
        }
      },
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/recharts') || id.includes('d3-') || id.includes('victory-vendor')) {
            return 'recharts';
          }
          if (id.includes('node_modules/lucide-react')) {
            return 'lucide';
          }
          if (id.includes('node_modules')) {
            return 'vendor';
          }
        }
      }
    }
  }
})
