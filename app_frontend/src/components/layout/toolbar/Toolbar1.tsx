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

/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import React, {FC} from 'react'
import {useLayout} from '@/utils/layout/LayoutProvider'

import {DefaultTitle} from '../header/page-title/DefaultTitle'
import Image from "next/image";

const Toolbar1: FC = () => {
  const {classes} = useLayout()

  return (
    <div className='toolbar' id='mt_toolbar'>
      {/* begin::Container */}
      <div
        id='mt_toolbar_container'
        className={clsx(classes.toolbarContainer.join(' '), 'd-flex flex-stack')}
      >
        <DefaultTitle />

        {/* begin::Actions */}
        <div className='d-flex align-items-center py-1'>
          {/* begin::Wrapper */}
          <div className='me-4'>
            {/* begin::Menu */}
            <a
              href='#'
              className='btn btn-sm btn-flex btn-light btn-active-primary fw-bolder'
              data-mt-menu-trigger='click'
              data-mt-menu-placement='bottom-end'
              data-mt-menu-flip='top-end'
            >
              <Image
                src='/media/icons/duotune/general/gen031.svg'
                className='svg-icon-5 svg-icon-gray-500 me-1'
               alt={"gen031"}/>
              Filter
            </a>

            {/* end::Menu */}
          </div>
          {/* end::Wrapper */}

          {/* begin::Button */}

          <a
            href='#'
            className='btn btn-sm btn-primary'
            data-bs-toggle='modal'
            data-bs-target='#mt_modal_create_app'
            id='mt_toolbar_primary_button'
          >
            Create
          </a>
          {/* end::Button */}
        </div>
        {/* end::Actions */}
      </div>
      {/* end::Container */}
    </div>
  )
}

export {Toolbar1}
