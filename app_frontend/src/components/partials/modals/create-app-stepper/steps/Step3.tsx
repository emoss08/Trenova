/* eslint-disable jsx-a11y/anchor-is-valid */
import {StepProps} from '../IAppModels'

const Step3 = ({data, updateData, hasError}: StepProps) => {
  return (
    <>
      {/*begin::Step 3 */}
      <div className='pb-5' data-kt-stepper-element='content'>
        <div className='w-100'>
          {/*begin::Form Group */}

          <div className='fv-row mb-10'>
            <label className='required fs-5 fw-semibold mb-2'>Database Name</label>

            <input
              type='text'
              className='form-control form-control-lg form-control-solid'
              name='dbname'
              value={data.appDatabase.databaseName}
              onChange={(e) =>
                updateData({
                  appDatabase: {
                    databaseName: e.target.value,
                    databaseSolution: data.appDatabase.databaseSolution,
                  },
                })
              }
            />
            {!data.appDatabase.databaseName && hasError && (
              <div className='fv-plugins-message-container'>
                <div data-field='appname' data-validator='notEmpty' className='fv-help-block'>
                  Database name is required
                </div>
              </div>
            )}
          </div>
          {/*end::Form Group */}

          {/*begin::Form Group */}
          <div className='fv-row'>
            <label className='d-flex align-items-center fs-5 fw-semibold mb-4'>
              <span className='required'>Select Database Engine</span>

              <i
                className='fas fa-exclamation-circle ms-2 fs-7'
                data-bs-toggle='tooltip'
                title='Select your app database engine'
              ></i>
            </label>

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between cursor-pointer mb-6'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-success'>
                    <i className='fas fa-database text-success fs-2x'></i>
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>MySQL</span>
                  <span className='fs-7 text-muted'>Basic MySQL database</span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='databaseSolution'
                  value='MySQL'
                  checked={data.appDatabase.databaseSolution === 'MySQL'}
                  onChange={() =>
                    updateData({
                      appDatabase: {
                        databaseName: data.appDatabase.databaseName,
                        databaseSolution: 'MySQL',
                      },
                    })
                  }
                />
              </span>
            </label>
            {/*end::Option */}

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between cursor-pointer mb-6'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-danger'>
                    <i className='fab fa-google text-danger fs-2x'></i>
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>Firebase</span>
                  <span className='fs-7 text-muted'>Google based app data management</span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='databaseSolution'
                  value='Firebase'
                  checked={data.appDatabase.databaseSolution === 'Firebase'}
                  onChange={() =>
                    updateData({
                      appDatabase: {
                        databaseName: data.appDatabase.databaseName,
                        databaseSolution: 'Firebase',
                      },
                    })
                  }
                />
              </span>
            </label>
            {/*end::Option */}

            {/*begin:Option */}
            <label className='d-flex align-items-center justify-content-between cursor-pointer mb-6'>
              <span className='d-flex align-items-center me-2'>
                <span className='symbol symbol-50px me-6'>
                  <span className='symbol-label bg-light-warning'>
                    <i className='fab fa-amazon text-warning fs-2x'></i>
                  </span>
                </span>

                <span className='d-flex flex-column'>
                  <span className='fw-bolder fs-6'>DynamoDB</span>
                  <span className='fs-7 text-muted'>Amazon Fast NoSQL Database</span>
                </span>
              </span>

              <span className='form-check form-check-custom form-check-solid'>
                <input
                  className='form-check-input'
                  type='radio'
                  name='databaseSolution'
                  value='DynamoDB'
                  checked={data.appDatabase.databaseSolution === 'DynamoDB'}
                  onChange={() =>
                    updateData({
                      appDatabase: {
                        databaseName: data.appDatabase.databaseName,
                        databaseSolution: 'DynamoDB',
                      },
                    })
                  }
                />
              </span>
            </label>
            {/*end::Option */}
          </div>
          {/*end::Form Group */}
        </div>
      </div>
      {/*end::Step 3 */}
    </>
  )
}

export {Step3}
