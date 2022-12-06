/* eslint-disable jsx-a11y/anchor-is-valid */
import {FC, useEffect, useState} from 'react'
import {MTSVG} from '../../../../helpers'

const ToolbarAccounting: FC = () => {
  const [progress, setProgress] = useState<string>('1')
  const [filter, setFilter] = useState<string>('1')

  useEffect(() => {
    document.body.setAttribute('data-kt-app-toolbar-fixed', 'true')
  }, [])

  return (
    <>
      <div className='d-flex align-items-center me-5'>
        {/* begin::Input group */}
        <div className='d-flex align-items-center flex-shrink-0'>
          {/* begin::Label */}
          <span className='fs-7 text-gray-700 fw-bold pe-3 d-none d-md-block'>Actions:</span>
          {/* end::Label */}

          {/* begin::Actions */}
          <div className='d-flex flex-shrink-0'>
            {/* begin::Button */}
            <div
              data-bs-toggle='tooltip'
              data-bs-placement='top'
              data-bs-trigger='hover'
              title='Add a team member'
            >
              <a href='#' className='btn btn-sm btn-icon btn-active-color-success'>
                <MTSVG path='/media/icons/duotune/general/gen035.svg' className='svg-icon-2x' />
              </a>
            </div>
            {/* end::Button */}

            {/* begin::Button */}
            <div
              data-bs-toggle='tooltip'
              data-bs-placement='top'
              data-bs-trigger='hover'
              title='Create new account'
            >
              <a href='#' className='btn btn-sm btn-icon btn-active-color-success'>
                <MTSVG path='/media/icons/duotune/general/gen037.svg' className='svg-icon-2x' />
              </a>
            </div>
            {/* end::Button */}

            {/* begin::Button */}
            <div
              data-bs-toggle='tooltip'
              data-bs-placement='top'
              data-bs-trigger='hover'
              title='Invite friends'
            >
              <a href='#' className='btn btn-sm btn-icon btn-active-color-success'>
                <MTSVG path='/media/icons/duotune/general/gen023.svg' className='svg-icon-2x' />
              </a>
            </div>
            {/* end::Button */}
          </div>
          {/* end::Actions */}
        </div>
        {/* end::Input group */}

        {/* begin::Input group */}
        <div className='d-flex align-items-center flex-shrink-0'>
          {/* begin::Desktop separartor */}
          <div className='bullet bg-secondary h-35px w-1px mx-5'></div>
          {/* end::Desktop separartor */}

          {/* begin::Label */}
          <span className='fs-7 text-gray-700 fw-bold pe-4 ps-1 d-none d-md-block'>Progress:</span>
          {/* end::Label */}

          <div className='progress w-100px w-xl-150px w-xxl-300px h-25px bg-light-success'>
            <div
              className='progress-bar rounded bg-success fs-7 fw-bold'
              role='progressbar'
              style={{width: '72%'}}
              aria-valuenow={72}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              72%
            </div>
          </div>
        </div>
        {/* end::Input group */}
        {/* end::Toolbar start */}
      </div>
      {/* begin::Toolbar end */}
      <div className='d-flex align-items-center'>
        {/* begin::Input group */}
        <div className='me-3'>
          {/* begin::Select */}
          <select
            className='form-select form-select-sm form-select-solid'
            data-control='select2'
            data-placeholder='Latest'
            data-hide-search='true'
            value={progress}
            onChange={(e) => setProgress(e.target.value)}
          >
            <option value=''></option>
            <option value='1'>Today 16 Feb</option>
            <option value='2'>In Progress</option>
            <option value='3'>Done</option>
          </select>
          {/* end::Select */}
        </div>
        {/* end::Input group- */}

        {/* begin::Input group- */}
        <div className='m-0'>
          {/* begin::Select */}
          <select
            className='form-select form-select-sm form-select-solid w-md-125px'
            data-control='select2'
            data-placeholder='Filters'
            data-hide-search='true'
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          >
            <option value=''></option>
            <option value='1'>Filters</option>
            <option value='2'>In Progress</option>
            <option value='3'>Done</option>
          </select>
          {/* end::Content */}
        </div>
        {/* end::Input group- */}
      </div>
    </>
  )
}

export {ToolbarAccounting}
