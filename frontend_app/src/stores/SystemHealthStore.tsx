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

import {create} from 'zustand'
import createRequestFactory from '../factory/RequestFactory'

interface SystemHealthData {
  [key: string]: any
}

interface SystemHealthStore {
  loading: boolean
  serviceData: SystemHealthData | null
  fetchData: () => void
}

export const useSystemHealthStore = create<SystemHealthStore>((set) => ({
  loading: true,
  serviceData: null,
  fetchData: createRequestFactory(
    'http://127.0.0.1:8000/api/system_health/',
    (data) => set({serviceData: data}),
    (loading) => set({loading})
  ),
}))
