/* eslint-disable jsx-a11y/anchor-is-valid */
import {KTSVG} from '../../../../../_metronic/helpers'
import {StepProps} from '../IAppModels'

const Step1 = ({data, updateData, hasError}: StepProps) => {
  return (
    <div className='current' data-kt-stepper-element='content'>
      <div className='w-100'>
        {/*begin::Form Group */}
        <div className='fv-row mb-10'>
          <label className='d-flex align-items-center fs-5 fw-semibold mb-2'>
            <span className='required'>App Name</span>
            <i
              className='fas fa-exclamation-circle ms-2 fs-7'
              data-bs-toggle='tooltip'
              title='Specify your unique app name'
            ></i>
          </label>
          <input
            type='text'
            className='form-control form-control-lg form-control-solid'
            name='appname'
            placeholder=''
            value={data.appBasic.appName}
            onChange={(e) =>
              updateData({
                appBasic: {
                  appName: e.target.value,
                  appType: data.appBasic.appType,
                },
              })
            }
          />
          {!data.appBasic.appName && hasError && (
            <div className='fv-plugins-message-container'>
              <div data-field='appname' data-validator='notEmpty' className='fv-help-block'>
                App name is required
              </div>
            </div>
          )}
        </div>
        {/*end::Form Group */}

        {/*begin::Form Group */}
        <div className='fv-row'>
          {/* begin::Label */}
          <label className='d-flex align-items-center fs-5 fw-semibold mb-4'>
            <span className='required'>Category</span>

            <i
              className='fas fa-exclamation-circle ms-2 fs-7'
              data-bs-toggle='tooltip'
              title='Select your app category'
            ></i>
          </label>
          {/* end::Label */}
          <div>
            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between mb-6 cursor-pointer'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-primary'>
                    <KTSVG
                      path='/media/icons/duotune/maps/map004.svg'
                      className='svg-icon-1 svg-icon-primary'
                    />
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>Quick Online Courses</span>
                  <span className='fs-7 text-muted'>
                    Creating a clear text structure is just one SEO
                  </span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='appType'
                  value='Quick Online Courses'
                  checked={data.appBasic.appType === 'Quick Online Courses'}
                  onChange={() =>
                    updateData({
                      appBasic: {
                        appName: data.appBasic.appName,
                        appType: 'Quick Online Courses',
                      },
                    })
                  }
                />
              </span>
            </label>
            {/*end::Option */}

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between mb-6 cursor-pointer'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-danger'>
                    <KTSVG
                      path='/media/icons/duotune/general/gen024.svg'
                      className='svg-icon-1 svg-icon-danger'
                    />
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>Face to Face Discussions</span>
                  <span className='fs-7 text-muted'>
                    Creating a clear text structure is just one aspect
                  </span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='appType'
                  value='Face to Face Discussions'
                  checked={data.appBasic.appType === 'Face to Face Discussions'}
                  onChange={() =>
                    updateData({
                      appBasic: {
                        appName: data.appBasic.appName,
                        appType: 'Face to Face Discussions',
                      },
                    })
                  }
                />
              </span>
            </label>
            {/*end::Option */}

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between mb-6 cursor-pointer'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-success'>
                    <KTSVG
                      path='/media/icons/duotune/general/gen013.svg'
                      className='svg-icon-1 svg-icon-success'
                    />
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>Full Intro Training</span>
                  <span className='fs-7 text-muted'>
                    Creating a clear text structure copywriting
                  </span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='appType'
                  value='Full Intro Training'
                  checked={data.appBasic.appType === 'Full Intro Training'}
                  onChange={() =>
                    updateData({
                      appBasic: {
                        appName: data.appBasic.appName,
                        appType: 'Full Intro Training',
                      },
                    })
                  }
                />
              </span>
            </label>
            {/*end::Option */}
          </div>
        </div>
        {/*end::Form Group */}
      </div>
    </div>
  )
}

export {Step1}
