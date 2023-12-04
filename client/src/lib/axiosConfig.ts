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

import { getCookie } from "@/lib/auth";
import { API_URL } from "@/lib/constants";
import axios from "axios";

/**
 * Axios request interceptor.
 * It sets the base URL and credentials of the request.
 * It also logs the request details to the console.
 */
axios.interceptors.request.use(
  (req) => {
    req.baseURL = API_URL;
    req.withCredentials = true;

    // Set CSRF Token
    const csrfToken = getCookie("csrftoken");

    if (csrfToken) {
      req.headers["X-CSRFToken"] = csrfToken;
    }

    console.log(
      `%c[MONTA] Axios request: ${req.method?.toUpperCase()} ${req.url}`,
      "color: #34ebe5; font-weight: bold",
    );
    return req;
  },
  (error: any) => Promise.reject(error),
);

export default axios;
