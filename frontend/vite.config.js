import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
export default defineConfig({
    appType: 'spa',
    plugins: [vue()],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url)),
        },
        extensions: ['.mjs', '.mts', '.ts', '.tsx', '.vue', '.js', '.jsx', '.json'],
    },
    server: {
        host: '0.0.0.0',
        port: 12000,
        strictPort: true,
        proxy: {
            '/api': {
                target: process.env.VITE_API_PROXY || 'http://127.0.0.1:12080',
                changeOrigin: true,
            },
        },
    },
    preview: { host: '0.0.0.0', port: 12000, strictPort: true },
});
