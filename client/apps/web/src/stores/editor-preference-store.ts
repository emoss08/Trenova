import { create } from "zustand";
import { persist } from "zustand/middleware";

interface EditorPreferenceState {
  vimKeys: boolean;
  setVimKeys: (on: boolean) => void;
}

/**
 * How this person likes to type code: Vim keys in the formula editor, or the editor's own.
 * Like the theme it is kept on the device, since it follows the keyboard in front of them.
 */
export const useEditorPreferenceStore = create<EditorPreferenceState>()(
  persist(
    (set) => ({
      vimKeys: false,
      setVimKeys: (on) => set({ vimKeys: on }),
    }),
    { name: "trenova-editor-preference" },
  ),
);
