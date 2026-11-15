import { hotkeyOf } from "@/config/keybinds.config";
import { SETTINGS_PATH } from "@/lib/settings-path";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useMatchRoute, useNavigate } from "@tanstack/react-router";

/**
 * ⌘⇧S opens Settings from any page in the shell, even with focus in a field. On Settings
 * itself it does nothing, so a second press does not stack another history entry.
 */
export function useSettingsShortcut() {
  const navigate = useNavigate();
  const onSettings = useMatchRoute()({ to: SETTINGS_PATH }) !== false;

  useHotkey(
    hotkeyOf("general", "user-settings"),
    () => {
      if (onSettings) return;
      void navigate({ to: SETTINGS_PATH });
    },
    { ignoreInputs: false, preventDefault: true },
  );
}
