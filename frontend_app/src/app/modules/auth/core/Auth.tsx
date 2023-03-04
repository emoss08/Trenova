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

import {
  FC,
  useState,
  useEffect,
  createContext,
  useContext,
  useRef,
  Dispatch,
  SetStateAction,
} from 'react'
import {LayoutSplashScreen} from '../../../../_monta/layout/core'
import {AuthModel, JobTitleModel, UserModel} from './_models'
import * as authHelper from './AuthHelpers'
import {getJobTitle, getUserByToken} from './_requests'
import {WithChildren} from '../../../../_monta/helpers'

type AuthContextProps = {
  auth: AuthModel | undefined
  saveAuth: (auth: AuthModel | undefined) => void
  currentUser: UserModel | undefined
  jobTitle: JobTitleModel | undefined
  setJobTitle: Dispatch<SetStateAction<JobTitleModel | undefined>>
  setCurrentUser: Dispatch<SetStateAction<UserModel | undefined>>
  loadingUser: boolean
  setLoadingUser: Dispatch<SetStateAction<boolean>>
  logout: () => void
}

const initAuthContextPropsState = {
  auth: authHelper.getAuth(),
  saveAuth: () => {},
  jobTitle: undefined,
  setJobTitle: () => {},
  currentUser: undefined,
  setCurrentUser: () => {},
  logout: () => {},
  loadingUser: false,
  setLoadingUser: () => {},
}

const AuthContext = createContext<AuthContextProps>(initAuthContextPropsState)

const useAuth = () => {
  return useContext(AuthContext)
}

const AuthProvider: FC<WithChildren> = ({children}) => {
  const [auth, setAuth] = useState<AuthModel | undefined>(authHelper.getAuth())
  const [currentUser, setCurrentUser] = useState<UserModel | undefined>()
  const [loadingUser, setLoadingUser] = useState(false)
  const [jobTitle, setJobTitle] = useState<JobTitleModel | undefined>()
  const saveAuth = (auth: AuthModel | undefined) => {
    setAuth(auth)
    if (auth) {
      authHelper.setAuth(auth)
    } else {
      authHelper.removeAuth()
    }
  }

  const logout = () => {
    saveAuth(undefined)
    setCurrentUser(undefined)
    setJobTitle(undefined)
  }

  const contextValue = {
    auth,
    saveAuth,
    currentUser,
    setCurrentUser,
    loadingUser,
    setLoadingUser,
    jobTitle,
    setJobTitle,
    logout,
  }

  return <AuthContext.Provider value={contextValue}>{children}</AuthContext.Provider>
}

const AuthInit: FC<WithChildren> = ({children}) => {
  const {auth, logout, setCurrentUser, setLoadingUser, setJobTitle} = useAuth()
  const didRequest = useRef(false)
  const [showSplashScreen, setShowSplashScreen] = useState(true)

  useEffect(() => {
    const requestJobTitle = async (jobTitleId: string) => {
      try {
        const {data} = await getJobTitle(jobTitleId)
        if (data) {
          setJobTitle(data)
        }
      } catch (error) {
        console.error(error)
      }
    }

    const requestUser = async (apiToken: string) => {
      try {
        setLoadingUser(true)
        if (!didRequest.current) {
          const {data} = await getUserByToken(apiToken)
          if (data) {
            setCurrentUser(data)
            if (data.job_title_id) {
              requestJobTitle(data.job_title_id)
            }
          }
        }
      } catch (error) {
        console.error(error)
        if (!didRequest.current) {
          logout()
        }
      } finally {
        setShowSplashScreen(false)
        setLoadingUser(false)
      }

      return () => (didRequest.current = true)
    }

    if (auth && auth.token) {
      requestUser(auth.token)
    } else {
      logout()
      setShowSplashScreen(false)
      setLoadingUser(false)
    }
    // eslint-disable-next-line
  }, [])

  return showSplashScreen ? <LayoutSplashScreen /> : <>{children}</>
}

export {AuthProvider, AuthInit, useAuth}
