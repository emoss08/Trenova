import {FC} from 'react'
import {Link} from 'react-router-dom'
import {toAbsoluteUrl} from '../../../../_metronic/helpers'

const Error500: FC = () => {
  return (
    <div className='d-flex flex-column flex-root'>
      {/*begin::Authentication - Error 500 */}
      <div className='d-flex flex-column flex-column-fluid'>
        {/*begin::Content*/}
        <div className='d-flex flex-column flex-column-fluid text-center p-10 py-lg-15'>
          {/*begin::Logo*/}
          <Link to='/' className='mb-10 pt-lg-10'>
            <img
              alt='Logo'
              src={toAbsoluteUrl('/media/logos/default.svg')}
              className='h-40px mb-5'
            />
          </Link>
          {/*end::Logo*/}
          {/*begin::Wrapper*/}
          <div className='pt-lg-10 mb-10'>
            {/*begin::Logo*/}
            <h1 className='fw-bolder fs-2qx text-gray-800 mb-10'>System Error</h1>
            {/*end::Logo*/}
            {/*begin::Message*/}
            <div className='fw-bold fs-3 text-muted mb-15'>
              Something went wrong!
              <br />
              Please try again later.
            </div>
            {/*end::Message*/}
            {/*begin::Action*/}
            <div className='text-center'>
              <Link to='/' className='btn btn-lg btn-primary fw-bolder'>
                Go to homepage
              </Link>
            </div>
            {/*end::Action*/}
          </div>
          {/*end::Wrapper*/}
          {/*begin::Illustration*/}
          <div
            className='d-flex flex-row-auto bgi-no-repeat bgi-position-x-center bgi-size-contain bgi-position-y-bottom min-h-100px min-h-lg-350px'
            style={{
              backgroundImage: `url('${toAbsoluteUrl('/media/illustrations/sketchy-1/17.png')}')`,
            }}
          ></div>
          {/*end::Illustration*/}
        </div>
        {/*end::Content*/}
        {/*begin::Footer*/}
        <div className='d-flex flex-center flex-column-auto p-10'>
          {/*begin::Links*/}
          <div className='d-flex align-items-center fw-bold fs-6'>
            <a href='https://keenthemes.com' className='text-muted text-hover-primary px-2'>
              About
            </a>
            <a href='mailto:support@keenthemes.com' className='text-muted text-hover-primary px-2'>
              Contact
            </a>
            <a href='https://1.envato.market/EA4JP' className='text-muted text-hover-primary px-2'>
              Contact Us
            </a>
          </div>
          {/*end::Links*/}
        </div>
        {/*end::Footer*/}
      </div>
      {/*end::Authentication - Error 500*/}
    </div>
  )
}

export {Error500}
