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
  // i18n-ignore: key names as printed on the keyboard
  return mac ? `⌘${key}` : `Ctrl+${key}`;
}

/**
 * The label for an Alt (Option) shortcut, spelled the way the person's
 * platform spells it: "⌥L" on a Mac, "Alt+L" elsewhere.
 */
export function formatAltShortcut(key: string, mac: boolean = isMacPlatform()): string {
  // i18n-ignore: key names as printed on the keyboard
  return mac ? `⌥${key}` : `Alt+${key}`;
}
