/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { type JobFunctionChoiceProps } from "@/lib/choices";
import { type TimezoneChoices } from "@/lib/timezone";
import { type StatusChoiceProps } from ".";
import { type BaseModel } from "./organization";

interface UserRole extends BaseModel {
  name: string;
  description: string;
  permissions: UserPermission[];
}

export interface UserPermission extends BaseModel {
  codename: string;
  description: string;
  action: string;
  label: string;
  readDescription?: string;
  writeDescription?: string;
  resourceId: string;
}

export type UserFavorite = {
  id: string;
  userID: string;
  created: string;
  pageLink: string;
};

export interface User extends BaseModel {
  id: string;
  username: string;
  name: string;
  email: string;
  isAdmin: boolean;
  status: StatusChoiceProps;
  timezone: TimezoneChoices;
  phoneNumber?: string;
  profilePicUrl?: string;
  thumbnailUrl?: string;
  lastLogin?: string | null;
  role: UserRole[];
}

export type UserFormValues = Omit<
  User,
  | "organizationId"
  | "createdAt"
  | "updatedAt"
  | "id"
  | "version"
  | "role"
  | "lastLogin"
  | "profilePicUrl"
  | "thumbnailUrl"
>;
export type JobTitle = {
  id: string;
  organization: string;
  name: string;
  description?: string | null;
  status: StatusChoiceProps;
  jobFunction: JobFunctionChoiceProps | "";
  created: string;
  modified: string;
};

export type JobTitleFormValues = Omit<
  JobTitle,
  "id" | "organization" | "created" | "modified"
>;

export type UserReport = {
  id: string;
  user: string;
  report: string;
  created: string;
  fileName: string;
  modified: string;
};

export type UserReportResponse = {
  count: number;
  next?: string | null;
  previous?: string | null;
  results: UserReport[];
};

export type Notification = {
  id: number;
  userID: string;
  isRead: boolean;
  title: string;
  description: string;
  actionUrl: string;
  createdAt: string;
};

export type UserNotification = {
  unreadCount: number;
  unreadList?: Notification[] | null;
};

export type GroupType = {
  id: string;
  name: string;
  codename: string;
};
