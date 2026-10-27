import { create } from "zustand";
import { persist } from "zustand/middleware";

interface EditorPreferenceState {
  vimKeys: boolean;
  setVimKeys: (on: boolean) => void;
  /** Whether code draws its ligatures (`=>` as one sign); on unless the person turns it off. */
  codeLigatures: boolean;
  setCodeLigatures: (on: boolean) => void;
}

/**
 * How this person likes to read and type code: Vim keys in the formula editor or the editor's
 * own, and code's ligatures on or off. Like the theme it is kept on the device, since it follows
 * the keyboard and the screen in front of them.
 */
export const useEditorPreferenceStore = create<EditorPreferenceState>()(
  persist(
    (set) => ({
      vimKeys: false,
      setVimKeys: (on) => set({ vimKeys: on }),
      codeLigatures: true,
      setCodeLigatures: (on) => set({ codeLigatures: on }),
    }),
    { name: "trenova-editor-preference" },
  ),
);
