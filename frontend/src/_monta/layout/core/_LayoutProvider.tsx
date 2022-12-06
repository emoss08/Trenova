import {FC, createContext, useContext, useState, useEffect} from 'react'
import {DefaultConfig} from './_LayoutConfig'
import {
  setLayoutIntoLocalStorage,
  getEmptyCssClasses,
  getEmptyCSSVariables,
  getEmptyHTMLAttributes,
  LayoutSetup,
} from './_LayoutSetup'
import {
  ILayout,
  ILayoutCSSVariables,
  ILayoutCSSClasses,
  ILayoutHTMLAttributes,
  LayoutType,
  ToolbarType,
} from './_Models'
import {WithChildren} from '../../helpers'

export interface LayoutContextModel {
  config: ILayout
  classes: ILayoutCSSClasses
  attributes: ILayoutHTMLAttributes
  cssVariables: ILayoutCSSVariables
  setLayout: (config: LayoutSetup) => void
  setLayoutType: (layoutType: LayoutType) => void
  setToolbarType: (toolbarType: ToolbarType) => void
}

const LayoutContext = createContext<LayoutContextModel>({
  config: DefaultConfig,
  classes: getEmptyCssClasses(),
  attributes: getEmptyHTMLAttributes(),
  cssVariables: getEmptyCSSVariables(),
  setLayout: (config: LayoutSetup) => {},
  setLayoutType: (layoutType: LayoutType) => {},
  setToolbarType: (toolbarType: ToolbarType) => {},
})

const enableSplashScreen = () => {
  const splashScreen = document.getElementById('splash-screen')
  if (splashScreen) {
    splashScreen.style.setProperty('display', 'flex')
  }
}

const disableSplashScreen = () => {
  const splashScreen = document.getElementById('splash-screen')
  if (splashScreen) {
    splashScreen.style.setProperty('display', 'none')
  }
}

const LayoutProvider: FC<WithChildren> = ({children}) => {
  const [config, setConfig] = useState(LayoutSetup.config)
  const [classes, setClasses] = useState(LayoutSetup.classes)
  const [attributes, setAttributes] = useState(LayoutSetup.attributes)
  const [cssVariables, setCSSVariables] = useState(LayoutSetup.cssVariables)

  const setLayout = (_themeConfig: Partial<ILayout>) => {
    enableSplashScreen()
    const bodyClasses = Array.from(document.body.classList)
    bodyClasses.forEach((cl) => document.body.classList.remove(cl))
    const updatedConfig = LayoutSetup.updatePartialConfig(_themeConfig)
    setConfig(Object.assign({}, updatedConfig))
    setClasses(LayoutSetup.classes)
    setAttributes(LayoutSetup.attributes)
    setCSSVariables(LayoutSetup.cssVariables)
    setTimeout(() => {
      disableSplashScreen()
    }, 500)
  }

  const setToolbarType = (toolbarType: ToolbarType) => {
    const updatedConfig = {...config}
    if (updatedConfig.app?.toolbar) {
      updatedConfig.app.toolbar.layout = toolbarType
    }

    setLayoutIntoLocalStorage(updatedConfig)
    window.location.reload()
  }

  const setLayoutType = (layoutType: LayoutType) => {
    const updatedLayout = {...config, layoutType}
    setLayoutIntoLocalStorage(updatedLayout)
    window.location.reload()
  }

  const value: LayoutContextModel = {
    config,
    classes,
    attributes,
    cssVariables,
    setLayout,
    setLayoutType,
    setToolbarType,
  }

  useEffect(() => {
    disableSplashScreen()
  }, [])

  return <LayoutContext.Provider value={value}>{children}</LayoutContext.Provider>
}

export {LayoutContext, LayoutProvider}

export function useLayout() {
  return useContext(LayoutContext)
}
