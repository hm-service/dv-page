import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

// 开发时前端跑 Vite，接口仍由 Go 提供：`just run` 起后端，`pnpm dev` 起前端。
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
})
