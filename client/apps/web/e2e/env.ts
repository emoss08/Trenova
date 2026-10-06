import path from "node:path";
import { fileURLToPath } from "node:url";

/** Settings the tests read from the environment, with the development stack's defaults. */
const dirname = path.dirname(fileURLToPath(import.meta.url));

export const E2E_DIR = dirname;
export const BASE_URL = process.env.E2E_BASE_URL ?? "http://localhost:5173";
export const EMAIL = process.env.E2E_EMAIL ?? "admin@trenova.app";
export const PASSWORD = process.env.E2E_PASSWORD ?? "admin123!";
export const STORAGE_STATE =
  process.env.E2E_STORAGE_STATE ?? path.join(dirname, ".auth", "user.json");
export const CHROMIUM_PATH = process.env.E2E_CHROMIUM_PATH || undefined;
/** Let approvals go through instead of undoing them inside their undo window. */
export const COMMIT_APPROVALS = process.env.E2E_COMMIT === "1";
