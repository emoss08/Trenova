/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string;
  readonly VITE_CSRF_HEADER_NAME?: string;
  /** Brandfetch Logo CDN client id. Public by design; the API key is never used in the browser. */
  readonly VITE_BRANDFETCH_CLIENT_ID?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
