import React, {FC} from 'react'

const PurchaseButton: FC = () => (
  <a
    href={process.env.REACT_APP_PURCHASE_URL}
    className='engage-purchase-link btn btn-flex h-35px bg-body btn-color-gray-700 btn-active-color-gray-900 px-5 shadow-sm rounded-top-0'
  >
    Buy Now
  </a>
)

export {PurchaseButton}
