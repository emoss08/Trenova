/* eslint-disable jsx-a11y/anchor-is-valid */
import {toAbsoluteUrl} from '../../../../helpers'

const Step5 = () => {
  return (
    <>
      <div data-kt-stepper-element='content'>
        <div className='w-100 text-center'>
          {/* begin::Heading */}
          <h1 className='fw-bold text-dark mb-3'>Release!</h1>
          {/* end::Heading */}

          {/* begin::Description */}
          <div className='text-muted fw-semibold fs-3'>
            Submit your app to kickstart your project.
          </div>
          {/* end::Description */}

          {/* begin::Illustration */}
          <div className='text-center px-4 py-15'>
            <img
              src={toAbsoluteUrl('/media/illustrations/sketchy-1/9.png')}
              alt=''
              className='mw-100 mh-300px'
            />
          </div>
          {/* end::Illustration */}
        </div>
      </div>
    </>
  )
}

export {Step5}
