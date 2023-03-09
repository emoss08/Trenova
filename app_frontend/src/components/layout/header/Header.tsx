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
import {MenuInner} from './MenuInner'

const Header: FC = () => {
  return (
    <div
      id='mt_header_menu'
      className='header-menu align-items-stretch'
      data-mt-drawer='true'
      data-mt-drawer-name='header-menu'
      data-mt-drawer-activate='{default: true, lg: false}'
      data-mt-drawer-overlay='true'
      data-mt-drawer-width="{default:'200px', '300px': '250px'}"
      data-mt-drawer-direction='end'
      data-mt-drawer-toggle='#mt_header_menu_mobile_toggle'
      data-mt-swapper='true'
      data-mt-swapper-mode='prepend'
      data-mt-swapper-parent="{default: '#mt_body', lg: '#mt_header_nav'}"
    >
      <div
        className='menu menu-lg-rounded menu-column menu-lg-row menu-state-bg menu-title-gray-700 menu-state-title-primary menu-state-icon-primary menu-state-bullet-primary menu-arrow-gray-400 fw-bold my-5 my-lg-0 align-items-stretch'
        id='#mt_header_menu'
        data-mt-menu='true'
      >
        <MenuInner />
      </div>
    </div>
  )
}

export {Header}
