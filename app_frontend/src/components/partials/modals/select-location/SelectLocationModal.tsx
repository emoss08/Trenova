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

import React, {Dispatch, SetStateAction, useState, useEffect} from 'react'
import {Modal} from 'react-bootstrap'

type Props = {
  data: {location: string; setLocation: Dispatch<SetStateAction<string>>}
  show: boolean
  handleClose: () => void
}

const SelectLocationModal: React.FC<Props> = ({show, handleClose, data}) => {
  useEffect(() => {
    initMap()
  }, [])

  const [location, setLocation] = useState(data.location)
  const dissmissLocation = () => {
    setLocation(data.location)
    handleClose()
  }
  const applyLocation = () => {
    data.setLocation(location)
    handleClose()
  }
  const initMap = () => {}

  return (
    <Modal
      className='modal fade'
      id='mt_modal_select_location'
      data-backdrop='static'
      tabIndex={-1}
      role='dialog'
      show={show}
      dialogClassName='modal-xl'
      aria-hidden='true'
      onHide={dissmissLocation}
    >
      <div className='modal-content'>
        <div className='modal-header'>
          <h5 className='modal-title'>Select Location</h5>

          <div
            className='btn btn-icon btn-sm btn-active-light-primary ms-2'
            onClick={dissmissLocation}
          >
            <img src='/media/icons/duotune/arrows/arr061.svg' className='svg-icon-2x' />
          </div>
        </div>
        <div className='modal-body'>
          <input type='text' value={location} onChange={(e) => setLocation(e.target.value)} />
          <div id='mt_modal_select_location_map' className='map h-450px'></div>
        </div>
        <div className='modal-footer'>
          <button type='button' className='btn btn-light-primary' onClick={dissmissLocation}>
            Cancel
          </button>
          <button id='submit' type='button' className='btn btn-primary' onClick={applyLocation}>
            Apply
          </button>
        </div>
      </div>
    </Modal>
  )
}

export {SelectLocationModal}
