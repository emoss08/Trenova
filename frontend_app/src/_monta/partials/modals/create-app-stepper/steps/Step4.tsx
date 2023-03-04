/* eslint-disable jsx-a11y/anchor-is-valid */
import {StepProps} from '../IAppModels'

const Step4 = ({data, updateData}: StepProps) => {
  return (
    <>
      {/*begin::Step 4 */}
      <div className='pb-5' data-kt-stepper-element='content'>
        <div className='w-100'>
          {/*begin::Form Group */}
          <div className='fv-row'>
            <label className='fs-6 fw-bolder text-dark mb-7d-flex align-items-center fs-5 fw-semibold mb-4'>
              <span className='required'>Select Storage</span>
            </label>

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between cursor-pointer mb-6'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-primary'>
                    <i className='fab fa-linux text-primary fs-2x'></i>
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>Basic Server</span>
                  <span className='fs-7 text-muted'>Linux based server storage</span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='appStorage'
                  value='Basic Server'
                  checked={data.appStorage === 'Basic Server'}
                  onChange={() => updateData({appStorage: 'Basic Server'})}
                />
              </span>
            </label>
            {/*end::Option */}

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between cursor-pointer mb-6'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-warning'>
                    <i className='fab fa-aws text-warning fs-2x'></i>
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>AWS</span>
                  <span className='fs-7 text-muted'>Amazon Web Services</span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='appStorage'
                  value='AWS'
                  checked={data.appStorage === 'AWS'}
                  onChange={() => updateData({appStorage: 'AWS'})}
                />
              </span>
            </label>
            {/*end::Option */}

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between cursor-pointer mb-6'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-success  '>
                    <i className='fab fa-google text-success fs-2x'></i>
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>Google</span>
                  <span className='fs-7 text-muted'>Google Cloud Storage</span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='appStorage'
                  value='Google'
                  checked={data.appStorage === 'Google'}
                  onChange={() => updateData({appStorage: 'Google'})}
                />
              </span>
            </label>
            {/*end::Option */}
          </div>
          {/*end::Form Group */}
        </div>
      </div>
      {/*end::Step 4 */}
    </>
  )
}

export {Step4}
