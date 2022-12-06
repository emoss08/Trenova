import React, {createContext, useContext, useEffect, useState} from 'react'
import {ThemeModeComponent} from '../../../assets/ts/layout'
import {toAbsoluteUrl} from '../../../helpers'

export type ThemeModeType = 'dark' | 'light' | 'system'
export const themeModelSKey = 'mt_theme_mode_value'
export const themeMenuModeLSKey = 'mt_theme_mode_menu'

const systemMode = ThemeModeComponent.getSystemMode() as 'light' | 'dark'

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

const getThemeModeFromLocalStorage = (lsKey: string): ThemeModeType => {
  if (!localStorage) {
    return 'dark'
  }

  const data = localStorage.getItem(lsKey)
  if (data === 'dark' || data === 'light' || data === 'system') {
    return data
  }

  if (document.documentElement.hasAttribute('data-theme')) {
    const dataTheme = document.documentElement.getAttribute('data-theme')
    if (dataTheme && (dataTheme === 'dark' || dataTheme === 'light' || dataTheme === 'system')) {
      return dataTheme
    }
  }

  return 'system'
}

const defaultThemeMode: ThemeModeContextType = {
  mode: getThemeModeFromLocalStorage(themeModelSKey),
  menuMode: getThemeModeFromLocalStorage(themeMenuModeLSKey),
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
    setMode(_mode)
    // themeModeSwitchHelper(updatedMode)
    if (saveInLocalStorage && localStorage) {
      localStorage.setItem(themeModelSKey, _mode)
    }

    if (saveInLocalStorage) {
      const updatedMode = _mode === 'system' ? systemMode : _mode
      document.documentElement.setAttribute('data-theme', updatedMode)
    }
    ThemeModeComponent.init()
  }

  const updateMenuMode = (_menuMode: ThemeModeType, saveInLocalStorage: boolean = true) => {
    setMenuMode(_menuMode)
    if (saveInLocalStorage && localStorage) {
      localStorage.setItem(themeMenuModeLSKey, _menuMode)
    }
  }

  useEffect(() => {
    updateMode(mode, false)
    updateMenuMode(menuMode, false)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <ThemeModeContext.Provider value={{mode, menuMode, updateMode, updateMenuMode}}>
      {children}
    </ThemeModeContext.Provider>
  )
}

export {ThemeModeProvider, useThemeMode, systemMode, themeModeSwitchHelper}
