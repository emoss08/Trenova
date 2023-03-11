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

import axios, { Axios, AxiosResponse } from "axios";
import { AuthModel, JobTitleModel, UserAuthModel, UserModel } from "@/models/user";
import { JOB_TITLE_URL, LOGIN_URL, USER_URL, VERIFY_TOKEN } from "@/utils/_links";

/**
 * Send a post request to LOGIN_URL to get
 * @param username
 * @param password
 * @returns {Promise<AxiosResponse<AuthModel>>}
 */
export function login(username: string, password: string): Promise<AxiosResponse<AuthModel>> {
  return axios.post<AuthModel>(LOGIN_URL, {
    username,
    password
  });
}

/**
 * Get a job title by id
 * @param id
 * @returns {Promise<AxiosResponse<JobTitleModel>>}
 */
export function getJobTitle(id?: string): Promise<AxiosResponse<JobTitleModel>> {
  return axios.get<JobTitleModel>(`${JOB_TITLE_URL}${id}/`);
}


/**
 * Send a post request to VERIFY_TOKEN to get
 * @param token
 * @returns {Promise<AxiosResponse<UserAuthModel>>}
 */
export function getUserByToken(token: string): Promise<AxiosResponse<UserAuthModel>> {
  return axios.post<UserAuthModel>(VERIFY_TOKEN, {
    token: token
  });
}

export function getUsersList(): Promise<AxiosResponse<UserModel[]>> {
  return axios.get<UserModel[]>(USER_URL)
}

export function getUser(id: string): Promise<AxiosResponse<UserModel>> {
  return axios.get<UserModel>(`${USER_URL}${id}/`)
}