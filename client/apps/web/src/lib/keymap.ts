import { keybindGroups } from "@/config/keybinds.config";

/** A config key as `KeyboardEvent.key` reports it: letters in lower case, Esc as Escape. */
export function eventKeyOf(configKey: string): string {
  if (configKey === "Esc") {
    return "Escape";
  }

  return configKey.length === 1 ? configKey.toLowerCase() : configKey;
}

/** A pressed key as the keymap is keyed: a single letter in lower case. */
export function normalizedKey(key: string): string {
  return key.length === 1 ? key.toLowerCase() : key;
}

/**
 * `event.key` → command, read from one group of the keybinds config, so the
 * shortcut sheet and the bindings are one list. Only single-key bindings are
 * read; `aliases` adds keys the sheet does not list, such as the arrows that
 * do what J and K do.
 */
export function keymapFromConfig<Command extends string>(
  groupId: string,
  commands: Readonly<Record<string, Command>>,
  aliases: Readonly<Record<string, Command>> = {},
): ReadonlyMap<string, Command> {
  const keybinds = keybindGroups.find((group) => group.id === groupId)?.keybinds ?? [];

  return new Map<string, Command>([
    ...keybinds.flatMap((keybind): Array<[string, Command]> => {
      const command = commands[keybind.id];
      return command === undefined || keybind.keys.length !== 1
        ? []
        : [[eventKeyOf(keybind.keys[0]), command]];
    }),
    ...Object.entries(aliases),
  ]);
}

/** Whether a key landed in something that takes text. */
export function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  return (
    target.isContentEditable ||
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.tagName === "SELECT"
  );
}
