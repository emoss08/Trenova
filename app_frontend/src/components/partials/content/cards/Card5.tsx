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
import {toAbsoluteUrl, KTSVG} from '../../../helpers'
import {Dropdown1} from '../dropdown/Dropdown1'

type Props = {
  image: string
  title: string
  description: string
  status: 'up' | 'down'
  statusValue: number
  statusDesc: string
  progress: number
  progressType: string
}

const Card5: FC<Props> = ({
  image,
  title,
  description,
  status,
  statusValue,
  statusDesc,
  progress,
  progressType,
}) => {
  return (
    <div className='card h-100'>
      <div className='card-header flex-nowrap border-0 pt-9'>
        <div className='card-title m-0'>
          <div className='symbol symbol-45px w-45px bg-light me-5'>
            <img src={toAbsoluteUrl(image)} alt='Metronic' className='p-3' />
          </div>

          <a href='#' className='fs-4 fw-bold text-hover-primary text-gray-600 m-0'>
            {title}
          </a>
        </div>

        <div className='card-toolbar m-0'>
          <button
            type='button'
            className='btn btn-clean btn-sm btn-icon btn-icon-primary btn-active-light-primary me-n3'
            data-mt-menu-trigger='click'
            data-mt-menu-placement='bottom-end'
            data-mt-menu-flip='top-end'
          >
            <KTSVG
              path='/media/icons/duotune/general/gen024.svg'
              className='svg-icon-3 svg-icon-primary'
            />
          </button>

          <Dropdown1 />
        </div>
      </div>

      <div className='card-body d-flex flex-column px-9 pt-6 pb-8'>
        <div className='fs-2tx fw-bolder mb-3'>{description}</div>

        <div className='d-flex align-items-center flex-wrap mb-5 mt-auto fs-6'>
          {status === 'up' && (
            <KTSVG
              path='/media/icons/duotune/arrows/arr007.svg'
              className='svg-icon-3 me-1 svg-icon-success'
            />
          )}

          {status === 'down' && (
            <KTSVG
              path='/media/icons/duotune/arrows/arr006.svg'
              className='svg-icon-3 me-1 svg-icon-danger'
            />
          )}

          <div className={`fw-bolder me-2 ` + (status === 'up' ? 'text-success' : 'text-danger')}>
            {status === 'up' ? '+' : '-'}
            {statusValue}%
          </div>

          <div className='fw-bold text-gray-400'>{statusDesc}</div>
        </div>

        <div className='d-flex align-items-center fw-bold'>
          <span className='badge bg-light text-gray-700 px-3 py-2 me-2'>{progress}%</span>
          <span className='text-gray-400 fs-7'>{progressType}</span>
        </div>
      </div>
    </div>
  )
}

export {Card5}
