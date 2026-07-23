import { MAP_ID_DARK, MAP_ID_LIGHT } from "@/lib/constants";
import { useTheme } from "@/components/theme-provider";
import { useMemo } from "react";

export function useMapId() {
  const { theme } = useTheme();

  return useMemo(() => {
    if (theme === "system") {
      const prefersDark = window.matchMedia(
        "(prefers-color-scheme: dark)",
      ).matches;
      return prefersDark ? MAP_ID_DARK : MAP_ID_LIGHT;
    }
    return theme === "dark" ? MAP_ID_DARK : MAP_ID_LIGHT;
  }, [theme]);
}
