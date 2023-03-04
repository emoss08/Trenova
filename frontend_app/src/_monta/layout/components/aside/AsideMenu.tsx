import React, {useRef, useEffect} from 'react'
import {useLocation} from 'react-router'
import clsx from 'clsx'
import {AsideMenuMain} from './AsideMenuMain'
import {DrawerComponent, ScrollComponent, ToggleComponent} from '../../../assets/ts/components'

type Props = {
  asideMenuCSSClasses: string[]
}

const AsideMenu: React.FC<Props> = ({asideMenuCSSClasses}) => {
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const {pathname} = useLocation()

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
      id='kt_aside_menu_wrapper'
      ref={scrollRef}
      className='hover-scroll-overlay-y px-2 my-5 my-lg-5'
      data-kt-scroll='true'
      data-kt-scroll-height='auto'
      data-kt-scroll-dependencies="{default: '#kt_aside_toolbar, #kt_aside_footer', lg: '#kt_header, #kt_aside_toolbar, #kt_aside_footer'}"
      data-kt-scroll-wrappers='#kt_aside_menu'
      data-kt-scroll-offset='5px'
    >
      <div
        id='#kt_aside_menu'
        data-kt-menu='true'
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
