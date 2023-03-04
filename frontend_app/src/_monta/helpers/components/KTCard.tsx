import {FC} from 'react'
import clsx from 'clsx'
import {WithChildren} from '../react18MigrationHelpers'

// Wrapper on html card:
// https://preview.keenthemes.com/metronic8/demo1/documentation/base/cards.html

type Props = {
  className?: string
  shadow?: boolean
  flush?: boolean // https://preview.keenthemes.com/metronic8/demo1/documentation/base/cards.html#flush
  resetSidePaddings?: boolean // https://preview.keenthemes.com/metronic8/demo1/documentation/base/cards.html#reset-side-paddings
  border?: boolean // https://preview.keenthemes.com/metronic8/demo1/documentation/base/cards.html#bordered
  dashed?: boolean // https://preview.keenthemes.com/metronic8/demo1/documentation/base/cards.html#dashed
  stretch?: 'stretch' | 'stretch-75' | 'stretch-50' | 'stretch-33' | 'stretch-25' // https://preview.keenthemes.com/metronic8/demo1/documentation/base/cards.html#stretch
  rounded?: 'rounded' | 'rounded-top' | 'rounded-bottom'
  // https://preview.keenthemes.com/metronic8/demo1/documentation/base/cards.html#utilities
  utilityP?: number
  utilityPY?: number
  utilityPX?: number
}

const KTCard: FC<Props & WithChildren> = (props) => {
  const {
    className,
    shadow,
    flush,
    resetSidePaddings,
    border,
    dashed,
    stretch,
    rounded,
    utilityP,
    utilityPY,
    utilityPX,
    children,
  } = props
  return (
    <div
      className={clsx(
        'card',
        className && className,
        {
          'shadow-sm': shadow,
          'card-flush': flush,
          'card-px-0': resetSidePaddings,
          'card-bordered': border,
          'card-dashed': dashed,
        },
        stretch && `card-${stretch}`,
        utilityP && `p-${utilityP}`,
        utilityPX && `px-${utilityPX}`,
        utilityPY && `py-${utilityPY}`,
        rounded && `card-${rounded}`
      )}
    >
      {children}
    </div>
  )
}

export {KTCard}
