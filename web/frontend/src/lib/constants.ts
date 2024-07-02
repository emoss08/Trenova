// Web socket constants
export const WEB_SOCKET_URL = import.meta.env.VITE_WS_URL;
export const ENABLE_WEBSOCKETS = import.meta.env
  .VITE_ENABLE_WEBSOCKETS as boolean;

// API constants
export const API_URL = import.meta.env.VITE_API_URL as string;

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string;
// Theme constants
export const THEME_KEY = import.meta.env.VITE_THEME_KEY as string;

// Environment constant
export const ENVIRONMENT = import.meta.env.VITE_ENVIRONMENT as string;

export const DEBOUNCE_DELAY = 500; // debounce delay in ms

export const TOAST_STYLE = {
  background: "hsl(var(--background))",
  color: "hsl(var(--foreground))",
  boxShadow: "0 0 0 1px hsl(var(--border))",
};
