import React, {FC} from 'react'

const DemosToggleDrawer: FC = () => (
  <button
    id='kt_engage_demos_toggle'
    className='engage-demos-toggle btn btn-flex h-35px bg-body btn-color-gray-700 btn-active-color-gray-900 shadow-sm fs-6 px-4 rounded-top-0'
    title={`Check out ${process.env.REACT_APP_THEME_NAME} more demos`}
    data-bs-toggle='tooltip'
    data-bs-placement='left'
    data-bs-dismiss='click'
    data-bs-trigger='hover'
  >
    <span id='kt_engage_demos_label'>Demos</span>
  </button>
)

export {DemosToggleDrawer}
