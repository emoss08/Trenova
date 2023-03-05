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

import axios from 'axios'
import {AuthModel, JobTitleModel, UserModel} from './_models'
import {IProfileDetails} from '../../accounts/components/settings/cards/ProfileDetails'

const API_URL = process.env.REACT_APP_API_URL

export const GET_USER_BY_ACCESSTOKEN_URL = `${API_URL}/verify_token/`
export const LOGIN_URL = `${API_URL}/login/`
export const REGISTER_URL = `${API_URL}/register/`
export const REQUEST_PASSWORD_URL = `${API_URL}/forgot_password/`

export const JOB_TITLE_URL = `${API_URL}/job_titles/`

// Server should return AuthModel
export function login(username: string, password: string) {
  return axios.post<AuthModel>(LOGIN_URL, {
    username,
    password,
  })
}

// Server should return AuthModel
export function register(
  email: string,
  firstname: string,
  lastname: string,
  password: string,
  password_confirmation: string
) {
  return axios.post(REGISTER_URL, {
    email,
    first_name: firstname,
    last_name: lastname,
    password,
    password_confirmation,
  })
}

// Server should return object => { result: boolean } (Is Email in DB)
export function requestPassword(email: string) {
  return axios.post<{result: boolean}>(REQUEST_PASSWORD_URL, {
    email,
  })
}

/**
 * Send a post request to GET_USER_BY_ACCESSTOKEN_URL to get
 * @param token
 */
export function getUserByToken(token: string) {
  return axios.post<UserModel>(GET_USER_BY_ACCESSTOKEN_URL, {
    token: token,
  })
}

export function getJobTitle(id?: string) {
  return axios.get<JobTitleModel>(`${JOB_TITLE_URL}${id}/`)
}

export function getUser(id?: string) {
  return axios.get<UserModel>(`${API_URL}/users/${id}/`)
}

export function getFullUser(id?: string) {
  return axios.get<IProfileDetails>(`${API_URL}/users/${id}/`)
}
