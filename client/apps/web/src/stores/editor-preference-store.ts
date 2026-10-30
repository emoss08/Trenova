import { create } from "zustand";
import { persist } from "zustand/middleware";

interface EditorPreferenceState {
  vimKeys: boolean;
  setVimKeys: (on: boolean) => void;
  /** Whether code draws its ligatures (`=>` as one sign); on unless the person turns it off. */
  codeLigatures: boolean;
  setCodeLigatures: (on: boolean) => void;
  /** Whether a Studio formats the source before it saves; off unless the person turns it on. */
  formatOnSave: boolean;
  setFormatOnSave: (on: boolean) => void;
}

/**
 * How this person likes to read and type code: Vim keys in the formula editor or the editor's
 * own, code's ligatures on or off, and whether a save formats the source first. Like the theme it
 * is kept on the device, since it follows the keyboard and the screen in front of them.
 */
export const useEditorPreferenceStore = create<EditorPreferenceState>()(
  persist(
    (set) => ({
      vimKeys: false,
      setVimKeys: (on) => set({ vimKeys: on }),
      codeLigatures: true,
      setCodeLigatures: (on) => set({ codeLigatures: on }),
      formatOnSave: false,
      setFormatOnSave: (on) => set({ formatOnSave: on }),
    }),
    { name: "trenova-editor-preference" },
  ),
);
