import {FC, useEffect} from 'react'
import {ILayout, useLayout} from '../../core'
import {MenuInner} from './header-menus'

const Header: FC = () => {
  const {config} = useLayout()
  useEffect(() => {
    updateDOM(config)
  }, [config])

  return (
    <div
      className='
        menu
        menu-rounded
        menu-column
        menu-lg-row
        my-5
        my-lg-0
        align-items-stretch
        fw-semibold
        px-2 px-lg-0
    '
      id='kt_app_header_menu'
      data-kt-menu='true'
    >
      <MenuInner />
    </div>
  )
}

const updateDOM = (config: ILayout) => {
  if (config.app?.header?.default?.fixed?.desktop) {
    document.body.setAttribute('data-kt-app-header-fixed', 'true')
  }

  if (config.app?.header?.default?.fixed?.mobile) {
    document.body.setAttribute('data-kt-app-header-fixed-mobile', 'true')
  }

  if (config.app?.header?.default?.stacked) {
    document.body.setAttribute('data-kt-app-header-stacked', 'true')
  }

  const appHeaderDefaultStickyEnabled = config.app?.header?.default?.sticky?.enabled
  let appHeaderDefaultStickyAttributes: {[attrName: string]: string} = {}
  if (appHeaderDefaultStickyEnabled) {
    appHeaderDefaultStickyAttributes = config.app?.header?.default?.sticky?.attributes as {
      [attrName: string]: string
    }
  }

  const appHeaderDefaultMinimizeEnabled = config.app?.header?.default?.minimize?.enabled
  let appHeaderDefaultMinimizeAttributes: {[attrName: string]: string} = {}
  if (appHeaderDefaultMinimizeEnabled) {
    appHeaderDefaultMinimizeAttributes = config.app?.header?.default?.minimize?.attributes as {
      [attrName: string]: string
    }
  }

  setTimeout(() => {
    const headerElement = document.getElementById('mt_app_header')
    // header
    if (headerElement) {
      const headerAttributes = headerElement
        .getAttributeNames()
        .filter((t) => t.indexOf('data-') > -1)
      headerAttributes.forEach((attr) => headerElement.removeAttribute(attr))

      if (appHeaderDefaultStickyEnabled) {
        for (const key in appHeaderDefaultStickyAttributes) {
          if (appHeaderDefaultStickyAttributes.hasOwnProperty(key)) {
            headerElement.setAttribute(key, appHeaderDefaultStickyAttributes[key])
          }
        }
      }

      if (appHeaderDefaultMinimizeEnabled) {
        for (const key in appHeaderDefaultMinimizeAttributes) {
          if (appHeaderDefaultMinimizeAttributes.hasOwnProperty(key)) {
            headerElement.setAttribute(key, appHeaderDefaultMinimizeAttributes[key])
          }
        }
      }
    }
  }, 0)
}

export {Header}
