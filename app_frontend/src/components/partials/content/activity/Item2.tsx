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
import Image from "next/image";

const Item2: FC = () => {
  return (
    <div className='timeline-item'>
      <div className='timeline-line w-40px'></div>

      <div className='timeline-icon symbol symbol-circle symbol-40px'>
        <div className='symbol-label bg-light'>
          <Image
            src='/media/icons/duotune/communication/com009.svg'
            className='svg-icon-2 svg-icon-gray-500'
           alt={"com009"}/>
        </div>
      </div>

      <div className='timeline-content mb-10 mt-n2'>
        <div className='overflow-auto pe-3'>
          <div className='fs-5 fw-bold mb-2'>
            Invitation for crafting engaging designs that speak human workshop
          </div>

          <div className='d-flex align-items-center mt-1 fs-6'>
            <div className='text-muted me-2 fs-7'>Sent at 4:23 PM by</div>

            <div
              className='symbol symbol-circle symbol-25px'
              data-bs-toggle='tooltip'
              data-bs-boundary='window'
              data-bs-placement='top'
              title='Alan Nilson'
            >
              <img src={'/media/avatars/300-1.jpg'} alt='img' />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export {Item2}
