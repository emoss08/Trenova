/* eslint-disable jsx-a11y/anchor-is-valid */
import {useEffect, useState} from 'react'
import {MTSVG} from '../../../../helpers'

const ToolbarReports = () => {
  const [progress, setProgress] = useState<string>('1')

  useEffect(() => {
    document.body.setAttribute('data-kt-app-toolbar-fixed', 'true')
  }, [])

  return (
    <div className='d-flex align-items-center overflow-auto'>
      {/* begin::Wrapper */}
      <div className='d-flex align-items-center flex-shrink-0'>
        {/* begin::Label */}
        <span className='fs-7 fw-bold text-gray-700 flex-shrink-0 pe-4 d-none d-md-block'>
          Filter By:
        </span>
        {/* end::Label */}

        <div className='flex-shrink-0 '>
          <ul className='nav'>
            <li className='nav-item'>
              <a
                className='nav-link btn btn-sm btn-color-muted btn-active-color-primary btn-active-light active fw-semibold fs-7 px-4 me-1'
                data-bs-toggle='tab'
                href='#'
              >
                Today
              </a>
            </li>

            <li className='nav-item'>
              <a
                className='nav-link btn btn-sm btn-color-muted btn-active-color-primary btn-active-light fw-semibold fs-7 px-4 me-1'
                data-bs-toggle='tab'
                href=''
              >
                Week
              </a>
            </li>

            <li className='nav-item'>
              <a
                className='nav-link btn btn-sm btn-color-muted btn-active-color-primary btn-active-light fw-semibold fs-7 px-4'
                data-bs-toggle='tab'
                href='#'
              >
                Day
              </a>
            </li>
          </ul>
        </div>
      </div>
      {/* end::Wrapper */}

      {/* begin::Separartor */}
      <div className='bullet bg-secondary h-35px w-1px mx-5'></div>
      {/* end::Separartor */}

      {/* begin::Wrapper */}
      <div className='d-flex align-items-center'>
        {/* begin::Label */}
        <span className='fs-7 fw-bold text-gray-700 flex-shrink-0 pe-4 d-none d-md-block'>
          Sort By:
        </span>
        {/* end::Label */}

        {/* begin::Select */}
        <select
          className='form-select form-select-sm w-md-125px form-select-solid'
          data-control='select2'
          data-placeholder='Latest'
          data-hide-search='true'
          value={progress}
          onChange={(e) => setProgress(e.target.value)}
        >
          <option value=''></option>
          <option value='1'>Latest</option>
          <option value='2'>In Progress</option>
          <option value='3'>Done</option>
        </select>
        {/* end::Select */}

        {/* begin::Actions */}
        <div className='d-flex align-items-center ms-3'>
          <button
            type='button'
            className='btn btn-sm btn-icon btn-light-primary me-3'
            data-bs-toggle='tooltip'
            data-bs-placement='top'
            title='Enable grid view'
          >
            <MTSVG
              path='/media/icons/duotune/general/gen025.svg'
              className='svg-icon-2 svg-icon-primary'
            />
          </button>

          <button
            type='button'
            className='btn btn-sm btn-icon btn-light'
            data-bs-toggle='tooltip'
            data-bs-placement='top'
            title='Enable row view'
          >
            <MTSVG
              path='/media/icons/duotune/abstract/abs015.svg'
              className=' svg-icon-2 svg-icon-gray-400'
            />
          </button>
        </div>
        {/* end::Actions */}
      </div>
      {/* end::Wrapper */}
    </div>
  )
}

export {ToolbarReports}
