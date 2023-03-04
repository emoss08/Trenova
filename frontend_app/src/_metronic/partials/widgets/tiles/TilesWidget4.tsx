/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'

type Props = {
  className?: string
}
const TilesWidget4 = ({className}: Props) => {
  return (
    <div className={clsx('card h-150px', className)}>
      <div className='card-body d-flex align-items-center justify-content-between flex-wrap'>
        <div className='me-2'>
          <h2 className='fw-bold text-gray-800 mb-3'>Create CRM Reports</h2>

          <div className='text-muted fw-semibold fs-6'>
            Generate the latest CRM report for company projects
          </div>
        </div>
        <a
          href='#'
          className='btn btn-primary fw-semibold'
          data-bs-toggle='modal'
          data-bs-target='#kt_modal_create_campaign'
        >
          Start Now
        </a>
      </div>
    </div>
  )
}

export {TilesWidget4}
