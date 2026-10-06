import { defineEdition } from "./resolve";

/**
 * The edition the web app loads when no overlay is installed: the self-hosted product,
 * with nothing added. `@trenova/edition-entry` resolves here unless
 * `client/packages/cloud/src/index.ts` exists.
 */
export default defineEdition({ id: "self-hosted", name: "Trenova" });
