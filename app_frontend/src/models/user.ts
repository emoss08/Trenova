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

export interface AuthModel {
  token: string;
}

export interface UserAuthModel {
  id: string;
  username: string;
  email: string;
  first_name: string;
  last_name: string;
  full_name?: string;
  organization_id?: string;
  department_id?: string;
  is_staff: boolean;
  is_superuser: boolean;
  address_line_1: string;
  address_line_2?: string;
  city: string;
  state: string;
  zip_code: string;
  full_address: string;
  phone_number: string;
  phone_verified: boolean;
  job_title?: string;
  token: string;
}

export interface JobTitleModel {
  id: string;
  is_active: boolean;
  name: string;
  description: string;
  job_function: string;
}
