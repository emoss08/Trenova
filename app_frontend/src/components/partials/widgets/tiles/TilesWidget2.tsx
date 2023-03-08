/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import {toAbsoluteUrl} from '../../../helpers'

type Props = {
  className?: string
  bgColor?: string
  title?: string
  title2?: string
}
const TilesWidget2 = ({
  className,
  bgColor = '#663259',
  title = 'Create SaaS',
  title2 = 'Based Reports',
}: Props) => {
  return (
    <div
      className={clsx('card h-175px bgi-no-repeat bgi-size-contain', className)}
      style={{
        backgroundColor: bgColor,
        backgroundPosition: 'right',
        backgroundImage: `url("${toAbsoluteUrl('/media/svg/misc/taieri.svg')}")`,
      }}
    >
      <div className='card-body d-flex flex-column justify-content-between'>
        <h2 className='text-white fw-bold mb-5'>
          {title} <br /> {title2}{' '}
        </h2>

        <div className='m-0'>
          <a
            href='#'
            className='btn btn-danger fw-semibold px-6 py-3'
            data-bs-toggle='modal'
            data-bs-target='#kt_modal_create_app'
          >
            Create Campaign
          </a>
        </div>
      </div>
    </div>
  )
}

export {TilesWidget2}
