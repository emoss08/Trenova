/* eslint-disable jsx-a11y/anchor-is-valid */
import {Link} from 'react-router-dom'
import {toAbsoluteUrl} from '../../../../helpers'

type Props = {
  className: string
}

const EngageWidget10 = ({className}: Props) => (
  <div className={`card card-flush ${className}`}>
    <div
      className='card-body d-flex flex-column justify-content-between mt-9 bgi-no-repeat bgi-size-cover bgi-position-x-center pb-0'
      style={{
        backgroundPosition: '100% 50%',
        backgroundImage: `url('${toAbsoluteUrl('/media/stock/900x600/42.png')}')`,
      }}
    >
      <div className='mb-10'>
        <div className='fs-2hx fw-bold text-gray-800 text-center mb-13'>
          <span className='me-2'>
            Try our all new Enviroment with
            <br />
            <span className='position-relative d-inline-block text-danger'>
              <Link
                to='/crafted/pages/profile/overview'
                className='text-danger
              opacity-75-hover'
              >
                Pro Plan
              </Link>

              <span className='position-absolute opacity-15 bottom-0 start-0 border-4 border-danger border-bottom w-100'></span>
            </span>
          </span>
          for Free
        </div>

        <div className='text-center'>
          <a href='#'>Upgrade Now</a>
        </div>
      </div>
      <img
        className='mx-auto h-150px h-lg-200px  theme-light-show'
        src={toAbsoluteUrl('/media/illustrations/misc/upgrade.svg')}
        alt=''
      />
      <img
        className='mx-auto h-150px h-lg-200px  theme-dark-show'
        src={toAbsoluteUrl('/media/illustrations/misc/upgrade-dark.svg')}
        alt=''
      />
    </div>
  </div>
)
export {EngageWidget10}
