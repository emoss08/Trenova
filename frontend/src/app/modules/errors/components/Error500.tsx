import {FC} from 'react'
import {Link} from 'react-router-dom'
import {toAbsoluteUrl} from '../../../../_monta/helpers'

const Error500: FC = () => {
  return (
    <>
      {/* begin::Title */}
      <h1 className='fw-bolder fs-2qx text-gray-900 mb-4'>System Error</h1>
      {/* end::Title */}

      {/* begin::Text */}
      <div className='fw-semibold fs-6 text-gray-500 mb-7'>
        Something went wrong! Please try again later.
      </div>
      {/* end::Text */}

      {/* begin::Illustration */}
      <div className='mb-11'>
        <img
          src={toAbsoluteUrl('/media/auth/500-error.png')}
          className='mw-100 mh-300px theme-light-show'
          alt=''
        />
        <img
          src={toAbsoluteUrl('/media/auth/500-error-dark.png')}
          className='mw-100 mh-300px theme-dark-show'
          alt=''
        />
      </div>
      {/* end::Illustration */}

      {/* begin::Link */}
      <div className='mb-0'>
        <Link to='/dashboard' className='btn btn-sm btn-primary'>
          Return Home
        </Link>
      </div>
      {/* end::Link */}
    </>
  )
}

export {Error500}
