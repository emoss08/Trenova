/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import {KTSVG} from '../../../helpers'

type Props = {
  className?: string
  svgIcon?: string
  titleClass?: string
  descriptionClass?: string
  iconClass?: string
  title?: string
  description?: string
}
const TilesWidget5 = (props: Props) => {
  const {className, svgIcon, titleClass, descriptionClass, iconClass, title, description} = props
  return (
    <a href='#' className={clsx('card', className)}>
      <div className='card-body d-flex flex-column justify-content-between'>
        <KTSVG path={svgIcon || ''} className={clsx(iconClass, 'svg-icon-2hx ms-n1 flex-grow-1')} />
        <div className='d-flex flex-column'>
          <div className={clsx(titleClass, 'fw-bold fs-1 mb-0 mt-5')}>{title}</div>
          <div className={clsx(descriptionClass, 'fw-semibold fs-6')}>{description}</div>
        </div>
      </div>
    </a>
  )
}

export {TilesWidget5}
