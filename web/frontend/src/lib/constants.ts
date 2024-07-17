/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

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
