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

/* eslint-disable react-hooks/exhaustive-deps */
/* eslint-disable jsx-a11y/anchor-is-valid */
import React from 'react'
import Link from "next/link";

const HelpDrawer = () => {
  return (
    <div
      id='mt_help'
      className='bg-body'
      data-mt-drawer='true'
      data-mt-drawer-name='help'
      data-mt-drawer-activate='true'
      data-mt-drawer-overlay='true'
      data-mt-drawer-width="{default:'350px', 'md': '525px'}"
      data-mt-drawer-direction='end'
      data-mt-drawer-toggle='#mt_help_toggle'
      data-mt-drawer-close='#mt_help_close'
    >
      {/* begin::Card */}
      <div className='card shadow-none rounded-0 w-100'>
        {/* begin::Header */}
        <div className='card-header' id='mt_help_header'>
          <h5 className='card-title fw-bold text-gray-600'>Learn & Get Inspired</h5>

          <div className='card-toolbar'>
            <button
              type='button'
              className='btn btn-sm btn-icon explore-btn-dismiss me-n5'
              id='mt_help_close'
            >
              <img src='/media/icons/duotune/arrows/arr061.svg' className='svg-icon-2' />
            </button>
          </div>
        </div>
        {/* end::Header */}

        {/* begin::Body */}
        <div className='card-body' id='mt_help_body'>
          {/* begin::Content */}
          <div
            id='mt_help_scroll'
            className='hover-scroll-overlay-y'
            data-mt-scroll='true'
            data-mt-scroll-height='auto'
            data-mt-scroll-wrappers='#mt_help_body'
            data-mt-scroll-dependencies='#mt_help_header'
            data-mt-scroll-offset='5px'
          >
            {/* begin::Support */}
            <div className='rounded border border-dashed border-gray-300 p-6 p-lg-8 mb-10'>
              {/* begin::Heading */}
              <h2 className='fw-bolder mb-5'>
                Support at{' '}
                <a href='https://devs.keenthemes.com' className=''>
                  devs.keenthemes.com
                </a>
              </h2>
              {/* end::Heading */}

              {/* begin::Description */}
              <div className='fs-5 fw-bold mb-5'>
                <span className='text-gray-500'>
                  Join our developers community to find answer to your question and help others.
                </span>
                <a className='explore-link d-none' href='https://keenthemes.com/licensing'>
                  FAQs
                </a>
              </div>
              {/* end::Description */}

              {/* begin::Link */}
              <a
                href='https://devs.keenthemes.com'
                className='btn btn-lg explore-btn-primary w-100'
              >
                Get Support
              </a>
              {/* end::Link */}
            </div>
            {/* end::Support */}

            {/* begin::Link */}
            <div className='d-flex align-items-center mb-7'>
              {/* begin::Icon */}
              <div className='d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-warning'>
                <img
                  src='/media/icons/duotune/abstract/abs027.svg'
                  className='svg-icon-warning svg-icon-2x svg-icon-lg-3x'
                />
              </div>
              {/* end::Icon */}
              {/* begin::Info */}
              <div className='d-flex flex-stack flex-grow-1 ms-4 ms-lg-6'>
                {/* begin::Wrapper */}
                <div className='d-flex flex-column me-2 me-lg-5'>
                  {/* begin::Title */}
                  <a
                    href='https://preview.keenthemes.com/metronic8/react/docs/docs/quick-start'
                    className='text-dark text-hover-primary fw-bolder fs-6 fs-lg-4 mb-1'
                  >
                    Documentation &amp; Videos
                  </a>
                  {/* end::Title */}
                  {/* begin::Description */}
                  <div className='text-muted fw-bold fs-7 fs-lg-6'>
                    From guides and video tutorials, to live demos and code examples to get started.
                  </div>
                  {/* end::Description */}
                </div>
                {/* end::Wrapper */}
                <img
                  src='/media/icons/duotune/arrows/arr064.svg'
                  className='svg-icon-gray-400 svg-icon-2'
                />
              </div>
              {/* end::Info */}
            </div>
            {/* end::Link */}
            {/* begin::Link */}
            <div className='d-flex align-items-center mb-7'>
              {/* begin::Icon */}
              <div className='d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-primary'>
                <img
                  src='/media/icons/duotune/ecommerce/ecm007.svg'
                  className='svg-icon-primary svg-icon-2x svg-icon-lg-3x'
                />
              </div>
              {/* end::Icon */}
              {/* begin::Info */}
              <div className='d-flex flex-stack flex-grow-1 ms-4 ms-lg-6'>
                {/* begin::Wrapper */}
                <div className='d-flex flex-column me-2 me-lg-5'>
                  {/* begin::Title */}
                  <a
                    href='https://preview.keenthemes.com/metronic8/react/docs/docs/utilities'
                    className='text-dark text-hover-primary fw-bolder fs-6 fs-lg-4 mb-1'
                  >
                    Plugins &amp; Components
                  </a>
                  {/* end::Title */}
                  {/* begin::Description */}
                  <div className='text-muted fw-bold fs-7 fs-lg-6'>
                    Check out our 300+ in-house components and customized 3rd-party plugins.
                  </div>
                  {/* end::Description */}
                </div>
                {/* end::Wrapper */}
                <img
                  src='/media/icons/duotune/arrows/arr064.svg'
                  className='svg-icon-gray-400 svg-icon-2'
                />
              </div>
              {/* end::Info */}
            </div>
            {/* end::Link */}
            {/* begin::Link */}
            <div className='d-flex align-items-center mb-7'>
              {/* begin::Icon */}
              <div className='d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-info'>
                <img
                  src='/media/icons/duotune/art/art006.svg'
                  className='svg-icon-info svg-icon-2x svg-icon-lg-3x'
                />
              </div>
              {/* end::Icon */}
              {/* begin::Info */}
              <div className='d-flex flex-stack flex-grow-1 ms-4 ms-lg-6'>
                {/* begin::Wrapper */}
                <div className='d-flex flex-column me-2 me-lg-5'>
                  {/* begin::Title */}
                  <Link
                    href='/builder'
                    className='text-dark text-hover-primary fw-bolder fs-6 fs-lg-4 mb-1'
                  >
                    Layout Builder
                  </Link>
                  {/* end::Title */}
                  {/* begin::Description */}
                  <div className='text-muted fw-bold fs-7 fs-lg-6'>
                    Dynamically modify and preview layout
                  </div>
                  {/* end::Description */}
                </div>
                {/* end::Wrapper */}
                <img
                  src='/media/icons/duotune/arrows/arr064.svg'
                  className='svg-icon-gray-400 svg-icon-2'
                />
              </div>
              {/* end::Info */}
            </div>
            {/* end::Link */}
            {/* begin::Link */}
            <div className='d-flex align-items-center mb-7'>
              {/* begin::Icon */}
              <div className='d-flex flex-center w-50px h-50px w-lg-75px h-lg-75px flex-shrink-0 rounded bg-light-danger'>
                <img
                  src='/media/icons/duotune/electronics/elc009.svg'
                  className='svg-icon-danger svg-icon-2x svg-icon-lg-3x'
                />
              </div>
              {/* end::Icon */}
              {/* begin::Info */}
              <div className='d-flex flex-stack flex-grow-1 ms-4 ms-lg-6'>
                {/* begin::Wrapper */}
                <div className='d-flex flex-column me-2 me-lg-5'>
                  {/* begin::Title */}
                  <a
                    href='https://preview.keenthemes.com/metronic8/react/docs/docs/changelog'
                    className='text-dark text-hover-primary fw-bolder fs-6 fs-lg-4 mb-1'
                  >
                    {/* eslint-disable-next-line react/no-unescaped-entities */}
                    What's New
                  </a>
                  {/* end::Title */}
                  {/* begin::Description */}
                  <div className='text-muted fw-bold fs-7 fs-lg-6'>
                    Latest features and improvements added with our users feedback in mind.
                  </div>
                  {/* end::Description */}
                </div>
                {/* end::Wrapper */}
                <img
                  src='/media/icons/duotune/arrows/arr064.svg'
                  className='svg-icon-gray-400 svg-icon-2'
                />
              </div>
              {/* end::Info */}
            </div>
            {/* end::Link */}
          </div>
          {/* end::Content */}
        </div>
        {/* end::Body */}
      </div>
      {/* end::Card */}
    </div>
  )
}

export {HelpDrawer}
