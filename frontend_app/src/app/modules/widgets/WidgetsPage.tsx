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

import {Navigate, Route, Routes, Outlet} from 'react-router-dom'
import {PageLink, PageTitle} from '../../../_monta/layout/core'
import {Charts} from './components/Charts'
import {Feeds} from './components/Feeds'
import {Lists} from './components/Lists'
import {Tables} from './components/Tables'
import {Mixed} from './components/Mixed'
import {Statistics} from './components/Statistics'

const widgetsBreadCrumbs: Array<PageLink> = [
  {
    title: 'Widgets',
    path: '/crafted/widgets/charts',
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

const WidgetsPage = () => {
  return (
    <Routes>
      <Route element={<Outlet />}>
        <Route
          path='charts'
          element={
            <>
              <PageTitle breadcrumbs={widgetsBreadCrumbs}>Charts</PageTitle>
              <Charts />
            </>
          }
        />
        <Route
          path='feeds'
          element={
            <>
              <PageTitle breadcrumbs={widgetsBreadCrumbs}>Feeds</PageTitle>
              <Feeds />
            </>
          }
        />
        <Route
          path='lists'
          element={
            <>
              <PageTitle breadcrumbs={widgetsBreadCrumbs}>Lists</PageTitle>
              <Lists />
            </>
          }
        />
        <Route
          path='mixed'
          element={
            <>
              <PageTitle breadcrumbs={widgetsBreadCrumbs}>Mixed</PageTitle>
              <Mixed />
            </>
          }
        />
        <Route
          path='tables'
          element={
            <>
              <PageTitle breadcrumbs={widgetsBreadCrumbs}>Tables</PageTitle>
              <Tables />
            </>
          }
        />
        <Route
          path='statistics'
          element={
            <>
              <PageTitle breadcrumbs={widgetsBreadCrumbs}>Statiscics</PageTitle>
              <Statistics />
            </>
          }
        />
        <Route index element={<Navigate to='/crafted/widgets/lists' />} />
      </Route>
    </Routes>
  )
}

export default WidgetsPage
