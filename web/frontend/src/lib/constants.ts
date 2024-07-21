/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

// Web socket constants
export const WEB_SOCKET_URL = import.meta.env.VITE_WS_URL;

// API constants
export const API_URL = import.meta.env.VITE_API_URL as string;

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string;
// Theme constants
export const THEME_KEY = import.meta.env.VITE_THEME_KEY as string;

export const RECENT_SEARCH_LOCAL_KEY = import.meta.env
  .VITE_RECENT_SEARCH_LOCAL_KEY as string;

export const APP_DEFAULT_LOGO = import.meta.env.VITE_DEFAULT_LOGO_URL as string;
