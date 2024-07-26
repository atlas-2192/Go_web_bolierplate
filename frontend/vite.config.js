import { defineConfig } from "vite"
import FullReload from "vite-plugin-full-reload"

export default defineConfig({
  root: "src",
  base: "/assets",
  server: {
    port: 8080,
    strictPort: true,
    proxy: {
      "^/(?!(assets))" : {
        target: "http://localhost:8081"
      }
    },
  },
  build: {
    // generate .vite/manifest.json in outDir
    manifest: true,
    rollupOptions: {
      // overwrite default .html entry
      input: "/js/main.js",
    },
  },
  plugins: [
    // Delay 100ms to allow the server to restart before triggering the page reload. Adjust as needed.
    FullReload("../bin/web-starter-app", {delay: 250})
  ]
})
