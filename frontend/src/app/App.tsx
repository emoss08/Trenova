import {Suspense} from 'react'
import {Outlet} from 'react-router-dom'
import {I18nProvider} from '../_monta/i18n/i18nProvider'
import {LayoutProvider, LayoutSplashScreen} from '../_monta/layout/core'
import {MasterInit} from '../_monta/layout/MasterInit'
import {AuthInit} from './modules/auth'

const App = () => {
  return (
    <Suspense fallback={<LayoutSplashScreen />}>
      <I18nProvider>
        <LayoutProvider>
          <AuthInit>
            <Outlet />
            <MasterInit />
          </AuthInit>
        </LayoutProvider>
      </I18nProvider>
    </Suspense>
  )
}

export {App}
