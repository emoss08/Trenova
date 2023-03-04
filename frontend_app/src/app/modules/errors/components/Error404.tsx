import {FC} from 'react'
import {Link} from 'react-router-dom'
import {toAbsoluteUrl} from '../../../../_metronic/helpers'

const Error404: FC = () => {
  return (
    <div className='d-flex flex-column flex-root'>
      <div className='d-flex flex-column flex-center flex-column-fluid p-10'>
        {/* begin::Illustration */}
        <img
          src={toAbsoluteUrl('/media/illustrations/sketchy-1/18.png')}
          alt=''
          className='mw-100 mb-10 h-lg-450px'
        />
        {/* end::Illustration */}
        {/* begin::Message */}
        <h1 className='fw-bold mb-10' style={{color: '#A3A3C7'}}>
          Seems there is nothing here
        </h1>
        {/* end::Message */}
        {/* begin::Link */}
        <Link to='/' className='btn btn-primary'>
          Return Home
        </Link>
        {/* end::Link */}
      </div>
    </div>
  )
}

export {Error404}
