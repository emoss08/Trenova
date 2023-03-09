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
  bgImage?: string
  title?: string
}
const TilesWidget1 = ({
  className,
  bgImage = toAbsoluteUrl('/media/stock/600x400/img-75.jpg'),
  title = 'Properties',
}: Props) => {
  return (
    <div
      className={clsx('card h-150px bgi-no-repeat bgi-size-cover', className)}
      style={{
        backgroundImage: `url("${bgImage}")`,
      }}
    >
      <div className='card-body p-6'>
        <a
          href='#'
          className='text-black text-hover-primary fw-bold fs-2'
          data-bs-toggle='modal'
          data-bs-target='#mt_modal_create_app'
        >
          {title}
        </a>
      </div>
    </div>
  )
}

export {TilesWidget1}
