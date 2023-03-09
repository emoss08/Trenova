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

import React, {useEffect, useRef} from 'react'
import {
  ScrollTopComponent,
  DrawerComponent,
  ToggleComponent,
  StickyComponent,
} from '@/utils/assets/ts/components'
import { useRouter } from "next/router";
import Image from "next/image";

import arr066 from "../../../public/media/icons/duotune/arrows/arr066.svg";

export function ScrollTop() {
  const {pathname} = useRouter()
  const isFirstRun = useRef(true)

  const pluginsReinitialization = () => {
    setTimeout(() => {
      StickyComponent.reInitialization()
      setTimeout(() => {
        ToggleComponent.reinitialization()
        DrawerComponent.reinitialization()
      }, 70)
    }, 140)
  }

  const scrollTop = () => {
    ScrollTopComponent.goTop()
  }

  const updateHeaderSticky = () => {
    const stickyHeader = document.body.querySelectorAll(`[data-mt-sticky-name="header"]`)
    if (stickyHeader && stickyHeader.length > 0) {
      const sticky = StickyComponent.getInstance(stickyHeader[0] as HTMLElement)
      if (sticky) {
        sticky.update()
      }
    }
  }

  useEffect(() => {
    if (isFirstRun.current) {
      isFirstRun.current = false
    } else {
      pluginsReinitialization()
    }

    updateHeaderSticky()
    setTimeout(() => {
      scrollTop()
    }, 0)
  }, [pathname])

  return (
    <div id='mt_scrolltop' className='scrolltop' data-mt-scrolltop='true'>
      <Image src={arr066} alt={"Arrow"} />
    </div>
  )
}
