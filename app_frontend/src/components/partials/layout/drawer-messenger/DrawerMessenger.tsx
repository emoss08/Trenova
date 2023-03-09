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
import Image from "next/image";

const DrawerMessenger: FC = () => (
  <div
    id='mt_drawer_chat'
    className='bg-white'
    data-mt-drawer='true'
    data-mt-drawer-name='chat'
    data-mt-drawer-activate='true'
    data-mt-drawer-overlay='true'
    data-mt-drawer-width="{default:'300px', 'md': '500px'}"
    data-mt-drawer-direction='end'
    data-mt-drawer-toggle='#mt_drawer_chat_toggle'
    data-mt-drawer-close='#mt_drawer_chat_close'
  >
    <div className='card w-100 rounded-0' id='mt_drawer_chat_messenger'>
      <div className='card-header pe-5' id='mt_drawer_chat_messenger_header'>
        <div className='card-title'>
          <div className='d-flex justify-content-center flex-column me-3'>
            <a href='#' className='fs-4 fw-bolder text-gray-900 text-hover-primary me-1 mb-2 lh-1'>
              Brian Cox
            </a>
            <div className='mb-0 lh-1'>
              <span className='badge badge-success badge-circle w-10px h-10px me-1'></span>
              <span className='fs-7 fw-bold text-gray-400'>Active</span>
            </div>
          </div>
        </div>
        <div className='card-toolbar'>
          <div className='me-2'>
            <button
              className='btn btn-sm btn-icon btn-active-light-primary'
              data-mt-menu-trigger='click'
              data-mt-menu-placement='bottom-end'
              data-mt-menu-flip='top-end'
            >
              <i className='bi bi-three-dots fs-3'></i>
            </button>
          </div>
          <div className='btn btn-sm btn-icon btn-active-light-primary' id='mt_drawer_chat_close'>
            <Image src='/media/icons/duotune/arrows/arr061.svg' className='svg-icon-2'  alt={"arr061"}/>
          </div>
        </div>
      </div>
    </div>
  </div>
)

export {DrawerMessenger}
