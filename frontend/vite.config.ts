import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { compression } from 'vite-plugin-compression2'
import { fileURLToPath, URL } from 'node:url'

// The Go server proxies nothing back to Vite; instead Vite proxies the
// wakapi backend during development. Port comes from wakapi.yml (server.port).
const BACKEND = process.env.WAKAPI_DEV_BACKEND ?? 'http://localhost:3001'

export default defineConfig({
  // Relative base is mandatory: the shell injects `<base href="{basePath}/assets/app/">`
  // so the very same bundle works under any `server.base_path`.
  base: './',
  plugins: [
    react(),
    tailwindcss(),
    compression({
      algorithms: ['brotliCompress'],
      // index.html is rewritten per request by the Go handler, so a pre-compressed
      // copy would be served stale by gzipped.FileServer.
      exclude: [/\.html$/],
      deleteOriginalAssets: false,
    }),
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    outDir: '../static/assets/app',
    assetsDir: 'static',
    emptyOutDir: true,
    target: 'es2022',
    sourcemap: false,
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: Object.fromEntries(
      ['/api', '/assets', '/login', '/logout', '/signup', '/oidc', '/subscription', '/imprint', '/setup', '/webauthn']
        .map((p) => [p, { target: BACKEND, changeOrigin: false }]),
    ),
  },
})
