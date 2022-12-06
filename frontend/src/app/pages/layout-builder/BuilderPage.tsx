/* eslint-disable jsx-a11y/anchor-is-valid */
import clsx from 'clsx'
import React, {useState} from 'react'
import {MTSVG, toAbsoluteUrl} from '../../../_monta/helpers'
import {getLayoutFromLocalStorage, ILayout, LayoutSetup} from '../../../_monta/layout/core'

const BuilderPage: React.FC = () => {
  const [tab, setTab] = useState('Sidebar')
  const [config, setConfig] = useState<ILayout>(getLayoutFromLocalStorage())
  const [configLoading, setConfigLoading] = useState<boolean>(false)
  const [resetLoading, setResetLoading] = useState<boolean>(false)

  const updateConfig = () => {
    setConfigLoading(true)
    try {
      LayoutSetup.setConfig(config)
      window.location.reload()
    } catch (error) {
      setConfig(getLayoutFromLocalStorage())
      setConfigLoading(false)
    }
  }

  const reset = () => {
    setResetLoading(true)
    setTimeout(() => {
      setConfig(getLayoutFromLocalStorage())
      setResetLoading(false)
    }, 1000)
  }

  return (
    <>
      <div className='card mb-10'>
        <div className='card-body d-flex align-items-center py-8'>
          <div className='d-flex h-80px w-80px flex-shrink-0 flex-center position-relative'>
            <MTSVG
              path='/media/icons/duotune/abstract/abs051.svg'
              className='svg-icon-primary position-absolute opacity-15'
              svgClassName='h-80px w-80px'
            />
            <MTSVG
              path='/media/icons/duotune/coding/cod009.svg'
              className='svg-icon-3x svg-icon-primary position-absolute'
            />
          </div>

          <div className='ms-6'>
            <p className='list-unstyled text-gray-600 fw-bold fs-6 p-0 m-0'>
              The layout builder is to assist your set and configure your preferred project layout
              specifications and preview it in real-time.
            </p>
            <p className='list-unstyled text-gray-600 fw-bold fs-6 p-0 m-0'>
              Also, you can configurate the Layout in the code (
              <code>src/_metronic/layout/core/_LayoutConfig.ts</code> file). Don't forget clear your
              local storage when you are changing _LayoutConfig.
            </p>
          </div>
        </div>
      </div>

      <div className='card card-custom'>
        <div className='card-header card-header-stretch overflow-auto'>
          <ul
            className='nav nav-stretch nav-line-tabs
            fw-bold
            border-transparent
            flex-nowrap
          '
            role='tablist'
          >
            <li className='nav-item'>
              <a
                className={clsx(`nav-link cursor-pointer`, {active: tab === 'Sidebar'})}
                onClick={() => setTab('Sidebar')}
                role='tab'
              >
                Sidebar
              </a>
            </li>
            <li className='nav-item'>
              <a
                className={clsx(`nav-link cursor-pointer`, {active: tab === 'Header'})}
                onClick={() => setTab('Header')}
                role='tab'
              >
                Header
              </a>
            </li>
            <li className='nav-item'>
              <a
                className={clsx(`nav-link cursor-pointer`, {active: tab === 'Toolbar'})}
                onClick={() => setTab('Toolbar')}
                role='tab'
              >
                Toolbar
              </a>
            </li>
          </ul>
        </div>

        <form className='form'>
          <div className='card-body'>
            <div className='tab-content pt-3'>
              <div className={clsx('tab-pane', {active: tab === 'Sidebar'})}>
                <div className='form-group d-flex flex-stack'>
                  <div className='d-flex flex-column'>
                    <h4 className='fw-bold text-dark'>Fixed</h4>
                    <div className='fs-7 fw-semibold text-muted'>Fixed sidebar mode</div>
                  </div>
                  <div className='d-flex justify-content-end'>
                    <div className='form-check form-check-custom form-check-solid form-check-success form-switch'>
                      <div
                        className='form-check form-check-custom form-check-solid form-switch
                    mb-2'
                      >
                        <input
                          className='form-check-input'
                          type='checkbox'
                          name='model.app.sidebar.default.fixed.desktop'
                          checked={config.app?.sidebar?.default?.fixed?.desktop}
                          onChange={() => {
                            const con = {...config}
                            if (
                              con.app &&
                              con.app.sidebar &&
                              con.app.sidebar.default &&
                              con.app.sidebar.default.fixed
                            ) {
                              con.app.sidebar.default.fixed.desktop =
                                !con.app.sidebar.default.fixed.desktop
                              setConfig({...con})
                            }
                          }}
                        />
                      </div>
                    </div>
                  </div>
                </div>
                <div className='separator separator-dashed my-6'></div>
                <div className='form-group d-flex flex-stack'>
                  <div className='d-flex flex-column'>
                    <h4 className='fw-bold text-dark'>Minimize</h4>
                    <div className='fs-7 fw-semibold text-muted'>Sidebar minimize mode</div>
                  </div>
                  <div className='d-flex justify-content-end'>
                    <div className='form-check form-check-custom form-check-solid form-check-success form-switch'>
                      <div
                        className='
                form-check form-check-custom form-check-solid form-check-success form-switch
                  '
                      >
                        <input
                          className='form-check-input'
                          type='checkbox'
                          name='model.app.sidebar.default.minimize.desktop.enabled'
                          id='kt_builder_sidebar_minimize_desktop_enabled'
                          checked={config.app?.sidebar?.default?.minimize?.desktop?.enabled}
                          onChange={() => {
                            const con = {...config}
                            if (
                              con.app &&
                              con.app.sidebar &&
                              con.app.sidebar.default &&
                              con.app.sidebar.default.minimize &&
                              con.app.sidebar.default.minimize.desktop
                            ) {
                              con.app.sidebar.default.minimize.desktop.enabled =
                                !con.app.sidebar.default.minimize.desktop.enabled
                              setConfig({...con})
                            }
                          }}
                        />
                        <label
                          className='form-check-label text-gray-700 fw-bold'
                          htmlFor='kt_builder_sidebar_minimize_desktop_enabled'
                          data-bs-toggle='tooltip'
                          data-kt-initialized='1'
                        >
                          Minimize Toggle
                        </label>
                      </div>
                      <div
                        className='
                form-check form-check-custom form-check-solid form-check-success form-switch ms-10
                  '
                      >
                        <input
                          className='form-check-input'
                          type='checkbox'
                          id='kt_builder_sidebar_minimize_desktop_hoverable'
                          name='model.app.sidebar.default.minimize.desktop.hoverable'
                          checked={config.app?.sidebar?.default?.minimize?.desktop?.hoverable}
                          onChange={() => {
                            const con = {...config}
                            if (
                              con.app &&
                              con.app.sidebar &&
                              con.app.sidebar.default &&
                              con.app.sidebar.default.minimize &&
                              con.app.sidebar.default.minimize.desktop
                            ) {
                              con.app.sidebar.default.minimize.desktop.hoverable =
                                !con.app.sidebar.default.minimize.desktop.hoverable
                              setConfig({...con})
                            }
                          }}
                        />
                        <label
                          className='form-check-label text-gray-700 fw-bold'
                          htmlFor='kt_builder_sidebar_minimize_desktop_hoverable'
                          data-bs-toggle='tooltip'
                          data-kt-initialized='1'
                        >
                          Hoverable
                        </label>
                      </div>
                      <div
                        className='
                form-check form-check-custom form-check-solid form-check-success form-switch ms-10
                  '
                      >
                        <input
                          className='form-check-input'
                          type='checkbox'
                          id='kt_builder_sidebar_minimize_desktop_default'
                          name='model.app.sidebar.default.minimize.desktop.default'
                          checked={config.app?.sidebar?.default?.minimize?.desktop?.default}
                          onChange={() => {
                            const con = {...config}
                            if (
                              con.app &&
                              con.app.sidebar &&
                              con.app.sidebar.default &&
                              con.app.sidebar.default.minimize &&
                              con.app.sidebar.default.minimize.desktop
                            ) {
                              con.app.sidebar.default.minimize.desktop.default =
                                !con.app.sidebar.default.minimize.desktop.default
                              setConfig({...con})
                            }
                          }}
                        />
                        <label
                          className='form-check-label text-gray-700 fw-bold'
                          htmlFor='kt_builder_sidebar_minimize_desktop_default'
                          data-bs-toggle='tooltip'
                          data-kt-initialized='1'
                        >
                          Default Minimized
                        </label>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div className={clsx('tab-pane', {active: tab === 'Header'})}>
                <div className='form-group d-flex flex-stack'>
                  <div className='d-flex flex-column'>
                    <h4 className='fw-bold text-dark'>Fixed</h4>
                    <div className='fs-7 fw-semibold text-muted'>Fixed header mode</div>
                  </div>
                  <div className='d-flex justify-content-end'>
                    <div className='form-check form-check-custom form-check-solid form-check-success form-switch'>
                      <div
                        className='
                    form-check form-check-custom form-check-solid form-switch
                    mb-2
                  '
                      >
                        <input
                          className='form-check-input'
                          type='checkbox'
                          name='model.app.header.default.fixed.desktop'
                          checked={config.app?.header?.default?.fixed?.desktop}
                          onChange={() => {
                            const con = {...config}
                            if (
                              con.app &&
                              con.app.header &&
                              con.app.header.default &&
                              con.app.header.default.fixed
                            ) {
                              con.app.header.default.fixed.desktop =
                                !con.app.header.default.fixed.desktop
                              setConfig({...con})
                            }
                          }}
                          // [(ngModel)]="model.app.header.default.fixed.desktop"
                        />
                      </div>
                    </div>
                  </div>
                </div>
                <div className='separator separator-dashed my-6'></div>
                <div className='form-group d-flex flex-stack'>
                  <div className='d-flex flex-column'>
                    <h4 className='fw-bold text-dark'>Content</h4>
                    <div className='fs-7 fw-semibold text-muted'>Header content type</div>
                  </div>
                  <div className='d-flex justify-content-end'>
                    <div className='form-check form-check-custom form-check-success form-check-solid form-check-sm ms-10'>
                      <input
                        className='form-check-input'
                        type='radio'
                        checked={config.app?.header?.default?.content === 'menu'}
                        onChange={() => {
                          const con = {...config}
                          if (con.app && con.app.header && con.app.header.default) {
                            con.app.header.default.content = 'menu'
                            setConfig({...con})
                          }
                        }}
                        // [(ngModel)]="model.app.header.default.content}
                        value='menu'
                        id='kt_builder_header_content_menu'
                        name='model.app.header.default.content'
                      />
                      <label
                        className='form-check-label text-gray-700 fw-bold text-nowrap'
                        htmlFor='kt_builder_header_content_menu'
                      >
                        Menu
                      </label>
                    </div>
                    <div className='form-check form-check-custom form-check-success form-check-solid form-check-sm ms-10'>
                      <input
                        className='form-check-input'
                        type='radio'
                        value='page-title'
                        id='kt_builder_header_content_page-title'
                        checked={config.app?.header?.default?.content === 'page-title'}
                        onChange={() => {
                          const con = {...config}
                          if (con.app && con.app.header && con.app.header.default) {
                            con.app.header.default.content = 'page-title'
                            setConfig({...con})
                          }
                        }}
                      />
                      <label
                        className='form-check-label text-gray-700 fw-bold text-nowrap'
                        htmlFor='kt_builder_header_content_page-title'
                      >
                        Page Title
                      </label>
                    </div>
                  </div>
                </div>
              </div>

              <div className={clsx('tab-pane', {active: tab === 'Toolbar'})}>
                <div className='form-group d-flex flex-stack'>
                  <div className='d-flex flex-column'>
                    <h4 className='fw-bold text-dark'>Fixed</h4>
                    <div className='fs-7 fw-semibold text-muted'>Fixed toolbar mode</div>
                  </div>
                  <div className='d-flex justify-content-end'>
                    <div className='d-flex justify-content-end'>
                      <div className='form-check form-check-custom form-check-solid form-check-success form-switch me-10'>
                        <input
                          className='form-check-input w-45px h-30px'
                          type='checkbox'
                          id='kt_builder_toolbar_fixed_desktop'
                          name='model.app.toolbar.fixed.desktop'
                          checked={config.app?.toolbar?.fixed?.desktop}
                          onChange={() => {
                            const con = {...config}
                            if (con.app && con.app.toolbar && con.app.toolbar.fixed) {
                              con.app.toolbar.fixed.desktop = !con.app.toolbar.fixed.desktop
                              setConfig({...con})
                            }
                          }}
                        />
                        <label
                          className='form-check-label text-gray-700 fw-bold'
                          htmlFor='kt_builder_toolbar_fixed_desktop'
                        >
                          Desktop Mode
                        </label>
                      </div>
                      <div className='form-check form-check-custom form-check-solid form-check-success form-switch'>
                        <input
                          className='form-check-input w-45px h-30px'
                          type='checkbox'
                          name='model.app.toolbar.fixed.mobile'
                          checked={config.app?.toolbar?.fixed?.mobile}
                          onChange={() => {
                            const con = {...config}
                            if (con.app && con.app.toolbar && con.app.toolbar.fixed) {
                              con.app.toolbar.fixed.mobile = !con.app.toolbar.fixed.mobile
                              setConfig({...con})
                            }
                          }}
                          id='kt_builder_toolbar_fixed_mobile'
                        />
                        <label
                          className='form-check-label text-gray-700 fw-bold'
                          htmlFor='kt_builder_toolbar_fixed_mobile'
                        >
                          Mobile Mode
                        </label>
                      </div>
                    </div>
                  </div>
                </div>
                <div className='separator separator-dashed my-6'></div>
                <div className='mb-6'>
                  <h4 className='fw-bold text-dark'>Layout</h4>
                  <div className='fw-semibold text-muted fs-7 d-block lh-1'>
                    Select a toolbar layout
                  </div>
                </div>

                <div
                  data-kt-buttons='true'
                  data-kt-buttons-target='.form-check-image:not(.disabled),.form-check-input:not([disabled])'
                  data-kt-initialized='1'
                >
                  <label
                    className={clsx('form-check-image form-check-success mb-10', {
                      active: config.app?.toolbar?.layout === 'classic',
                    })}
                  >
                    <div className='form-check-wrapper'>
                      <img
                        src={toAbsoluteUrl('/media/misc/layout/toolbar-classic.png')}
                        className='mw-100'
                        alt=''
                      />
                    </div>
                    <div className='form-check form-check-custom form-check-success form-check-sm form-check-solid'>
                      <input
                        className='form-check-input'
                        type='radio'
                        // checked="checked"
                        value='classic'
                        name='model.app.toolbar.layout'
                        checked={config.app?.toolbar?.layout === 'classic'}
                        onChange={() => {
                          const con = {...config}
                          if (con.app && con.app.toolbar) {
                            con.app.toolbar.layout = 'classic'
                            setConfig({...con})
                          }
                        }}
                        // [(ngModel)]="model.app.toolbar.layout"
                      />
                      <div className='form-check-label text-gray-800'>Classic</div>
                    </div>
                  </label>

                  <label
                    className={clsx('form-check-image form-check-success mb-10', {
                      active: config.app?.toolbar?.layout === 'saas',
                    })}
                  >
                    <div className='form-check-wrapper'>
                      <img
                        src={toAbsoluteUrl('/media/misc/layout/toolbar-saas.png')}
                        className='mw-100'
                        alt=''
                      />
                    </div>
                    <div className='form-check form-check-custom form-check-success form-check-sm form-check-solid'>
                      <input
                        className='form-check-input'
                        type='radio'
                        value='saas'
                        name='model.app.toolbar.layout'
                        checked={config.app?.toolbar?.layout === 'saas'}
                        onChange={() => {
                          const con = {...config}
                          if (con.app && con.app.toolbar) {
                            con.app.toolbar.layout = 'saas'
                            setConfig({...con})
                          }
                        }}
                        // [(ngModel)]="model.app.toolbar.layout"
                      />
                      <div className='form-check-label text-gray-800'>SaaS</div>
                    </div>
                  </label>

                  <label
                    className={clsx('form-check-image form-check-success mb-10', {
                      active: config.app?.toolbar?.layout === 'accounting',
                    })}
                  >
                    <div className='form-check-wrapper'>
                      <img
                        src={toAbsoluteUrl('/media/misc/layout/toolbar-accounting.png')}
                        className='mw-100'
                        alt=''
                      />
                    </div>
                    <div className='form-check form-check-custom form-check-success form-check-sm form-check-solid'>
                      <input
                        className='form-check-input'
                        type='radio'
                        value='accounting'
                        name='model.app.toolbar.layout'
                        checked={config.app?.toolbar?.layout === 'accounting'}
                        onChange={() => {
                          const con = {...config}
                          if (con.app && con.app.toolbar) {
                            con.app.toolbar.layout = 'accounting'
                            setConfig({...con})
                          }
                        }}
                        // [(ngModel)]="model.app.toolbar.layout"
                      />
                      <div className='form-check-label text-gray-800'>Accounting</div>
                    </div>
                  </label>

                  <label
                    className={clsx('form-check-image form-check-success mb-10', {
                      active: config.app?.toolbar?.layout === 'extended',
                    })} // [ngClass]="{'active': model.app.toolbar.layout === 'extended'}"
                  >
                    <div className='form-check-wrapper'>
                      <img
                        src={toAbsoluteUrl('/media/misc/layout/toolbar-extended.png')}
                        className='mw-100'
                        alt=''
                      />
                    </div>
                    <div className='form-check form-check-custom form-check-success form-check-sm form-check-solid'>
                      <input
                        className='form-check-input'
                        type='radio'
                        value='extended'
                        name='model.app.toolbar.layout'
                        checked={config.app?.toolbar?.layout === 'extended'}
                        onChange={() => {
                          const con = {...config}
                          if (con.app && con.app.toolbar) {
                            con.app.toolbar.layout = 'extended'
                            setConfig({...con})
                          }
                        }}
                        // [(ngModel)]="model.app.toolbar.layout"
                      />
                      <div className='form-check-label text-gray-800'>Extended</div>
                    </div>
                  </label>

                  <label
                    className={clsx('form-check-image form-check-success mb-10', {
                      active: config.app?.toolbar?.layout === 'reports',
                    })}
                  >
                    {/* begin::Image */}
                    <div className='form-check-wrapper'>
                      <img
                        src={toAbsoluteUrl('/media/misc/layout/toolbar-reports.png')}
                        className='mw-100'
                        alt=''
                      />
                    </div>
                    {/* end::Image */}
                    {/* begin::Check */}
                    <div className='form-check form-check-custom form-check-success form-check-sm form-check-solid'>
                      <input
                        className='form-check-input'
                        type='radio'
                        value='reports'
                        name='model.app.toolbar.layout'
                        checked={config.app?.toolbar?.layout === 'reports'}
                        onChange={() => {
                          const con = {...config}
                          if (con.app && con.app.toolbar) {
                            con.app.toolbar.layout = 'reports'
                            setConfig({...con})
                          }
                        }}
                        // [(ngModel)]="model.app.toolbar.layout"
                      />
                      {/* begin::Label */}
                      <div className='form-check-label text-gray-800'>Reports</div>
                      {/* end::Label */}
                    </div>
                    {/* end::Check */}
                  </label>
                  {/* end::Option */}
                </div>
              </div>
            </div>

            <div className='card-footer py-6'>
              <div className='row'>
                <div className='col-lg-3'></div>
                <div className='col-lg-9'>
                  <button type='button' onClick={updateConfig} className='btn btn-primary me-2'>
                    {!configLoading && <span className='indicator-label'>Preview</span>}
                    {configLoading && (
                      <span className='indicator-progress' style={{display: 'block'}}>
                        Please wait...{' '}
                        <span className='spinner-border spinner-border-sm align-middle ms-2'></span>
                      </span>
                    )}
                  </button>

                  <button
                    type='button'
                    id='kt_layout_builder_reset'
                    className='btn btn-active-light btn-color-muted'
                    onClick={reset}
                  >
                    {!resetLoading && <span className='indicator-label'>Reset</span>}
                    {resetLoading && (
                      <span className='indicator-progress' style={{display: 'block'}}>
                        Please wait...{' '}
                        <span className='spinner-border spinner-border-sm align-middle ms-2'></span>
                      </span>
                    )}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </form>
      </div>
    </>
  )
}

export {BuilderPage}
