import { defineConfig } from "vite"
import FullReload from "vite-plugin-full-reload"

const proxyErrorHtml = `<!DOCTYPE html>
<html>
<head>
  <title>Proxy error</title>
  <script>
    setTimeout(() => { location.reload() }, 100)
  </script>
</head>
<body>
  Backend server not accepting connections. Retrying...
</body>
</html>`

export default defineConfig({
  root: "src",
  base: "/assets",
  server: {
    port: 8080,
    strictPort: true,
    proxy: {
      "^/(?!(assets))" : {
        target: "http://localhost:8081",
        configure: (proxy) => {
          proxy.on("error", (err, req, res) => {
            if (err.code === "ECONNREFUSED") {
              res.writeHead(502, {
                "Content-Type": "text/html"
              })

              res.end(proxyErrorHtml)
            }
          })
        }
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
    FullReload("../bin/web-starter-app")
  ]
})
