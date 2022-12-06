/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import {toAbsoluteUrl} from '../../../../helpers'

type Props = {
  className: string
  description: string
  icon: boolean
  stats: number
  labelColor: string
  textColor: string
}

const items: Array<{
  name: string
  initials?: string
  src?: string
  state?: string
}> = [
  {name: 'Alan Warden', initials: 'A', state: 'warning'},
  {name: 'Michael Eberon', src: toAbsoluteUrl('/media/avatars/300-11.jpg')},
  {name: 'Susan Redwood', initials: 'S', state: 'primary'},
  {name: 'Melody Macy', src: toAbsoluteUrl('/media/avatars/300-2.jpg')},
  {name: 'Perry Matthew', initials: 'P', state: 'danger'},
  {name: 'Barry Walter', src: toAbsoluteUrl('/media/avatars/300-12.jpg')},
]

const CardsWidget7 = ({className, description, icon, stats, labelColor, textColor}: Props) => (
  <div className={`card card-flush ${className}`}>
    <div className='card-header pt-5'>
      <div className='card-title d-flex flex-column'>
        <div className='card-title d-flex flex-column'>
          <span className='fs-2hx fw-bold text-dark me-2 lh-1 ls-n2'>{stats}</span>
          <span className='text-gray-400 pt-1 fw-semibold fs-6'>{description}</span>
        </div>
      </div>
    </div>
    <div className='card-body d-flex flex-column justify-content-end pe-0'>
      <span className='fs-6 fw-bolder text-gray-800 d-block mb-2'>Today’s Heroes</span>
      <div className='symbol-group symbol-hover flex-nowrap'>
        {items.map((item, index) => (
          <div
            className='symbol symbol-35px symbol-circle'
            data-bs-toggle='tooltip'
            title={item.name}
            key={`cw7-item-${index}`}
          >
            {item.src && <img alt='Pic' src={item.src} />}
            {item.state && item.initials && (
              <span
                className={clsx(
                  'symbol-label fw-bold',
                  'bg-' + item.state,
                  'text-inverse-' + item.state
                )}
              >
                {item.initials}
              </span>
            )}
          </div>
        ))}

        <a href='#' className='symbol symbol-35px symbol-circle'>
          <span
            className={clsx('symbol-label fs-8 fw-bold', 'bg-' + labelColor, 'text-' + textColor)}
          >
            +42
          </span>
        </a>
      </div>
    </div>
  </div>
)
export {CardsWidget7}
