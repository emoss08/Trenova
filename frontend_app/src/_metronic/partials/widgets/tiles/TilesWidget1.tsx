/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import {toAbsoluteUrl} from '../../../helpers'

type Props = {
  className?: string
  bgImage?: string
  title?: string
}
const TilesWidget1 = ({
  className,
  bgImage = toAbsoluteUrl('/media/stock/600x400/img-75.jpg'),
  title = 'Properties',
}: Props) => {
  return (
    <div
      className={clsx('card h-150px bgi-no-repeat bgi-size-cover', className)}
      style={{
        backgroundImage: `url("${bgImage}")`,
      }}
    >
      <div className='card-body p-6'>
        <a
          href='#'
          className='text-black text-hover-primary fw-bold fs-2'
          data-bs-toggle='modal'
          data-bs-target='#kt_modal_create_app'
        >
          {title}
        </a>
      </div>
    </div>
  )
}

export {TilesWidget1}
