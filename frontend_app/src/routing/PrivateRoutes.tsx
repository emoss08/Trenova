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

import React, {lazy, Suspense} from 'react'
import {Navigate, Route, Routes} from 'react-router-dom'
// import SystemHealthPage from '../pages/SystemHealthPage'

const PrivateRoutes = () => {
  const SystemHealthPage = lazy(() => import('../pages/SystemHealthPage'))

  return (
    <Routes>
      <Route
        path='/system_health'
        element={
          <Suspense fallback={<div>Loading...</div>}>
            <SystemHealthPage />
          </Suspense>
        }
      ></Route>
      {/* Page Not Found */}
      <Route path='*' element={<Navigate to='/error/404' />} />
    </Routes>
  )
}
export {PrivateRoutes}
