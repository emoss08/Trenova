/**
 * Whether the person is on an Apple platform, where the primary modifier is
 * the command key. Everywhere else it is Ctrl, and a hint that says ⌘ reads
 * as a typo to someone on Windows.
 */
export function isMacPlatform(): boolean {
  if (typeof navigator === "undefined") return false;
  return /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
}

/**
 * The label for a primary-modifier shortcut, spelled the way the person's
 * platform spells it: "⌘K" on a Mac, "Ctrl+K" elsewhere.
 */
export function formatShortcut(key: string, mac: boolean = isMacPlatform()): string {
  return mac ? `⌘${key}` : `Ctrl+${key}`;
}
