/* eslint-disable jsx-a11y/anchor-is-valid */
import {FC} from 'react'
import {Link} from 'react-router-dom'
import {toAbsoluteUrl} from '../../../../helpers'
import {useLayout} from '../../../core'

const MegaMenu: FC = () => {
  const {setLayoutType, setToolbarType} = useLayout()
  return (
    <div className='row'>
      {/* begin:Col */}
      <div className='col-lg-6'>
        {/* begin:Row */}
        <div className='row'>
          {/* begin:Col */}
          <div className='col-lg-6 mb-3'>
            {/* begin:Heading */}
            <h4 className='fs-6 fs-lg-4 text-gray-800 fw-bold mt-3 mb-3 ms-4'>Layouts</h4>
            {/* end:Heading */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setLayoutType('light-sidebar')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Light Sidebar</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setLayoutType('dark-sidebar')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Dark Sidebar</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setLayoutType('light-header')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Light Header</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setLayoutType('dark-header')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Dark Header</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
          </div>
          {/* end:Col */}
          {/* begin:Col */}
          <div className='col-lg-6 mb-3'>
            {/* begin:Heading */}
            <h4 className='fs-6 fs-lg-4 text-gray-800 fw-bold mt-3 mb-3 ms-4'>Toolbars</h4>
            {/* end:Heading */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setToolbarType('classic')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Classic</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setToolbarType('saas')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>SaaS</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setToolbarType('accounting')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Accounting</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setToolbarType('extended')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Extended</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
            {/* begin:Menu item */}
            <div className='menu-item p-0 m-0'>
              {/* begin:Menu link */}
              <a onClick={() => setToolbarType('reports')} className='menu-link'>
                <span className='menu-bullet'>
                  <span className='bullet bullet-dot bg-gray-300i h-6px w-6px'></span>
                </span>
                <span className='menu-title'>Reports</span>
              </a>
              {/* end:Menu link */}
            </div>
            {/* end:Menu item */}
          </div>
          {/* end:Col */}
        </div>
        {/* end:Row */}
        <div className='separator separator-dashed mx-lg-5 mt-2 mb-6'></div>
        {/* begin:Layout Builder */}
        <div className='d-flex flex-stack flex-wrap flex-lg-nowrap gap-2 mb-5 mb-lg-0 mx-lg-5'>
          <div className='d-flex flex-column me-5'>
            <div className='fs-6 fw-bold text-gray-800'>Layout Builder</div>
            <div className='fs-7 fw-semibold text-muted'>Customize view</div>
          </div>
          <Link to='/builder' className='btn btn-sm btn-primary fw-bold'>
            Try Builder
          </Link>
        </div>
        {/* end:Layout Builder */}
      </div>
      {/* end:Col */}
      {/* begin:Col */}
      <div className='col-lg-6 mb-3 py-lg-3 pe-lg-8 d-flex align-items-center'>
        <img src={toAbsoluteUrl('/media/stock/900x600/45.jpg')} className='rounded mw-100' alt='' />
      </div>
      {/* end:Col */}
    </div>
  )
}

export {MegaMenu}
