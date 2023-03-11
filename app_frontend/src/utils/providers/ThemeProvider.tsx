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

import React, {createContext, useContext, useEffect, useState} from 'react'
import { toAbsoluteUrl } from "@/utils/helpers/AssetHelpers";
import { ToastContainer } from "react-toastify";

export type ThemeModeType = 'dark' | 'light' | 'system'

const systemMode = typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';


type ThemeModeContextType = {
  mode: ThemeModeType
  menuMode: ThemeModeType
  updateMode: (_mode: ThemeModeType) => void
  updateMenuMode: (_mode: ThemeModeType) => void
}

const themeModeSwitchHelper = (_mode: ThemeModeType) => {
  // change background image url
  const mode = _mode !== 'system' ? _mode : systemMode
  const imageUrl = '/media/patterns/header-bg' + (mode === 'light' ? '.jpg' : '-dark.png')
  document.body.style.backgroundImage = `url("${toAbsoluteUrl(imageUrl)}")`
}

const themeModeLSKey = 'mt_theme_mode_value'
const themeMenuModeLSKey = 'mt_theme_mode_menu'

const getThemeModeFromLocalStorage = (lsKey: string, isMenu?: boolean): ThemeModeType => {
  if (typeof localStorage === 'undefined') {
    return 'light'
  }

  const data = localStorage.getItem(lsKey)
  if (data === 'dark' || data === 'light') {
    return data
  }

  if (isMenu && data === 'system') {
    return data
  }

  if (typeof document !== 'undefined' && document.documentElement.hasAttribute('data-bs-theme')) {
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

const ThemeModeContext = createContext<ThemeModeContextType>({
  mode: defaultThemeMode.mode,
  menuMode: defaultThemeMode.menuMode,
  updateMode: (_mode: ThemeModeType) => {},
  updateMenuMode: (_menuMode: ThemeModeType) => {},
})

const useThemeMode = () => useContext(ThemeModeContext)

const ThemeModeProvider = ({children}: {children: React.ReactNode}) => {
  const [mode, setMode] = useState<ThemeModeType>(defaultThemeMode.mode)
  const [menuMode, setMenuMode] = useState<ThemeModeType>(defaultThemeMode.menuMode)

  const updateMode = (_mode: ThemeModeType, saveInLocalStorage: boolean = true) => {
    const updatedMode = _mode === 'system' ? systemMode : _mode
    setMode(updatedMode)
    if (saveInLocalStorage && localStorage) {
      localStorage.setItem(themeModeLSKey, updatedMode)
    }

    if (updatedMode === 'dark') {
      console.log('add dark stylesheet');
      const darkStylesheet = document.createElement('link');
      darkStylesheet.rel = 'stylesheet';
      darkStylesheet.href = 'https://cdn.jsdelivr.net/npm/@sweetalert2/theme-dark@4/dark.css';
      document.head.appendChild(darkStylesheet);
    } else {
      console.log('remove dark stylesheet');
      const darkStylesheet = document.head.querySelector('link[href="https://cdn.jsdelivr.net/npm/@sweetalert2/theme-dark@4/dark.css"]');
      if (darkStylesheet) {
        document.head.removeChild(darkStylesheet);
      }
    }

    if (saveInLocalStorage) {
      document.documentElement.setAttribute('data-bs-theme', updatedMode)
    }
  }

  const updateMenuMode = (_menuMode: ThemeModeType, saveInLocalStorage: boolean = true) => {
    setMenuMode(_menuMode)
    if (saveInLocalStorage && localStorage) {
      localStorage.setItem(themeMenuModeLSKey, _menuMode)
    }
  }

  const themeString = mode === "light" ? "light" : "dark";

  useEffect(() => {
    updateMode(mode, true)
    updateMenuMode(menuMode, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <ThemeModeContext.Provider value={{mode, menuMode, updateMode, updateMenuMode}}>
      <ToastContainer theme={themeString} />
      {children}
    </ThemeModeContext.Provider>
  )
}

export {ThemeModeProvider, useThemeMode, systemMode, themeModeSwitchHelper}
