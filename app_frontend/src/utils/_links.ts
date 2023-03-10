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

const API_URL = process.env.NEXT_PUBLIC_API_URL

export const VERIFY_TOKEN = `${API_URL}/verify_token/`
export const JOB_TITLE_URL = `${API_URL}/job_titles/`
export const LOGIN_URL = `${API_URL}/login/`

export const USER_URL = `${API_URL}/users/?limit=20&offset=0`