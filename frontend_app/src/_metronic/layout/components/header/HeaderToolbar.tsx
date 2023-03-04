/* eslint-disable jsx-a11y/anchor-is-valid */
import {useState, useEffect} from 'react'
import noUiSlider from 'nouislider'
import {useLayout} from '../../core'
import {KTSVG} from '../../../helpers'
import {DefaultTitle} from './page-title/DefaultTitle'
import {ThemeModeSwitcher} from '../../../partials'

const HeaderToolbar = () => {
  const {classes} = useLayout()
  const [status, setStatus] = useState<string>('1')

  useEffect(() => {
    const rangeSlider = document.querySelector('#kt_toolbar_slider')
    const rangeSliderValueElement = document.querySelector('#kt_toolbar_slider_value')

    if (!rangeSlider || !rangeSliderValueElement) {
      return
    }

    // @ts-ignore
    noUiSlider.create(rangeSlider, {
      start: [5],
      connect: [true, false],
      step: 1,
      format: {
        to: function (value) {
          const val = +value
          return Math.round(val).toString()
        },
        from: function (value) {
          return value
        },
      },
      range: {
        min: [1],
        max: [10],
      },
    })

    // @ts-ignore
    rangeSlider.noUiSlider.on('update', function (values, handle) {
      rangeSliderValueElement.innerHTML = values[handle]
    })

    const handle = rangeSlider.querySelector('.noUi-handle')
    if (handle) {
      handle.setAttribute('tabindex', '0')
    }

    // @ts-ignore
    handle.addEventListener('click', function () {
      // @ts-ignore
      this.focus()
    })

    // @ts-ignore
    handle.addEventListener('keydown', function (event) {
      // @ts-ignore
      const value = Number(rangeSlider.noUiSlider.get())
      // @ts-ignore
      switch (event.which) {
        case 37:
          // @ts-ignore
          rangeSlider.noUiSlider.set(value - 1)
          break
        case 39:
          // @ts-ignore
          rangeSlider.noUiSlider.set(value + 1)
          break
      }
    })
    return () => {
      // @ts-ignore
      rangeSlider.noUiSlider.destroy()
    }
  }, [])

  return (
    <div className='toolbar d-flex align-items-stretch'>
      {/* begin::Toolbar container */}
      <div
        className={`${classes.headerContainer.join(
          ' '
        )} py-6 py-lg-0 d-flex flex-column flex-lg-row align-items-lg-stretch justify-content-lg-between`}
      >
        <DefaultTitle />
        <div className='d-flex align-items-stretch overflow-auto pt-3 pt-lg-0'>
          {/* begin::Action wrapper */}
          <div className='d-flex align-items-center'>
            {/* begin::Label */}
            <span className='fs-7 fw-bolder text-gray-700 pe-4 text-nowrap d-none d-xxl-block'>
              Sort By:
            </span>
            {/* end::Label */}

            {/* begin::Select */}
            <select
              className='form-select form-select-sm form-select-solid w-100px w-xxl-125px'
              data-control='select2'
              data-placeholder='Latest'
              data-hide-search='true'
              defaultValue={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value=''></option>
              <option value='1'>Latest</option>
              <option value='2'>In Progress</option>
              <option value='3'>Done</option>
            </select>
            {/* end::Select  */}
          </div>
          {/* end::Action wrapper */}

          {/* begin::Action wrapper */}
          <div className='d-flex align-items-center'>
            {/* begin::Separartor */}
            <div className='bullet bg-secondary h-35px w-1px mx-5'></div>
            {/* end::Separartor */}

            {/* begin::Label */}
            <span className='fs-7 text-gray-700 fw-bolder d-none d-sm-block'>
              Impact <span className='d-none d-xxl-inline'>Level</span>:
            </span>
            {/* end::Label */}

            {/* begin::NoUiSlider */}
            <div className='d-flex align-items-center ps-4' id='kt_toolbar'>
              <div
                id='kt_toolbar_slider'
                className='noUi-target noUi-target-primary w-75px w-xxl-150px noUi-sm noUi-ltr noUi-horizontal noUi-txt-dir-ltr'
              ></div>

              <span
                id='kt_toolbar_slider_value'
                className='d-flex flex-center bg-light-primary rounded-circle w-35px h-35px ms-4 fs-7 fw-bolder text-primary'
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
            <span className='fs-7 text-gray-700 fw-bolder pe-3 d-none d-xxl-block'>
              Quick Tools:
            </span>
            {/* end::Label */}

            {/* begin::Actions */}
            <div className='d-flex'>
              {/* begin::Action */}
              <a
                href='#'
                className='btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary'
                data-bs-toggle='modal'
                data-bs-target='#kt_modal_invite_friends'
              >
                <KTSVG path='/media/icons/duotune/files/fil003.svg' className='svg-icon-1' />
              </a>
              {/* end::Action */}

              {/* begin::Notifications */}
              <div className='d-flex align-items-center'>
                {/* begin::Menu- wrapper */}
                <a href='#' className='btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary'>
                  <KTSVG path='/media/icons/duotune/files/fil005.svg' className='svg-icon-1' />
                </a>
                {/* end::Menu wrapper */}
              </div>
              {/* end::Notifications */}

              {/* begin::Quick links */}
              <div className='d-flex align-items-center'>
                {/* begin::Menu wrapper */}
                <a href='#' className='btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary'>
                  <KTSVG path='/media/icons/duotune/files/fil010.svg' className='svg-icon-1' />
                </a>
                {/* end::Menu wrapper */}
              </div>
              {/* end::Quick links */}

              {/* begin::Theme mode */}
              <div className='d-flex align-items-center'>
                <ThemeModeSwitcher toggleBtnClass='btn btn-sm btn-icon btn-icon-muted btn-active-icon-primary' />
              </div>
              {/* end::Theme mode */}
            </div>
            {/* end::Actions */}
          </div>
          {/* end::Action wrapper */}
        </div>
        {/* end::Toolbar container */}
      </div>
    </div>
  )
}

export {HeaderToolbar}
