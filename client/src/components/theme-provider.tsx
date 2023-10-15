/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { THEME_KEY } from "@/lib/constants";
import { ThemeOptions as Theme } from "@/types";
import { createContext, useContext, useEffect, useState } from "react";

type ThemeProviderProps = {
  children: React.ReactNode;
  defaultTheme?: Theme;
  storageKey?: string;
};

type ThemeProviderState = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  isRainbowAnimationActive: boolean;
  toggleRainbowAnimation: () => void;
};

const initialState: ThemeProviderState = {
  theme: "system",
  setTheme: () => null,
  isRainbowAnimationActive: false,
  toggleRainbowAnimation: () => null,
};

const ThemeProviderContext = createContext<ThemeProviderState>(initialState);

export function ThemeProvider({
  children,
  defaultTheme = "system",
  storageKey = THEME_KEY,
  ...props
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<Theme>(
    () => (localStorage.getItem(storageKey) as Theme) || defaultTheme,
  );

  const [isRainbowAnimationActive, setIsRainbowAnimationActive] =
    useState<boolean>(() => {
      const storedValue = localStorage.getItem(
        storageKey + "-rainbow-animation",
      );
      return storedValue ? JSON.parse(storedValue) : false;
    });

  useEffect(() => {
    const root = window.document.documentElement;

    root.classList.remove("light", "dark");

    if (theme === "system") {
      const systemTheme = window.matchMedia("(prefers-color-scheme: dark)")
        .matches
        ? "dark"
        : "light";

      root.classList.add(systemTheme);
      return;
    }

    root.classList.add(theme);

    if (isRainbowAnimationActive) {
      root.classList.add("rainbow-animation");
    } else {
      root.classList.remove("rainbow-animation");
    }
  }, [theme, isRainbowAnimationActive]);

  const value = {
    theme,
    setTheme: (theme: Theme) => {
      localStorage.setItem(storageKey, theme);
      setTheme(theme);
    },
    isRainbowAnimationActive,
    toggleRainbowAnimation: () => {
      const newValue = !isRainbowAnimationActive;
      localStorage.setItem(
        storageKey + "-rainbow-animation",
        JSON.stringify(newValue),
      );
      setIsRainbowAnimationActive(newValue);
    },
  };

  return (
    <ThemeProviderContext.Provider {...props} value={value}>
      {children}
    </ThemeProviderContext.Provider>
  );
}

export const useTheme = () => {
  const context = useContext(ThemeProviderContext);

  if (context === undefined)
    throw new Error("useTheme must be used within a ThemeProvider");

  return context;
};
