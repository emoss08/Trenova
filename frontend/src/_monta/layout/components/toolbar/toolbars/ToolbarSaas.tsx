/* eslint-disable jsx-a11y/anchor-is-valid */
import {FC, useEffect, useState} from 'react'
import {MTSVG} from '../../../../helpers'

const ToolbarSaas: FC = () => {
  const [progress, setProgress] = useState<string>('1')
  useEffect(() => {
    document.body.setAttribute('data-kt-app-toolbar-fixed', 'true')
  }, [])

  return (
    <div className='d-flex align-items-center gap-2'>
      {/* begin::Action wrapper */}
      <div className='d-flex align-items-center'>
        {/* begin::Label */}
        <span className='fs-7 fw-bold text-gray-700 pe-4 text-nowrap d-none d-md-block'>
          Sort By:
        </span>
        {/* end::Label */}

        {/* begin::Select */}
        <select
          className='form-select form-select-sm form-select-solid w-100px w-xxl-125px'
          data-control='select2'
          data-placeholder='Latest'
          data-hide-search='true'
          onChange={(e) => setProgress(e.target.value)}
          value={progress}
        >
          <option value=''></option>
          <option value='1'>Latest</option>
          <option value='2'>In Progress</option>
          <option value='3'>Done</option>
        </select>
        {/* end::Select */}
      </div>
      {/* end::Action wrapper */}

      {/* begin::Action wrapper */}
      <div className='d-flex align-items-center'>
        {/* begin::Separartor */}
        <div className='bullet bg-secondary h-35px w-1px mx-5'></div>
        {/* end::Separartor */}

        {/* begin::Label */}
        <span className='fs-7 text-gray-700 fw-bold'>Impact Level:</span>
        {/* end::Label */}

        {/* begin::NoUiSlider */}
        <div className='d-flex align-items-center ps-4'>
          <div
            id='kt_app_toolbar_slider'
            className='noUi-target noUi-target-success w-75px w-xxl-150px noUi-sm'
          ></div>

          <span
            id='kt_app_toolbar_slider_value'
            className='d-flex flex-center bg-light-success rounded-circle w-35px h-35px ms-4 fs-7 fw-bold text-success'
            data-bs-toggle='tooltip'
            data-bs-placement='top'
            title='Set impact level'
          ></span>
        </div>
        {/* end::NoUiSlider */}

        {/* begin::Separartor */}
        <div className='bullet bg-secondary h-35px w-1px mx-5'></div>
        {/* end::Separartor */}
      </div>
      {/* end::Action wrapper */}

      {/* begin::Action wrapper */}
      <div className='d-flex align-items-center'>
        {/* begin::Label */}
        <span className='fs-7 text-gray-700 fw-bold pe-3 d-none d-md-block'>Quick Tools:</span>
        {/* end::Label */}

        {/* begin::Actions */}
        <div className='d-flex'>
          {/* begin::Action */}
          <a
            href='#'
            className='btn btn-sm btn-icon btn-icon-muted btn-active-icon-success'
            data-bs-toggle='tooltip'
            data-bs-trigger='hover'
            data-bs-placement='top'
            title='Add new page'
          >
            <MTSVG path='/media/icons/duotune/files/fil003.svg' className='svg-icon-2x' />
          </a>
          {/* end::Action */}

          {/* begin::Action */}
          <a
            href='#'
            className='btn btn-sm btn-icon btn-icon-muted btn-active-icon-success'
            data-bs-toggle='tooltip'
            data-bs-trigger='hover'
            data-bs-placement='top'
            title='Add new category'
          >
            <MTSVG path='/media/icons/duotune/files/fil005.svg' className='svg-icon-2x' />
          </a>
          {/* end::Action */}

          {/* begin::Action */}
          <a
            href='#'
            className='btn btn-sm btn-icon btn-icon-muted btn-active-icon-success'
            data-bs-toggle='tooltip'
            data-bs-trigger='hover'
            data-bs-placement='top'
            title='Add new section'
          >
            <MTSVG path='/media/icons/duotune/files/fil024.svg' className='svg-icon-2x' />
          </a>
          {/* end::Action */}
        </div>
        {/* end::Actions */}
      </div>
      {/* end::Action wrapper */}
    </div>
  )
}

export {ToolbarSaas}
