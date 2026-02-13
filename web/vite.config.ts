import { defineConfig, Plugin } from 'vite'
import { resolve } from 'path'
import { readFileSync, existsSync } from 'fs'

// Usage: BOARD_DIR=/path/to/.board npm run dev
const boardDir = process.env.BOARD_DIR || resolve(__dirname, '..', '.board')

function serveBoardFiles(): Plugin {
  return {
    name: 'serve-board-files',
    configureServer(server) {
      console.log(`Serving board files from: ${boardDir}`)
      server.middlewares.use((req, res, next) => {
        if (req.url?.startsWith('/.board/')) {
          const fileName = req.url.slice('/.board/'.length)
          const filePath = resolve(boardDir, fileName)
          if (existsSync(filePath)) {
            const ext = fileName.split('.').pop()?.toLowerCase()
            const mimeTypes: Record<string, string> = {
              json: 'application/json',
              png: 'image/png',
              jpg: 'image/jpeg',
              jpeg: 'image/jpeg',
              gif: 'image/gif',
              webp: 'image/webp',
              svg: 'image/svg+xml',
            }
            const contentType = mimeTypes[ext || ''] || 'application/octet-stream'
            const content = readFileSync(filePath)
            res.setHeader('Content-Type', contentType)
            res.end(content)
            return
          }
        }
        next()
      })
    }
  }
}

export default defineConfig({
  plugins: [serveBoardFiles()],
  build: {
    outDir: '../pkg/web/dist',
    emptyDirBeforeWrite: true
  },
  server: {
    watch: {
      usePolling: true
    },
    fs: {
      allow: ['..']
    }
  }
})
