function resolveApiBaseUrl(): string {
  const configuredUrl = (import.meta.env.VITE_API_URL as string) || "/api/v1";

  if (
    import.meta.env.DEV &&
    typeof window !== "undefined" &&
    configuredUrl.startsWith("http://localhost:8080/")
  ) {
    return "/api/v1";
  }

  return configuredUrl;
}

export const API_BASE_URL = resolveApiBaseUrl();

export const APP_ENV = (import.meta.env.MODE as string) || "development";
export const TERMS_URL =
  (import.meta.env.VITE_TERMS_URL as string | undefined) ??
  "https://trenova.app/legal/terms/";
export const PRIVACY_URL =
  (import.meta.env.VITE_PRIVACY_URL as string | undefined) ??
  "https://trenova.app/legal/privacy/";

export const US_CENTER = { lat: 39.8, lng: -98.5 };
export const DEFAULT_ZOOM = 4;
export const MAP_ID_LIGHT = import.meta.env.VITE_GOOGLE_MAPS_ID_LIGHT as string;
export const MAP_ID_DARK = import.meta.env.VITE_GOOGLE_MAPS_ID_DARK as string;

export const GOOGLE_MAPS_ERROR_MESSAGE = "GoogleMaps integration is not configured";
