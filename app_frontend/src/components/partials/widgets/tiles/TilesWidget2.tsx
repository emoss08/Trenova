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
import {toAbsoluteUrl} from '../../../helpers'

type Props = {
  className?: string
  bgColor?: string
  title?: string
  title2?: string
}
const TilesWidget2 = ({
  className,
  bgColor = '#663259',
  title = 'Create SaaS',
  title2 = 'Based Reports',
}: Props) => {
  return (
    <div
      className={clsx('card h-175px bgi-no-repeat bgi-size-contain', className)}
      style={{
        backgroundColor: bgColor,
        backgroundPosition: 'right',
        backgroundImage: `url("${toAbsoluteUrl('/media/svg/misc/taieri.svg')}")`,
      }}
    >
      <div className='card-body d-flex flex-column justify-content-between'>
        <h2 className='text-white fw-bold mb-5'>
          {title} <br /> {title2}{' '}
        </h2>

        <div className='m-0'>
          <a
            href='#'
            className='btn btn-danger fw-semibold px-6 py-3'
            data-bs-toggle='modal'
            data-bs-target='#mt_modal_create_app'
          >
            Create Campaign
          </a>
        </div>
      </div>
    </div>
  )
}

export {TilesWidget2}
