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

import React, {useCallback, useEffect, useMemo} from 'react'
import {useSystemHealthStore} from '../stores/SystemHealthStore'
import SystemHealth from '../components/SystemHealth'
import SystemHealthLoader from '../components/SystemHealthLoader'

interface CacheBackend {
  name: string
  status: string
}

function SystemHealthPage() {
  const {serviceData} = useSystemHealthStore()

  const memoizedData = useMemo(() => serviceData, [serviceData])

  return (
    <div>
      {Object.entries(memoizedData || {}).map(([key, value], index) => (
        <div key={index}>
          <h2>{key}</h2>
          <ul>
            {key === 'cache_backend'
              ? value?.map((service: CacheBackend, index: number) => (
                  <li key={index}>
                    <SystemHealth service={service.name} status={service.status} />
                  </li>
                ))
              : value?.status && (
                  <li>
                    <SystemHealth service={key} status={value.status} />
                  </li>
                )}
          </ul>
        </div>
      ))}
    </div>
  )
}

function SystemHealthPageWrapper() {
  const {loading, fetchData} = useSystemHealthStore()

  const memoizedFetchData = useCallback(fetchData, [fetchData])

  useEffect(() => {
    memoizedFetchData()

    const interval = setInterval(() => {
      memoizedFetchData()
    }, 60000)

    return () => clearInterval(interval)
  }, [memoizedFetchData])

  return loading ? <SystemHealthLoader /> : <SystemHealthPage />
}

export default SystemHealthPageWrapper
