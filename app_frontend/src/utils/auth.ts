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

import { AuthModel } from "@/models/user";

const AUTH_LOCAL_STORAGE_KEY: string = "mt-auth-v";




export function getAuth(): AuthModel | undefined {
  try {
    if (typeof window !== "undefined") {
      const lsValue = localStorage.getItem(AUTH_LOCAL_STORAGE_KEY);
      if (!lsValue) {
        return undefined;
      }
      return JSON.parse(lsValue) as AuthModel;
    }
  } catch (error) {
    console.error("Error parsing user auth data from local storage:", error);
  }
  return undefined;
}

export function setAuth(auth: AuthModel): void {
  try {
    if (typeof window !== "undefined") {
      const lsValue: string = JSON.stringify(auth);
      localStorage.setItem(AUTH_LOCAL_STORAGE_KEY, lsValue);
    }
  } catch (error) {
    console.error("Error saving user auth data to local storage:", error);
  }
}

export function clearAuth(): void {
  try {
    if (typeof window !== "undefined") {
      localStorage.removeItem(AUTH_LOCAL_STORAGE_KEY);
    }
  } catch (error) {
    console.error("Error removing user auth data from local storage:", error);
  }
}

export function setupAxios(axiosInstance: any) {
  axiosInstance.defaults.headers.Accept = 'application/json';

  axiosInstance.interceptors.request.use(
    (config: any) => {
      const auth = getAuth();
      if (auth?.token) {
        config.headers.Authorization = `Token ${auth.token}`;
      }
      return Promise.resolve(config);
    },
    (error: any) => {
      console.error('Error adding authorization header to Axios request:', error);
      return Promise.reject(error);
    }
  );
}
