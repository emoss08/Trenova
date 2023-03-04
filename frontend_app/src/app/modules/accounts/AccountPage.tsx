/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import React from 'react'
import {Navigate, Route, Routes, Outlet} from 'react-router-dom'
import {PageLink, PageTitle} from '../../../_monta/layout/core'
import {Overview} from './components/Overview'
import {Settings} from './components/settings/Settings'
import {AccountHeader} from './AccountHeader'

const accountBreadCrumbs: Array<PageLink> = [
  {
    title: 'Account',
    path: '/crafted/account/overview',
    isSeparator: false,
    isActive: false,
  },
  {
    title: '',
    path: '',
    isSeparator: true,
    isActive: false,
  },
]

const AccountPage: React.FC = () => {
  return (
    <Routes>
      <Route
        element={
          <>
            <AccountHeader />
            <Outlet />
          </>
        }
      >
        <Route
          path='overview'
          element={
            <>
              <PageTitle breadcrumbs={accountBreadCrumbs}>Overview</PageTitle>
              <Overview />
            </>
          }
        />
        <Route
          path='settings'
          element={
            <>
              <PageTitle breadcrumbs={accountBreadCrumbs}>Settings</PageTitle>
              <Settings />
            </>
          }
        />
        <Route index element={<Navigate to='/crafted/account/overview' />} />
      </Route>
    </Routes>
  )
}

export default AccountPage
