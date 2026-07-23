import { useTheme } from "@/components/theme-provider";
import { useMediaQuery } from "@/hooks/use-media-query";

export type ResolvedTheme = "light" | "dark";

export function useResolvedTheme(): ResolvedTheme {
  const { theme } = useTheme();
  const prefersDark = useMediaQuery("(prefers-color-scheme: dark)");

  if (theme === "dark" || theme === "light") {
    return theme;
  }

  return prefersDark ? "dark" : "light";
}
