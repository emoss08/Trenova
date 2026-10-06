import { existsSync } from "node:fs";
import path from "node:path";

/** The specifier the web app imports its edition through. */
export const EDITION_ENTRY_SPECIFIER = "@trenova/edition-entry";

const CLIENT_ROOT = path.resolve(import.meta.dirname, "../../..");

/** The overlay's entry. It exists only when an edition has been copied into the tree. */
export const OVERLAY_ENTRY = path.join(CLIENT_ROOT, "packages/cloud/src/index.ts");

/** The overlay's source directory, for configs that glob its tests or stories. */
export const OVERLAY_SRC = path.dirname(OVERLAY_ENTRY);

/** The public no-op edition used when no overlay is present. */
export const DEFAULT_ENTRY = path.join(CLIENT_ROOT, "packages/edition/src/default-entry.ts");

/** Whether an overlay is installed. Read once per config load. */
export function overlayPresent(): boolean {
  return existsSync(OVERLAY_ENTRY);
}

/**
 * The file `@trenova/edition-entry` resolves to: the overlay when present, otherwise the
 * public default. Mirrors the `paths` fallback in apps/web/tsconfig.app.json, which tries
 * the same two files in the same order.
 */
export function editionEntryPath(): string {
  return overlayPresent() ? OVERLAY_ENTRY : DEFAULT_ENTRY;
}

/** Vite/Vitest alias for the edition entry. */
export function editionAlias(): Record<string, string> {
  return { [EDITION_ENTRY_SPECIFIER]: editionEntryPath() };
}
