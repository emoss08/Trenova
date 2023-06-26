/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { ApiResponse } from "@/types/server";

export type UserProfile = {
  id: string;
  organization: string;
  first_name: string;
  last_name: string;
  user: string;
  job_title: string;
  address_line_1: string;
  address_line_2?: string;
  city: string;
  state: string;
  zip_code: string;
  phone_number?: string;
  profile_picture: string;
  is_phone_verified: boolean;
};

export type User = {
  id: string;
  username: string;
  organization: string;
  email: string;
  department?: string;
  date_joined: string;
  is_superuser: boolean;
  is_staff: boolean;
  is_active: boolean;
  groups: string[];
  user_permissions: string[];
  online: boolean;
  last_login: string;
  profile?: UserProfile;
};

export interface UserFormValues {
  id: string;
  username: string;
  email: string;
  profile: {
    organization: string;
    first_name: string;
    last_name: string;
    address_line_1: string;
    address_line_2?: string;
    city: string;
    state: string;
    zip_code: string;
    phone_number?: string;
  };
}

export type JobTitle = {
  id: string;
  name: string;
  description: string;
  is_active: boolean;
  organization: string;
};

export type UserReport = {
  id: string;
  user: string;
  report: string;
  created: string;
  file_name: string;
  modified: string;
};

export type Notification = {
  id: number;
  level: string;
  recipient: string;
  unread: boolean;
  actor_content_type: number;
  actor_object_id: string;
  verb: string;
  description: string;
  target_content_type: number;
  target_object_id: string;
  action_object_content_type: number;
  action_object_object_id: string;
  timestamp: string;
  public: boolean;
  deleted: boolean;
  emailed: boolean;
  data: string;
  slug: number;
  actor: string;
  action_object: string;
};

export type UserNotification = {
  unread_count: number;
  unread_list: Notification[];
};
