import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { useEffect } from "react";

/**
 * Applies the person's code-ligature choice everywhere code is drawn, through the root attribute
 * app.css reads (`--code-ligatures`).
 */
export function CodeLigaturesPreference() {
  const codeLigatures = useEditorPreferenceStore((state) => state.codeLigatures);

  useEffect(() => {
    const root = document.documentElement;
    if (codeLigatures) {
      delete root.dataset.codeLigatures;
    } else {
      root.dataset.codeLigatures = "off";
    }
  }, [codeLigatures]);

  return null;
}
