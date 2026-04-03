/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL_PREFIX?: string
  readonly VITE_WS_URL_PREFIX?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// BUILD_TIMESTAMP is injected at build time via vite.config.ts define, not a VITE_ env var
declare namespace NodeJS {
  interface ProcessEnv {
    readonly BUILD_TIMESTAMP: string
  }
}
