import {themeMenuModeLSKey, themeModelSKey} from '../../../partials'
import {EventHandlerUtil} from '../_utils'

type Mode = 'light' | 'dark' | 'system'

class ThemeMode {
  menu: HTMLElement | null = null
  element: HTMLElement | null = null

  public getMode = (): Mode => {
    const menuMode: Mode | '' = this.getMenuMode()
    const defaultMode = 'light'
    if (!localStorage) {
      return defaultMode
    }

    const ls = localStorage.getItem(themeModelSKey)
    if (ls) {
      return ls as Mode
    }

    const dataTheme = this.element?.getAttribute('data-theme')
    if (dataTheme) {
      return dataTheme as Mode
    }

    if (!menuMode) {
      return defaultMode
    }

    if (menuMode === 'system') {
      return this.getSystemMode()
    }

    return menuMode
  }

  public setMode = (mode: Mode, menuMode: Mode | ''): void => {
    // Check input values
    if (mode !== 'light' && mode !== 'dark') {
      return
    }

    // Reset mode if system mode was changed
    if (menuMode === 'system') {
      if (this.getSystemMode() !== mode) {
        mode = this.getSystemMode()
      }
    }

    // Check menu mode
    if (!menuMode) {
      menuMode = mode
    }

    // Read active menu mode value
    const activeMenuItem: HTMLElement | null =
      this.menu?.querySelector('[data-mt-element="mode"][data-mt-value="' + menuMode + '"]') || null

    // Enable switching state
    this.element?.setAttribute('data-mt-theme-mode-switching', 'true')

    // Set mode to the target element
    this.element?.setAttribute('data-theme', mode)

    // Disable switching state
    const self = this
    setTimeout(function () {
      self.element?.removeAttribute('data-mt-theme-mode-switching')
    }, 300)

    // Store mode value in storage
    if (localStorage) {
      localStorage.setItem(themeModelSKey, mode)
    }

    // Set active menu item
    if (activeMenuItem && localStorage) {
      localStorage.setItem(themeMenuModeLSKey, menuMode)
      this.setActiveMenuItem(activeMenuItem)
    }

    // Flip images
    this.flipImages()
  }

  public getMenuMode = (): Mode | '' => {
    const menuItem = this.menu?.querySelector('.active[data-mt-element="mode"]')
    const dataKTValue = menuItem?.getAttribute('data-mt-value')
    if (dataKTValue) {
      return dataKTValue as Mode
    }

    const ls = localStorage ? localStorage.getItem(themeMenuModeLSKey) : null
    return (ls as Mode) || ''
  }

  public getSystemMode = (): Mode => {
    return window.matchMedia('(prefers-color-scheme: dark)') ? 'dark' : 'light'
  }

  private initMode = (): void => {
    this.setMode(this.getMode(), this.getMenuMode())
    if (this.element) {
      EventHandlerUtil.trigger(this.element, 'kt.thememode.init')
    }
  }

  private getActiveMenuItem = (): HTMLElement | null => {
    return (
      this.menu?.querySelector(
        '[data-mt-element="mode"][data-mt-value="' + this.getMenuMode() + '"]'
      ) || null
    )
  }

  private setActiveMenuItem = (item: HTMLElement): void => {
    const menuMode = item.getAttribute('data-mt-value')
    const activeItem = this.menu?.querySelector('.active[data-mt-element="mode"]')
    if (activeItem) {
      activeItem.classList.remove('active')
    }

    item.classList.add('active')
    if (localStorage && menuMode) {
      localStorage.setItem(themeMenuModeLSKey, menuMode)
    }
  }

  private handleMenu = (): void => {
    this.menu
      ?.querySelectorAll<HTMLElement>('[data-mt-element="mode"]')
      ?.forEach((item: HTMLElement) => {
        item.addEventListener('click', (e) => {
          e.preventDefault()

          const menuMode: string | null = item.getAttribute('data-mt-value')
          const mode = menuMode === 'system' ? this.getSystemMode() : menuMode

          if (mode) {
            this.setMode(mode as Mode, menuMode as Mode | '')
          }
        })
      })
  }

  public flipImages = () => {
    document.querySelectorAll<HTMLElement>('[data-mt-img-dark]')?.forEach((item: HTMLElement) => {
      if (item.tagName === 'IMG') {
        if (this.getMode() === 'dark' && item.hasAttribute('data-mt-img-dark')) {
          item.setAttribute('data-mt-img-light', item.getAttribute('src') || '')
          item.setAttribute('src', item.getAttribute('data-mt-img-dark') || '')
        } else if (this.getMode() === 'light' && item.hasAttribute('data-mt-img-light')) {
          item.setAttribute('data-mt-img-dark', item.getAttribute('src') || '')
          item.setAttribute('src', item.getAttribute('data-mt-img-light') || '')
        }
      } else {
        if (this.getMode() === 'dark' && item.hasAttribute('data-mt-img-dark')) {
          item.setAttribute('data-mt-img-light', item.getAttribute('src') || '')
          item.style.backgroundImage = "url('" + item.getAttribute('data-mt-img-dark') + "')"
        } else if (this.getMode() === 'light' && item.hasAttribute('data-mt-img-light')) {
          item.setAttribute('data-mt-img-dark', item.getAttribute('src') || '')
          item.style.backgroundImage = "url('" + item.getAttribute('data-mt-img-light') + "')"
        }
      }
    })
  }

  public on = (name: string, hander: Function) => {
    if (this.element) {
      return EventHandlerUtil.on(this.element, name, hander)
    }
  }

  public off = (name: string, handlerId: string) => {
    if (this.element) {
      return EventHandlerUtil.off(this.element, name, handlerId)
    }
  }

  public init = () => {
    this.menu = document.querySelector<HTMLElement>('[data-kt-element="theme-mode-menu"]')
    this.element = document.documentElement

    this.initMode()

    if (this.menu) {
      this.handleMenu()
    }
  }
}

const ThemeModeComponent = new ThemeMode()
// Initialize app on document ready => ThemeModeComponent.init()
export {ThemeModeComponent}
