/* eslint-disable react-hooks/exhaustive-deps */
import clsx from 'clsx'
import {Link} from 'react-router-dom'
import {KTSVG, toAbsoluteUrl} from '../../../helpers'
import {useLayout} from '../../core'
import {HeaderToolbar} from './HeaderToolbar'

export function HeaderWrapper() {
  const {config, classes, attributes} = useLayout()
  const {aside} = config

  return (
    <div
      id='kt_header'
      className={clsx('header', classes.header.join(' '), 'align-items-stretch')}
      {...attributes.headerMenu}
    >
      {/* begin::Brand */}
      <div className='header-brand'>
        {/* begin::Logo */}
        <Link to='/'>
          <img
            alt='Logo'
            src={toAbsoluteUrl('/media/logos/default-dark.svg')}
            className='h-25px h-lg-25px'
          />
        </Link>
        {/* end::Logo */}

        {aside.minimize && (
          <div
            id='kt_aside_toggle'
            className='btn btn-icon w-auto px-0 btn-active-color-primary aside-minimize'
            data-kt-toggle='true'
            data-kt-toggle-state='active'
            data-kt-toggle-target='body'
            data-kt-toggle-name='aside-minimize'
          >
            <KTSVG
              path='/media/icons/duotune/arrows/arr092.svg'
              className='svg-icon-1 me-n1 minimize-default'
            />
            <KTSVG
              path='/media/icons/duotune/arrows/arr076.svg'
              className='svg-icon-1 minimize-active'
            />
          </div>
        )}

        {/* begin::Aside toggle */}
        <div className='d-flex align-items-center d-lg-none ms-n3 me-1' title='Show aside menu'>
          <div
            className='btn btn-icon btn-active-color-primary w-30px h-30px'
            id='kt_aside_mobile_toggle'
          >
            <KTSVG path='/media/icons/duotune/abstract/abs015.svg' className='svg-icon-1' />
          </div>
        </div>
        {/* end::Aside toggle */}
      </div>
      {/* end::Brand */}
      <HeaderToolbar />
    </div>
  )
}
