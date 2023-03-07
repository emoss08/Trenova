/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import { useRouter } from "next/router";
import React, { createContext, useContext, useEffect, useState } from "react";
import { ThemeProvider } from "styled-components";

export type ThemeModeType = "dark" | "light" | "system";
const systemMode =
  typeof window !== "undefined" &&
  window.matchMedia("(prefers-color-scheme: dark)")
    ? "dark"
    : "light";

const useThemeMode = () => useContext(ThemeModeContext);

type ThemeModeContextType = {
  mode: ThemeModeType;
  menuMode: ThemeModeType;
  updateMode: (_mode: ThemeModeType) => void;
  updateMenuMode: (_mode: ThemeModeType) => void;
};

const themeModeLSKey = "mt_theme_mode_value";
const themeMenuModeLSKey = "mt_theme_mode_menu";


const getThemeModeFromLocalStorage = (lsKey: string, isMenu?: boolean): ThemeModeType => {
  if (typeof localStorage === "undefined") {
    return 'light'
  }

  const data = localStorage.getItem(lsKey)
  if (data === 'dark' || data === 'light') {
    return data
  }

  if (isMenu && data === 'system') {
    return data
  }

  if (typeof document !== "undefined" && document.documentElement.hasAttribute('data-bs-theme')) {
    const dataTheme = document.documentElement.getAttribute('data-bs-theme')
    if (dataTheme && (dataTheme === 'dark' || dataTheme === 'light')) {
      return dataTheme
    }
  }

  return 'system'
}

const defaultThemeMode: ThemeModeContextType = {
  mode: getThemeModeFromLocalStorage(themeModeLSKey),
  menuMode: getThemeModeFromLocalStorage(themeMenuModeLSKey, true),
  updateMode: (_mode: ThemeModeType) => {},
  updateMenuMode: (_menuMode: ThemeModeType) => {},
}

const themeModeSwitchHelper = (_mode: ThemeModeType) => {
  // change background image url
  const mode = _mode !== 'system' ? _mode : systemMode
  const imageUrl = '/media/patterns/header-bg' + (mode === 'light' ? '.jpg' : '-dark.png')
  document.body.style.backgroundImage = `url("${imageUrl}")`
}

const ThemeModeContext = createContext<ThemeModeContextType>({
  mode: defaultThemeMode.mode,
  menuMode: defaultThemeMode.menuMode,
  updateMode: (_mode: ThemeModeType) => {},
  updateMenuMode: (_menuMode: ThemeModeType) => {},
})

const ThemeProviderWrapper = ({ children }: { children: React.ReactNode }) => {
  const router = useRouter();
  const [mode, setMode] = useState<ThemeModeType>(defaultThemeMode.mode)
  const [menuMode, setMenuMode] = useState<ThemeModeType>(defaultThemeMode.menuMode)
  const updateMode = (_mode: ThemeModeType, saveInLocalStorage: boolean = true) => {
    const updatedMode = _mode === 'system' ? systemMode : _mode
    setMode(updatedMode)
    if (saveInLocalStorage && typeof localStorage !== "undefined") {
      localStorage.setItem(themeModeLSKey, updatedMode)
    }
    if (saveInLocalStorage && typeof document !== "undefined") {
      document.documentElement.setAttribute('data-bs-theme', updatedMode)
    }
  }

  const updateMenuMode = (_menuMode: ThemeModeType, saveInLocalStorage: boolean = true) => {
    setMenuMode(_menuMode)
    if (saveInLocalStorage && typeof localStorage !== "undefined") {
      localStorage.setItem(themeMenuModeLSKey, _menuMode)
    }
  }

  useEffect(() => {
    updateMode(mode, false)
    updateMenuMode(menuMode, false)
// eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <ThemeProvider theme={{ mode, menuMode, updateMode, updateMenuMode }}>
      {children}
    </ThemeProvider>
  )
}

export { ThemeProviderWrapper as ThemeModeProvider, useThemeMode, systemMode, themeModeSwitchHelper }