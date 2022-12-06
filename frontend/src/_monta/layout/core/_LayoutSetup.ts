import {ILayout, ILayoutCSSClasses, ILayoutHTMLAttributes, ILayoutCSSVariables} from './_Models'
import {DefaultConfig} from './_LayoutConfig'

const LAYOUT_CONFIG_KEY = process.env.REACT_APP_BASE_LAYOUT_CONFIG_KEY || 'LayoutConfig'

const getLayoutFromLocalStorage = (): ILayout => {
  const ls = localStorage.getItem(LAYOUT_CONFIG_KEY)
  if (ls) {
    try {
      return JSON.parse(ls) as ILayout
    } catch (er) {
      console.error(er)
    }
  }
  return DefaultConfig
}

const setLayoutIntoLocalStorage = (config: ILayout) => {
  try {
    localStorage.setItem(LAYOUT_CONFIG_KEY, JSON.stringify(config))
  } catch (er) {
    console.error(er)
  }
}

const getEmptyCssClasses = (): ILayoutCSSClasses => {
  return {
    header: [],
    headerContainer: [],
    headerMobile: [],
    headerMenu: [],
    aside: [],
    asideMenu: [],
    asideToggle: [],
    toolbar: [],
    toolbarContainer: [],
    content: [],
    contentContainer: [],
    footerContainer: [],
    sidebar: [],
    pageTitle: [],
    pageContainer: [],
  }
}

const getEmptyHTMLAttributes = () => {
  return {
    asideMenu: new Map(),
    headerMobile: new Map(),
    headerMenu: new Map(),
    headerContainer: new Map(),
    pageTitle: new Map(),
  }
}

const getEmptyCSSVariables = () => {
  return {
    body: new Map(),
  }
}

class LayoutSetup {
  public static isLoaded: boolean = false
  public static config: ILayout = getLayoutFromLocalStorage()
  public static classes: ILayoutCSSClasses = getEmptyCssClasses()
  public static attributes: ILayoutHTMLAttributes = getEmptyHTMLAttributes()
  public static cssVariables: ILayoutCSSVariables = getEmptyCSSVariables()

  private static initCSSClasses(): void {
    LayoutSetup.classes = getEmptyCssClasses()
  }

  private static initHTMLAttributes(): void {
    LayoutSetup.attributes = Object.assign({}, getEmptyHTMLAttributes())
  }

  private static initCSSVariables(): void {
    LayoutSetup.cssVariables = getEmptyCSSVariables()
  }

  private static initConfig(config: ILayout): ILayout {
    let updatedConfig = LayoutSetup.initLayoutSettings(config)
    updatedConfig = LayoutSetup.initToolbarSetting(updatedConfig)
    return LayoutSetup.initWidthSettings(updatedConfig)
  }

  private static initLayoutSettings(config: ILayout): ILayout {
    const updatedConfig = {...config}
    // clear body classes
    const bodyClasses = document.body.classList.value.split(' ')
    bodyClasses.forEach((cssClass) => document.body.classList.remove(cssClass))
    // clear body attributes
    const bodyAttributes = document.body.getAttributeNames().filter((t) => t.indexOf('data-') > -1)
    bodyAttributes.forEach((attr) => document.body.removeAttribute(attr))
    document.body.setAttribute('style', '')
    document.body.setAttribute('id', 'kt_app_body')
    document.body.setAttribute('data-kt-app-layout', updatedConfig.layoutType)
    document.body.setAttribute('data-kt-name', 'metronic')
    document.body.classList.add('app-default')

    const pageWidth = updatedConfig.app?.general?.pageWidth
    if (updatedConfig.layoutType === 'light-header' || updatedConfig.layoutType === 'dark-header') {
      if (pageWidth === 'default') {
        const header = updatedConfig.app?.header
        if (header && header.default && header.default.container) {
          header.default.container = 'fixed'
        }

        const toolbar = updatedConfig.app?.toolbar
        if (toolbar) {
          toolbar.container = 'fixed'
        }

        const content = updatedConfig.app?.content
        if (content) {
          content.container = 'fixed'
        }

        const footer = updatedConfig.app?.footer
        if (footer) {
          footer.container = 'fixed'
        }

        const updatedApp = {
          ...updatedConfig.app,
          ...header,
          ...toolbar,
          ...content,
          ...footer,
        }
        return {...updatedConfig, app: updatedApp}
      }
    }
    return updatedConfig
  }

  private static initToolbarSetting(config: ILayout): ILayout {
    const updatedConfig = {...config}
    const appHeaderDefaultContent = updatedConfig.app?.header?.default?.content
    if (appHeaderDefaultContent === 'page-title') {
      const toolbar = updatedConfig.app?.toolbar
      if (toolbar) {
        toolbar.display = false
        const updatedApp = {...updatedConfig.app, ...toolbar}
        return {...updatedConfig, app: updatedApp}
      }
      return updatedConfig
    }

    const pageTitle = updatedConfig.app?.pageTitle
    if (pageTitle) {
      pageTitle.description = false
      pageTitle.breadCrumb = true
      const updatedApp = {...updatedConfig.app, ...pageTitle}
      return {...updatedConfig, app: updatedApp}
    }

    return updatedConfig
  }

  private static initWidthSettings(config: ILayout): ILayout {
    const updatedConfig = {...config}
    const pageWidth = updatedConfig.app?.general?.pageWidth
    if (!pageWidth || pageWidth === 'default') {
      return config
    }

    const header = updatedConfig.app?.header
    if (header && header.default) {
      header.default.container = pageWidth
    }
    const toolbar = updatedConfig.app?.toolbar
    if (toolbar) {
      toolbar.container = pageWidth
    }
    const content = updatedConfig.app?.content
    if (content) {
      content.container = pageWidth
    }
    const footer = updatedConfig.app?.footer
    if (footer) {
      footer.container = pageWidth
    }
    const updatedApp = {
      ...updatedConfig.app,
      ...header,
      ...toolbar,
      ...content,
      ...footer,
    }
    return {...updatedConfig, app: updatedApp}
  }

  public static updatePartialConfig(fieldsToUpdate: Partial<ILayout>): ILayout {
    const config = LayoutSetup.config
    const updatedConfig = {...config, ...fieldsToUpdate}
    LayoutSetup.initCSSClasses()
    LayoutSetup.initCSSVariables()
    LayoutSetup.initHTMLAttributes()
    LayoutSetup.isLoaded = false
    LayoutSetup.config = LayoutSetup.initConfig(Object.assign({}, updatedConfig))
    LayoutSetup.isLoaded = true // remove loading there
    return updatedConfig
  }

  public static setConfig(config: ILayout): void {
    setLayoutIntoLocalStorage(config)
  }

  public static bootstrap = (() => {
    LayoutSetup.updatePartialConfig(LayoutSetup.config)
  })()
}

export {
  LayoutSetup,
  getLayoutFromLocalStorage,
  setLayoutIntoLocalStorage,
  getEmptyCssClasses,
  getEmptyCSSVariables,
  getEmptyHTMLAttributes,
}
