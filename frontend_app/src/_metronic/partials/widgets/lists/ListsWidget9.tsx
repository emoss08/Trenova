/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import React from 'react'
import {KTSVG, toAbsoluteUrl} from '../../../helpers'
import {Dropdown1} from '../../content/dropdown/Dropdown1'

type Props = {
  className: string
}

const ListsWidget9: React.FC<Props> = ({className}) => {
  return (
    <div className={clsx('card', className)}>
      {/* begin::Header */}
      <div className='card-header align-items-center border-0 mt-3'>
        <h3 className='card-title align-items-start flex-column'>
          <span className='fw-bolder text-dark fs-3'>My Competitors</span>
          <span className='text-gray-400 mt-2 fw-bold fs-6'>More than 400+ new members</span>
        </h3>
        <div className='card-toolbar'>
          {/* begin::Menu */}
          <button
            type='button'
            className='btn btn-clean btn-sm btn-icon btn-icon-primary btn-active-light-primary me-n3'
            data-kt-menu-trigger='click'
            data-kt-menu-placement='bottom-end'
            data-kt-menu-flip='top-end'
          >
            <KTSVG path='/media/icons/duotune/general/gen024.svg' className='svg-icon-2' />
          </button>
          <Dropdown1 />
          {/* end::Menu */}
        </div>
      </div>
      {/* end::Header */}
      {/* begin::Body */}
      <div className='card-body pt-5'>
        {/*begin::Item*/}
        <div className='d-flex mb-7'>
          {/*begin::Symbol*/}
          <div className='symbol symbol-60px symbol-2by3 flex-shrink-0 me-4'>
            <img src={toAbsoluteUrl('/media/stock/600x400/img-3.jpg')} className='mw-100' alt='' />
          </div>
          {/*end::Symbol*/}
          {/*begin::Section*/}
          <div className='d-flex align-items-center flex-wrap flex-grow-1 mt-n2 mt-lg-n1'>
            {/*begin::Title*/}
            <div className='d-flex flex-column flex-grow-1 my-lg-0 my-2 pe-3'>
              <a href='#' className='fs-5 text-gray-800 text-hover-primary fw-bolder'>
                Cup &amp; Green
              </a>
              <span className='text-gray-400 fw-bold fs-7 my-1'>Study highway types</span>
              <span className='text-gray-400 fw-bold fs-7'>
                By:
                <a href='#' className='text-primary fw-bold'>
                  CoreAd
                </a>
              </span>
            </div>
            {/*end::Title*/}
            {/*begin::Info*/}
            <div className='text-end py-lg-0 py-2'>
              <span className='text-gray-800 fw-boldest fs-3'>24,900</span>
              <span className='text-gray-400 fs-7 fw-bold d-block'>Sales</span>
            </div>
            {/*end::Info*/}
          </div>
          {/*end::Section*/}
        </div>
        {/*end::Item*/}
        {/*begin::Item*/}
        <div className='d-flex mb-7'>
          {/*begin::Symbol*/}
          <div className='symbol symbol-60px symbol-2by3 flex-shrink-0 me-4'>
            <img src={toAbsoluteUrl('/media/stock/600x400/img-4.jpg')} className='mw-100' alt='' />
          </div>
          {/*end::Symbol*/}
          {/*begin::Section*/}
          <div className='d-flex align-items-center flex-wrap flex-grow-1 mt-n2 mt-lg-n1'>
            {/*begin::Title*/}
            <div className='d-flex flex-column flex-grow-1 my-lg-0 my-2 pe-3'>
              <a href='#' className='fs-5 text-gray-800 text-hover-primary fw-bolder'>
                Yellow Hearts
              </a>
              <span className='text-gray-400 fw-bold fs-7 my-1'>Study highway types</span>
              <span className='text-gray-400 fw-bold fs-7'>
                By:
                <a href='#' className='text-primary fw-bold'>
                  KeenThemes
                </a>
              </span>
            </div>
            {/*end::Title*/}
            {/*begin::Info*/}
            <div className='text-end py-lg-0 py-2'>
              <span className='text-gray-800 fw-boldest fs-3'>70,380</span>
              <span className='text-gray-400 fs-7 fw-bold d-block'>Sales</span>
            </div>
            {/*end::Info*/}
          </div>
          {/*end::Section*/}
        </div>
        {/*end::Item*/}
        {/*begin::Item*/}
        <div className='d-flex mb-7'>
          {/*begin::Symbol*/}
          <div className='symbol symbol-60px symbol-2by3 flex-shrink-0 me-4'>
            <img src={toAbsoluteUrl('/media/stock/600x400/img-5.jpg')} className='mw-100' alt='' />
          </div>
          {/*end::Symbol*/}
          {/*begin::Section*/}
          <div className='d-flex align-items-center flex-wrap flex-grow-1 mt-n2 mt-lg-n1'>
            {/*begin::Title*/}
            <div className='d-flex flex-column flex-grow-1 my-lg-0 my-2 pe-3'>
              <a href='#' className='fs-5 text-gray-800 text-hover-primary fw-bolder'>
                Nike &amp; Blue
              </a>
              <span className='text-gray-400 fw-bold fs-7 my-1'>Study highway types</span>
              <span className='text-gray-400 fw-bold fs-7'>
                By:
                <a href='#' className='text-primary fw-bold'>
                  Invision Inc.
                </a>
              </span>
            </div>
            {/*end::Title*/}
            {/*begin::Info*/}
            <div className='text-end py-lg-0 py-2'>
              <span className='text-gray-800 fw-boldest fs-3'>7,200</span>
              <span className='text-gray-400 fs-7 fw-bold d-block'>Sales</span>
            </div>
            {/*end::Info*/}
          </div>
          {/*end::Section*/}
        </div>
        {/*end::Item*/}
        {/*begin::Item*/}
        <div className='d-flex mb-7'>
          {/*begin::Symbol*/}
          <div className='symbol symbol-60px symbol-2by3 flex-shrink-0 me-4'>
            <img src={toAbsoluteUrl('/media/stock/600x400/img-6.jpg')} className='mw-100' alt='' />
          </div>
          {/*end::Symbol*/}
          {/*begin::Section*/}
          <div className='d-flex align-items-center flex-wrap flex-grow-1 mt-n2 mt-lg-n1'>
            {/*begin::Title*/}
            <div className='d-flex flex-column flex-grow-1 my-lg-0 my-2 pe-3'>
              <a href='#' className='fs-5 text-gray-800 text-hover-primary fw-bolder'>
                Red Boots
              </a>
              <span className='text-gray-400 fw-bold fs-7 my-1'>Study highway types</span>
              <span className='text-gray-400 fw-bold fs-7'>
                By:
                <a href='#' className='text-primary fw-bold'>
                  Figma Studio
                </a>
              </span>
            </div>
            {/*end::Title*/}
            {/*begin::Info*/}
            <div className='text-end py-lg-0 py-2'>
              <span className='text-gray-800 fw-boldest fs-3'>36,450</span>
              <span className='text-gray-400 fs-7 fw-bold d-block'>Sales</span>
            </div>
            {/*end::Info*/}
          </div>
          {/*end::Section*/}
        </div>
        {/*end::Item*/}
        {/*begin::Item*/}
        <div className='d-flex'>
          {/*begin::Symbol*/}
          <div className='symbol symbol-60px symbol-2by3 flex-shrink-0 me-4'>
            <img src={toAbsoluteUrl('/media/stock/600x400/img-7.jpg')} className='mw-100' alt='' />
          </div>
          {/*end::Symbol*/}
          {/*begin::Section*/}
          <div className='d-flex align-items-center flex-wrap flex-grow-1 mt-n2 mt-lg-n1'>
            {/*begin::Title*/}
            <div className='d-flex flex-column flex-grow-1 my-lg-0 my-2 pe-3'>
              <a href='#' className='fs-5 text-gray-800 text-hover-primary fw-bolder'>
                Desserts platter
              </a>
              <span className='text-gray-400 fw-bold fs-7 my-1'>Food trends &amp; reviews</span>
              <span className='text-gray-400 fw-bold fs-7'>
                By:
                <a href='#' className='text-primary fw-bold'>
                  Figma Studio
                </a>
              </span>
            </div>
            {/*end::Title*/}
            {/*begin::Info*/}
            <div className='text-end py-lg-0 py-2'>
              <span className='text-gray-800 fw-boldest fs-3'>64,753</span>
              <span className='text-gray-400 fs-7 fw-bold d-block'>Sales</span>
            </div>
            {/*end::Info*/}
          </div>
          {/*end::Section*/}
        </div>
        {/*end::Item*/}
      </div>
      {/* end::Body */}
    </div>
  )
}

export {ListsWidget9}
