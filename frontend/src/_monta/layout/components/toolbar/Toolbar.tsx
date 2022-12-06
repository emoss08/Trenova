import {useEffect} from 'react'
import {ILayout, useLayout} from '../../core'
import {
  ToolbarAccounting,
  ToolbarClassic,
  ToolbarExtended,
  ToolbarReports,
  ToolbarSaas,
} from './toolbars'

const Toolbar = () => {
  const {config} = useLayout()
  useEffect(() => {
    updateDOM(config)
    document.body.setAttribute('data-kt-app-toolbar-enabled', 'true')
  }, [config])

  switch (config.app?.toolbar?.layout) {
    case 'classic':
      return <ToolbarClassic />
    case 'accounting':
      return <ToolbarAccounting />
    case 'extended':
      return <ToolbarExtended />
    case 'reports':
      return <ToolbarReports />
    case 'saas':
      return <ToolbarSaas />
    default:
      return <ToolbarClassic />
  }
}

const updateDOM = (config: ILayout) => {
  let appToolbarSwapAttributes: {[attrName: string]: string} = {}
  const appToolbarSwapEnabled = config.app?.toolbar?.swap?.enabled
  if (appToolbarSwapEnabled) {
    appToolbarSwapAttributes = config.app?.toolbar?.swap?.attributes as {[attrName: string]: string}
  }

  let appToolbarStickyAttributes: {[attrName: string]: string} = {}
  const appToolbarStickyEnabled = config.app?.toolbar?.sticky?.enabled
  if (appToolbarStickyEnabled) {
    appToolbarStickyAttributes = config.app?.toolbar?.sticky?.attributes as {
      [attrName: string]: string
    }

    let appToolbarMinimizeAttributes: {[attrName: string]: string} = {}
    const appToolbarMinimizeEnabled = config.app?.toolbar?.minimize?.enabled
    if (appToolbarMinimizeEnabled) {
      appToolbarMinimizeAttributes = config.app?.toolbar?.minimize?.attributes as {
        [attrName: string]: string
      }
    }

    if (config.app?.toolbar?.fixed?.desktop) {
      document.body.setAttribute('data-kt-app-toolbar-fixed', 'true')
    }

    if (config.app?.toolbar?.fixed?.mobile) {
      document.body.setAttribute('data-kt-app-toolbar-fixed-mobile', 'true')
    }

    setTimeout(() => {
      const toolbarElement = document.getElementById('kt_app_toolbar')
      // toolbar
      if (toolbarElement) {
        const toolbarAttributes = toolbarElement
          .getAttributeNames()
          .filter((t) => t.indexOf('data-') > -1)
        toolbarAttributes.forEach((attr) => toolbarElement.removeAttribute(attr))

        if (appToolbarSwapEnabled) {
          for (const key in appToolbarSwapAttributes) {
            if (appToolbarSwapAttributes.hasOwnProperty(key)) {
              toolbarElement.setAttribute(key, appToolbarSwapAttributes[key])
            }
          }
        }

        if (appToolbarStickyEnabled) {
          for (const key in appToolbarStickyAttributes) {
            if (appToolbarStickyAttributes.hasOwnProperty(key)) {
              toolbarElement.setAttribute(key, appToolbarStickyAttributes[key])
            }
          }
        }

        if (appToolbarMinimizeEnabled) {
          for (const key in appToolbarMinimizeAttributes) {
            if (appToolbarMinimizeAttributes.hasOwnProperty(key)) {
              toolbarElement.setAttribute(key, appToolbarMinimizeAttributes[key])
            }
          }
        }
      }
    }, 0)
  }
}

export {Toolbar}
