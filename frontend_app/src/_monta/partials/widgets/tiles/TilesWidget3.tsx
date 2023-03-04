/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import {KTSVG, toAbsoluteUrl} from '../../../helpers'

type Props = {
  className?: string
  bgColor?: string
  title?: string
  title2?: string
}
const TilesWidget3 = ({
  className,
  bgColor = '#663259',
  title = 'Create SaaS',
  title2 = 'Based Reports',
}: Props) => {
  return (
    <div
      className={clsx('card h-100 bgi-no-repeat bgi-size-cover', className)}
      style={{backgroundImage: `url("${toAbsoluteUrl('/media/misc/bg-2.jpg')}")`}}
    >
      {/* begin::Body */}
      <div className='card-body d-flex flex-column justify-content-between'>
        {/* begin::Title */}
        <div className='text-white fw-bold fs-2'>
          <h2 className='fw-bold text-white mb-2'>Create Reports</h2>
          With App
        </div>
        {/* end::Title */}

        {/* begin::Link */}
        <a href='#' className='text-warning fw-semibold'>
          Create Report
          <KTSVG
            className='svg-icon-2 svg-icon-warning'
            path='/media/icons/duotune/arrows/arr064.svg'
          />
        </a>
        {/* end::Link */}
      </div>
      {/* end::Body */}
    </div>
  )
}

export {TilesWidget3}
