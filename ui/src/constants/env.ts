/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// API constants
export const API_URL = import.meta.env.VITE_API_URL as string;
export const WEBSOCKET_URL = import.meta.env.VITE_WEBSOCKET_URL as string;

// Client constants
export const CLIENT_VERSION = import.meta.env.VITE_CLIENT_VERSION as string;
export const CLIENT_NAME = import.meta.env.VITE_CLIENT_NAME as string;
export const AUTHOR_NAME = import.meta.env.VITE_AUTHOR_NAME as string;
export const AUTHOR_EMAIL = import.meta.env.VITE_AUTHOR_EMAIL as string;
export const APP_ENV = import.meta.env.VITE_APP_ENV as string;
// Dialog constants
export const MOVE_DELETE_DIALOG_KEY = import.meta.env
  .VITE_MOVE_DELETE_DIALOG_KEY as string;
export const COMMODITY_DELETE_DIALOG_KEY = import.meta.env
  .VITE_COMMODITY_DELETE_DIALOG_KEY as string;
export const HAZARDOUS_MATERIAL_NOTICE_KEY = import.meta.env
  .VITE_HAZARDOUS_MATERIAL_NOTICE_KEY as string;
export const GOOGLE_MAPS_NOTICE_KEY = import.meta.env
  .VITE_GOOGLE_MAPS_NOTICE_KEY as string;
export const HAZMAT_SEGREGATION_RULE_NOTICE_KEY = import.meta.env
  .VITE_HAZMAT_SEGREGATION_RULE_NOTICE_KEY as string;
export const USER_CREATE_NOTICE_KEY = import.meta.env
  .VITE_USER_CREATE_NOTICE_KEY as string;
export const STOP_DIALOG_NOTICE_KEY = import.meta.env
  .VITE_STOP_DIALOG_NOTICE_KEY as string;
export const ADDITIONAL_CHARGE_DELETE_DIALOG_KEY = import.meta.env
  .VITE_ADDITIONAL_CHARGE_DELETE_DIALOG_KEY as string;

export const SITE_SEARCH_RECENT_SEARCHES_KEY = import.meta.env
  .VITE_SITE_SEARCH_RECENT_SEARCHES_KEY as string;

export const PDF_STORAGE_KEY = import.meta.env.VITE_PDF_STORAGE_KEY as string;

export const DEBUG_TABLE = import.meta.env.VITE_DEBUG_TABLE as boolean;

export const TURNSTILE_SITE_KEY = import.meta.env
  .VITE_TURNSTILE_SITE_KEY as string;
export const SHOW_FAVORITES_KEY = import.meta.env
  .VITE_SHOW_FAVORITES_KEY as string;

export const SENTRY_DSN = import.meta.env.VITE_SENTRY_DSN as string;

export const ENABLE_SENTRY = import.meta.env.VITE_ENABLE_SENTRY as boolean;
