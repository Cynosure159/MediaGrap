import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";
import { sourceInventory } from "./source-inventory";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    vue(),
    sourceInventory(),
    VitePWA({
      registerType: "autoUpdate",
      manifest: {
        name: "MediaGrap",
        short_name: "MediaGrap",
        description: "Self-hosted media metadata manager",
        theme_color: "#0c1324",
        background_color: "#0c1324",
        display: "standalone",
        start_url: "/movies",
        icons: [
          {
            src: "pwa-192.svg",
            sizes: "192x192",
            type: "image/svg+xml",
            purpose: "any",
          },
          {
            src: "pwa-512.svg",
            sizes: "512x512",
            type: "image/svg+xml",
            purpose: "any",
          },
        ],
      },
      workbox: {
        navigateFallback: "/index.html",
        navigateFallbackDenylist: [/^\/source(?:[/?]|$)/],
        globPatterns: ["**/*.{js,css,html,svg,ico,png}"],
      },
    }),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    proxy: {
      "/source": "http://127.0.0.1:8080",
      "/api": "http://127.0.0.1:8080",
      "/healthz": "http://127.0.0.1:8080",
      "/readyz": "http://127.0.0.1:8080",
    },
  },
  test: {
    environment: "jsdom",
  },
});
