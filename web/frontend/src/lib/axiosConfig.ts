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

import { API_URL } from "@/lib/constants";
import axios from "axios";
import { v4 as uuidv4 } from "uuid";

/**
 * Function to generate Idempotency Key
 * @returns {string}
 */
export function generateIdempotencyKey(): string {
  return uuidv4();
}

/**
 * Axios request interceptor.
 * It sets the base URL and credentials of the request.
 * It also logs the request details to the console.
 */
axios.interceptors.request.use(
  (req) => {
    req.baseURL = API_URL;
    req.withCredentials = true;

    req.headers["X-Idempotency-Key"] = generateIdempotencyKey();

    console.log(
      `%c[Trenova] Axios request: ${req.method?.toUpperCase()} ${req.url}`,
      "color: #34ebe5; font-weight: bold",
    );
    return req;
  },
  (error: any) => Promise.reject(error),
);

export default axios;
