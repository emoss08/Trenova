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

import {FC} from 'react'
import clsx from 'clsx'
import {HeaderNotificationsMenu, HeaderUserMenu, QuickLinks, Search} from '@/components/partials'
import {useLayout} from '@/utils/layout/LayoutProvider'
import { toAbsoluteUrl } from "@/utils/AssetHelpers";

const toolbarButtonMarginClass = 'ms-1 ms-lg-3',
  toolbarButtonHeightClass = 'w-30px h-30px w-md-40px h-md-40px',
  toolbarUserAvatarHeightClass = 'symbol-30px symbol-md-40px',
  toolbarButtonIconSizeClass = 'svg-icon-1'

const Topbar: FC = () => {
  const {config} = useLayout()

  return (
    <div className='d-flex align-items-stretch flex-shrink-0'>
      {/* Search */}
      <div className={clsx('d-flex align-items-stretch', toolbarButtonMarginClass)}>
        <Search />
      </div>
      {/* Activities */}
      <div className={clsx('d-flex align-items-center', toolbarButtonMarginClass)}>
        {/* begin::Drawer toggle */}
        <div
          className={clsx(
            'btn btn-icon btn-active-light-primary btn-custom',
            toolbarButtonHeightClass
          )}
          id='mt_activities_toggle'
        >
          <img
            src='/media/icons/duotune/general/gen032.svg'
            className={toolbarButtonIconSizeClass}
          />
        </div>
        {/* end::Drawer toggle */}
      </div>

      {/* NOTIFICATIONS */}
      <div className={clsx('d-flex align-items-center', toolbarButtonMarginClass)}>
        {/* begin::Menu- wrapper */}
        <div
          className={clsx(
            'btn btn-icon btn-active-light-primary btn-custom',
            toolbarButtonHeightClass
          )}
          data-mt-menu-trigger='click'
          data-mt-menu-attach='parent'
          data-mt-menu-placement='bottom-end'
          data-mt-menu-flip='bottom'
        >
          <img
            src='/media/icons/duotune/general/gen022.svg'
            className={toolbarButtonIconSizeClass}
          />
        </div>
        <HeaderNotificationsMenu />
        {/* end::Menu wrapper */}
      </div>

      {/* CHAT */}
      <div className={clsx('d-flex align-items-center', toolbarButtonMarginClass)}>
        {/* begin::Menu wrapper */}
        <div
          className={clsx(
            'btn btn-icon btn-active-light-primary btn-custom position-relative',
            toolbarButtonHeightClass
          )}
          id='mt_drawer_chat_toggle'
        >
          <img
            src='/media/icons/duotune/communication/com012.svg'
            className={toolbarButtonIconSizeClass}
          />

          <span className='bullet bullet-dot bg-success h-6px w-6px position-absolute translate-middle top-0 start-50 animation-blink'></span>
        </div>
        {/* end::Menu wrapper */}
      </div>

      {/* Quick links */}
      <div className={clsx('d-flex align-items-center', toolbarButtonMarginClass)}>
        {/* begin::Menu wrapper */}
        <div
          className={clsx(
            'btn btn-icon btn-active-light-primary btn-custom',
            toolbarButtonHeightClass
          )}
          data-mt-menu-trigger='click'
          data-mt-menu-attach='parent'
          data-mt-menu-placement='bottom-end'
          data-mt-menu-flip='bottom'
        >
          <img
            src='/media/icons/duotune/general/gen025.svg'
            className={toolbarButtonIconSizeClass}
            width={20}
            height={20}
          />
        </div>
        <QuickLinks />
        {/* end::Menu wrapper */}
      </div>

      {/* begin::User */}
      <div
        className={clsx('d-flex align-items-center', toolbarButtonMarginClass)}
        id='mt_header_user_menu_toggle'
      >
        {/* begin::Toggle */}
        <div
          className={clsx('cursor-pointer symbol', toolbarUserAvatarHeightClass)}
          data-mt-menu-trigger='click'
          data-mt-menu-attach='parent'
          data-mt-menu-placement='bottom-end'
          data-mt-menu-flip='bottom'
        >
          <img src={toAbsoluteUrl('/media/avatars/300-1.jpg')} alt='metronic' />
        </div>
        <HeaderUserMenu />
        {/* end::Toggle */}
      </div>
      {/* end::User */}

      {/* begin::Aside Toggler */}
      {config.header.left === 'menu' && (
        <div className='d-flex align-items-center d-lg-none ms-2 me-n3' title='Show header menu'>
          <div
            className='btn btn-icon btn-active-light-primary w-30px h-30px w-md-40px h-md-40px'
            id='mt_header_menu_mobile_toggle'
          >
            <img src='/media/icons/duotune/text/txt001.svg' className='svg-icon-1' />
          </div>
        </div>
      )}
    </div>
  )
}

export {Topbar}
