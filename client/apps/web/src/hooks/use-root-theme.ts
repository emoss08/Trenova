import { useSyncExternalStore } from "react";
import type { ResolvedTheme } from "./use-resolved-theme";

function subscribe(onChange: () => void): () => void {
  const observer = new MutationObserver(onChange);
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
  return () => observer.disconnect();
}

function snapshot(): ResolvedTheme {
  return document.documentElement.classList.contains("dark") ? "dark" : "light";
}

function serverSnapshot(): ResolvedTheme {
  return "light";
}

/**
 * The theme the document is painted in right now, read from the `dark` class on the
 * root element. Unlike useResolvedTheme, it changes only after the class has, so code
 * that reads computed token values on a change reads the new theme's values.
 */
export function useRootTheme(): ResolvedTheme {
  return useSyncExternalStore(subscribe, snapshot, serverSnapshot);
}
