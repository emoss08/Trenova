/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
