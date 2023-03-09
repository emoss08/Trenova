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

import React, {useRef, useEffect} from 'react'
import clsx from 'clsx'
import {AsideMenuMain} from './AsideMenuMain'
import { useRouter } from "next/router";
import { DrawerComponent, ScrollComponent, ToggleComponent } from '@/utils/assets/ts/components';

type Props = {
  asideMenuCSSClasses: string[]
}

const AsideMenu: React.FC<Props> = ({asideMenuCSSClasses}) => {
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const {pathname} = useRouter()

  useEffect(() => {
    setTimeout(() => {
      DrawerComponent.reinitialization()
      ToggleComponent.reinitialization()
      ScrollComponent.reinitialization()
      if (scrollRef.current) {
        scrollRef.current.scrollTop = 0
      }
    }, 50)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname])

  return (
    <div
      id='mt_aside_menu_wrapper'
      ref={scrollRef}
      className='hover-scroll-overlay-y px-2 my-5 my-lg-5'
      data-mt-scroll='true'
      data-mt-scroll-height='auto'
      data-mt-scroll-dependencies="{default: '#mt_aside_toolbar, #mt_aside_footer', lg: '#mt_header, #mt_aside_toolbar, #mt_aside_footer'}"
      data-mt-scroll-wrappers='#mt_aside_menu'
      data-mt-scroll-offset='5px'
    >
      <div
        id='#mt_aside_menu'
        data-mt-menu='true'
        className={clsx(
          'menu menu-column menu-title-gray-800 menu-state-title-primary menu-state-icon-primary menu-state-bullet-primary menu-arrow-gray-500',
          asideMenuCSSClasses.join(' ')
        )}
      >
        <AsideMenuMain />
      </div>
    </div>
  )
}

export {AsideMenu}
