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

import React, {FC} from 'react'
import {
  ListsWidget1,
  ListsWidget2,
  ListsWidget3,
  ListsWidget4,
  ListsWidget5,
  ListsWidget6,
  ListsWidget7,
  ListsWidget8,
} from '../../../../_monta/partials/widgets'

const Lists: FC = () => {
  return (
    <>
      {/* begin::Row */}
      <div className='row g-5 g-xl-8'>
        <div className='col-xl-4'>
          <ListsWidget1 className='card-xl-stretch mb-xl-8' />
        </div>
        <div className='col-xl-4'>
          <ListsWidget2 className='card-xl-stretch mb-xl-8' />
        </div>
        <div className='col-xl-4'>
          <ListsWidget3 className='card-xl-stretch mb-5 mb-xl-8' />
        </div>
      </div>
      {/* end::Row */}

      {/* begin::Row */}
      <div className='row g-5 g-xl-8'>
        <div className='col-xl-4'>
          <ListsWidget4 className='card-xl-stretch mb-xl-8' />
        </div>
        <div className='col-xl-4'>
          <ListsWidget5 className='card-xl-stretch mb-xl-8' />
        </div>
        <div className='col-xl-4'>
          <ListsWidget6 className='card-xl-stretch mb-5 mb-xl-8' />
        </div>
      </div>
      {/* end::Row */}

      {/* begin::Row */}
      <div className='row g-5 g-xl-8'>
        <div className='col-xl-6'>
          <ListsWidget7 className='card-xl-stretch mb-xl-8' />
        </div>
        <div className='col-xl-6'>
          <ListsWidget8 className='card-xl-stretch mb-5 mb-xl-8' />
        </div>
      </div>
      {/* end::Row */}
    </>
  )
}

export {Lists}
