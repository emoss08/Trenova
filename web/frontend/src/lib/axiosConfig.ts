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
