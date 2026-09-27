import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

// PWA, bukan Flutter web: deliverable wajib hackathon adalah prototype
// yang bisa diakses lewat URL. Installable ke home screen supaya tetap
// terasa seperti app mobile saat rekaman demo.
export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      // Worklet `.js` sudah masuk glob precache bawaan; hanya aset SVG yang
      // perlu ditambahkan agar tidak menghasilkan entri cache duplikat.
      includeAssets: ['favicon.svg', 'bisik-icon.svg'],
      manifest: {
        name: 'Bisik — Kopilot Kepatuhan',
        short_name: 'Bisik',
        description:
          'Kopilot privat yang mengingatkan petugas keuangan Indonesia sebelum kewajiban terlewat.',
        theme_color: '#0b1211',
        background_color: '#0b1211',
        lang: 'id',
        display: 'standalone',
        orientation: 'portrait',
        start_url: '/officer',
        icons: [
          {
            src: 'bisik-icon.svg',
            sizes: 'any',
            type: 'image/svg+xml',
            purpose: 'any maskable',
          },
        ],
      },
    }),
  ],
  server: { port: 5173 },
})
