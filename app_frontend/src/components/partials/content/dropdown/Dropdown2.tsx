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
import {FC} from 'react'

const Dropdown2: FC = () => {
  return (
    <div
      className='menu menu-sub menu-sub-dropdown menu-column menu-rounded menu-gray-600 menu-state-bg-light-primary fw-bold w-200px'
      data-mt-menu='true'
    >
      <div className='menu-item px-3'>
        <div className='menu-content fs-6 text-dark fw-bolder px-3 py-4'>Quick Actions</div>
      </div>

      <div className='separator mb-3 opacity-75'></div>

      <div className='menu-item px-3'>
        <a href='#' className='menu-link px-3'>
          New Ticket
        </a>
      </div>

      <div className='menu-item px-3'>
        <a href='#' className='menu-link px-3'>
          New Customer
        </a>
      </div>

      <div
        className='menu-item px-3'
        data-mt-menu-trigger='hover'
        data-mt-menu-placement='right-start'
        data-mt-menu-flip='left-start, top'
      >
        <a href='#' className='menu-link px-3'>
          <span className='menu-title'>New Group</span>
          <span className='menu-arrow'></span>
        </a>

        <div className='menu-sub menu-sub-dropdown w-175px py-4'>
          <div className='menu-item px-3'>
            <a href='#' className='menu-link px-3'>
              Admin Group
            </a>
          </div>

          <div className='menu-item px-3'>
            <a href='#' className='menu-link px-3'>
              Staff Group
            </a>
          </div>

          <div className='menu-item px-3'>
            <a href='#' className='menu-link px-3'>
              Member Group
            </a>
          </div>
        </div>
      </div>

      <div className='menu-item px-3'>
        <a href='#' className='menu-link px-3'>
          New Contact
        </a>
      </div>

      <div className='separator mt-3 opacity-75'></div>

      <div className='menu-item px-3'>
        <div className='menu-content px-3 py-3'>
          <a className='btn btn-primary  btn-sm px-4' href='#'>
            Generate Reports
          </a>
        </div>
      </div>
    </div>
  )
}

export {Dropdown2}
