type Props = {
  className: string
  description: string
  color: string
  img: string
}

const CardsWidget20 = ({className, description, color, img}: Props) => (
  <div
    className={`card card-flush bgi-no-repeat bgi-size-contain bgi-position-x-end ${className}`}
    style={{
      backgroundColor: color,
      backgroundImage: `url('${img}')`,
    }}
  >
    <div className='card-header pt-5'>
      <div className='card-title d-flex flex-column'>
        <span className='fs-2hx fw-bold text-white me-2 lh-1 ls-n2'>69</span>

        <span className='text-white opacity-75 pt-1 fw-semibold fs-6'>{description}</span>
      </div>
    </div>
    <div className='card-body d-flex align-items-end pt-0'>
      <div className='d-flex align-items-center flex-column mt-3 w-100'>
        <div className='d-flex justify-content-between fw-bold fs-6 text-white opacity-75 w-100 mt-auto mb-2'>
          <span>43 Pending</span>
          <span>72%</span>
        </div>

        <div className='h-8px mx-3 w-100 bg-white bg-opacity-50 rounded'>
          <div
            className='bg-white rounded h-8px'
            role='progressbar'
            style={{width: '72%'}}
            aria-valuenow={50}
            aria-valuemin={0}
            aria-valuemax={100}
          ></div>
        </div>
      </div>
    </div>
  </div>
)
export {CardsWidget20}
