export const LOCAL_DEV_API_BASE_URL = "http://localhost:8080/api/v1";

function resolveApiBaseUrl(): string {
  const configuredUrl = import.meta.env.VITE_API_URL as string | undefined;
  if (configuredUrl) {
    return configuredUrl;
  }

  if (import.meta.env.MODE === "development") {
    return LOCAL_DEV_API_BASE_URL;
  }

  return "/api/v1";
}

export const API_BASE_URL = resolveApiBaseUrl();

export const APP_ENV = (import.meta.env.MODE as string) || "development";

/** A build-time setting, trimmed; empty when the build does not set it. */
export function envSetting(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function envUrl(value: unknown): string {
  const url = envSetting(value);
  return /^https?:\/\//i.test(url) ? url : "";
}

/**
 * The build's own terms and privacy addresses, used when the server's public config
 * names none. Empty when unset: a self-hosted install publishes its own documents or
 * none, and a link must never point at somebody else's.
 */
export const TERMS_URL = envUrl(import.meta.env.VITE_TERMS_URL);
export const PRIVACY_URL = envUrl(import.meta.env.VITE_PRIVACY_URL);

/** Where people are told to write when something is broken. Empty when the build names none. */
export const SUPPORT_EMAIL = envSetting(import.meta.env.VITE_SUPPORT_EMAIL);

export const US_CENTER = { lat: 39.8, lng: -98.5 };
export const DEFAULT_ZOOM = 4;
export const MAP_ID_LIGHT = import.meta.env.VITE_GOOGLE_MAPS_ID_LIGHT as string;
export const MAP_ID_DARK = import.meta.env.VITE_GOOGLE_MAPS_ID_DARK as string;

