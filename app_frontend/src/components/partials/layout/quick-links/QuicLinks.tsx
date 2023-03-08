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
import React, {FC} from 'react'
import { toAbsoluteUrl } from "@/utils/AssetHelpers";
import Link from 'next/link';

const QuickLinks: FC = () => (
  <div
    className='menu menu-sub menu-sub-dropdown menu-column w-250px w-lg-325px'
    data-kt-menu='true'
  >
    <div
      className='d-flex flex-column flex-center bgi-no-repeat rounded-top px-9 py-10'
      style={{backgroundImage: `url('${toAbsoluteUrl('/media/misc/pattern-1.jpg')}')`}}
    >
      <h3 className='text-white fw-bold mb-3'>Quick Links</h3>

      <span className='badge bg-primary py-2 px-3'>25 pending tasks</span>
    </div>

    <div className='row g-0'>
      <div className='col-6'>
        <a
          href='#'
          className='d-flex flex-column flex-center h-100 p-6 bg-hover-light border-end border-bottom'
        >
          <img
            src='/media/icons/duotune/finance/fin009.svg'
            className='svg-icon-3x svg-icon-primary mb-2'
          />
          <span className='fs-5 fw-bold text-gray-800 mb-0'>Accounting</span>
          <span className='fs-7 text-gray-400'>eCommerce</span>
        </a>
      </div>

      <div className='col-6'>
        <a
          href='#'
          className='d-flex flex-column flex-center h-100 p-6 bg-hover-light border-bottom'
        >
          <img
            src='/media/icons/duotune/communication/com010.svg'
            className='svg-icon-3x svg-icon-primary mb-2'
          />
          <span className='fs-5 fw-bold text-gray-800 mb-0'>Administration</span>
          <span className='fs-7 text-gray-400'>Console</span>
        </a>
      </div>

      <div className='col-6'>
        <a href='#' className='d-flex flex-column flex-center h-100 p-6 bg-hover-light border-end'>
          <img
            src='/media/icons/duotune/abstract/abs042.svg'
            className='svg-icon-3x svg-icon-primary mb-2'
          />
          <span className='fs-5 fw-bold text-gray-800 mb-0'>Projects</span>
          <span className='fs-7 text-gray-400'>Pending Tasks</span>
        </a>
      </div>

      <div className='col-6'>
        <a href='#' className='d-flex flex-column flex-center h-100 p-6 bg-hover-light'>
          <img
            src='/media/icons/duotune/finance/fin006.svg'
            className='svg-icon-3x svg-icon-primary mb-2'
          />
          <span className='fs-5 fw-bold text-gray-800 mb-0'>Customers</span>
          <span className='fs-7 text-gray-400'>Latest cases</span>
        </a>
      </div>
    </div>

    <div className='py-2 text-center border-top'>
      <Link href='/crafted/pages/profile' className='btn btn-color-gray-600 btn-active-color-primary'>
        View All <img src='/media/icons/duotune/arrows/arr064.svg' className='svg-icon-5' />
      </Link>
    </div>
  </div>
)

export {QuickLinks}
