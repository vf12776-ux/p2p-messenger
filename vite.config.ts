import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.ico', 'apple-touch-icon.png', 'masked-icon.svg'],
      manifest: {
        name: 'P2P Messenger',
        short_name: 'P2P',
        description: 'Messaging and file sharing app',
        theme_color: '#ffffff',
        icons: [
          {
            src: 'icon-45.png',
            sizes: '192x192',
            type: 'image/png',
          },
          {
            src: 'icon-54.png',
            sizes: '512x512',
            type: 'image/png',
          },
        ],
      },
    }),
  ],
})