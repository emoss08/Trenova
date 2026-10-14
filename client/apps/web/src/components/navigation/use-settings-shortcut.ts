import { SETTINGS_PATH } from "@/lib/settings-path";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useMatch, useNavigate } from "react-router";

/**
 * ⌘⇧S opens Settings from any page in the shell, even with focus in a field. On Settings
 * itself it does nothing, so a second press does not stack another history entry.
 */
export function useSettingsShortcut() {
  const navigate = useNavigate();
  const onSettings = useMatch(SETTINGS_PATH) !== null;

  useHotkey(
    "Mod+Shift+S",
    () => {
      if (onSettings) return;
      void navigate(SETTINGS_PATH);
    },
    { ignoreInputs: false, preventDefault: true },
  );
}
