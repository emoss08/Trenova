/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

// User info constants
export const USER_ID_KEY = import.meta.env.VITE_USER_ID_KEY;
export const ORGANIZATION_ID_KEY = import.meta.env.VITE_ORGANIZATION_ID_KEY;

// Web socket constants
export const WEB_SOCKET_URL = import.meta.env.VITE_WS_URL;
export const MAX_WEBSOCKET_RETRIES = import.meta.env.VITE_MAX_WEBSOCKET_RETRIES;
export const WEBSOCKET_RETRY_INTERVAL = import.meta.env
  .VITE_WEBSOCKET_RETRY_INTERVAL;
export const ENABLE_WEBSOCKETS = import.meta.env.VITE_ENABLE_WEBSOCKETS;

export const getUserCSRFToken = () => {
  const csrfCookie = document.cookie
    .split("; ")
    .find((row) => row.startsWith("csrftoken="));

  if (csrfCookie) {
    return csrfCookie.split("=")[1];
  } else {
    console.log("No CSRF token found");
  }
};

export const getSessionItem = (key: string) => {
  const item = sessionStorage.getItem(key);
  return item ? JSON.parse(item) : null;
};

/**
 * Returns the user's id from Session Storage.
 *
 * @returns {string | null} The user's id.
 *
 * @example
 * const userId = getUserId();
 * if (userId) {
 * // Do something with the userId
 * console.log(userId);
 * }
 */
export const getUserId = (): string | null => {
  const userId = sessionStorage.getItem(USER_ID_KEY);
  if (userId) {
    return userId;
  }
  return null;
};

/**
 * Returns the user's organization id from local storage.
 *
 * @returns {string | null} The user's organization id.
 *
 * @example
 * const organizationId = getUserOrganizationId();
 * if (organizationId) {
 * // Do something with the organizationId
 * console.log(organizationId);
 * }
 */
export const getUserOrganizationId = (): string | null => {
  const userOrganization = sessionStorage.getItem(ORGANIZATION_ID_KEY);
  if (userOrganization) {
    return userOrganization;
  }
  return null;
};
