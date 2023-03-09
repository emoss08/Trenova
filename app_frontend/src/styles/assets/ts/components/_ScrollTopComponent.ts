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

import {
  getScrollTop,
  getAttributeValueByBreakpoint,
  throttle,
  getObjectPropertyValueByKey,
  stringSnakeToCamel,
  getUniqueIdWithPrefix,
  DataUtil,
  ElementAnimateUtil,
} from '../_utils/index'

export interface IScrollTopOptions {
  offset: number
  speed: number
}

const defaultScrollTopOptions: IScrollTopOptions = {
  offset: 200,
  speed: 600,
}

class ScrollTopComponent {
  element: HTMLElement
  options: IScrollTopOptions
  instanceUid: string

  constructor(_element: HTMLElement, options: IScrollTopOptions) {
    this.element = _element
    this.options = Object.assign(defaultScrollTopOptions, options)
    this.instanceUid = getUniqueIdWithPrefix('scrolltop')

    // Event Handlers
    this._handlers()

    // Bind Instance
    DataUtil.set(this.element, 'scrolltop', this)
  }

  private _handlers = () => {
    let timer: number
    window.addEventListener('scroll', () => {
      throttle(timer, () => {
        this._scroll()
      })
    })

    this.element.addEventListener('click', (e: Event) => {
      e.preventDefault()
      this._go()
    })
  }

  private _scroll = () => {
    const offset = parseInt(this._getOption('offset') as string)
    const pos = getScrollTop() // current vertical position
    if (pos > offset) {
      if (!document.body.hasAttribute('data-mt-scrolltop')) {
        document.body.setAttribute('data-mt-scrolltop', 'on')
      }
    } else {
      if (document.body.hasAttribute('data-mt-scrolltop')) {
        document.body.removeAttribute('data-mt-scrolltop')
      }
    }
  }

  private _go = () => {
    const speed = parseInt(this._getOption('speed') as string)
    ElementAnimateUtil.scrollTop(0, speed)
  }

  private _getOption = (name: string) => {
    const attr = this.element.getAttribute(`data-mt-scrolltop-${name}`)
    if (attr) {
      const value = getAttributeValueByBreakpoint(attr)
      return value !== null && String(value) === 'true'
    }

    const optionName = stringSnakeToCamel(name)
    const option = getObjectPropertyValueByKey(this.options, optionName)
    if (option) {
      return getAttributeValueByBreakpoint(option)
    }

    return null
  }

  ///////////////////////
  // ** Public API  ** //
  ///////////////////////

  // Plugin API
  public go = () => {
    return this._go()
  }

  public getElement = () => {
    return this.element
  }

  // Static methods
  public static getInstance = (el: HTMLElement): ScrollTopComponent | undefined => {
    const scrollTop = DataUtil.get(el, 'scrolltop')
    if (scrollTop) {
      return scrollTop as ScrollTopComponent
    }
  }

  public static createInstances = (selector: string) => {
    const elements = document.body.querySelectorAll(selector)
    elements.forEach((el) => {
      const item = el as HTMLElement
      let scrollTop = ScrollTopComponent.getInstance(item)
      if (!scrollTop) {
        scrollTop = new ScrollTopComponent(item, defaultScrollTopOptions)
      }
    })
  }

  public static createInsance = (
    selector: string,
    options: IScrollTopOptions = defaultScrollTopOptions
  ): ScrollTopComponent | undefined => {
    const element = document.body.querySelector(selector)
    if (!element) {
      return
    }
    const item = element as HTMLElement
    let scrollTop = ScrollTopComponent.getInstance(item)
    if (!scrollTop) {
      scrollTop = new ScrollTopComponent(item, options)
    }
    return scrollTop
  }

  public static bootstrap = () => {
    ScrollTopComponent.createInstances('[data-mt-scrolltop="true"]')
  }

  public static reinitialization = () => {
    ScrollTopComponent.createInstances('[data-mt-scrolltop="true"]')
  }

  public static goTop = () => {
    ElementAnimateUtil.scrollTop(0, defaultScrollTopOptions.speed)
  }
}
export {ScrollTopComponent, defaultScrollTopOptions}
