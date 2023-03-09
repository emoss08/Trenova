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

type Props = {
  className?: string
}
const TilesWidget4 = ({className}: Props) => {
  return (
    <div className={clsx('card h-150px', className)}>
      <div className='card-body d-flex align-items-center justify-content-between flex-wrap'>
        <div className='me-2'>
          <h2 className='fw-bold text-gray-800 mb-3'>Create CRM Reports</h2>

          <div className='text-muted fw-semibold fs-6'>
            Generate the latest CRM report for company projects
          </div>
        </div>
        <a
          href='#'
          className='btn btn-primary fw-semibold'
          data-bs-toggle='modal'
          data-bs-target='#mt_modal_create_campaign'
        >
          Start Now
        </a>
      </div>
    </div>
  )
}

export {TilesWidget4}
