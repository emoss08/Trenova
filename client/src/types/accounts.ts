/*
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

import { JobFunctionChoiceProps, TimezoneChoices } from "@/lib/choices";
import { StatusChoiceProps } from "@/types/index";

export type UserFavorite = {
  id: string;
  user: string;
  created: string;
  page: string;
};

/**
 * MinimalUser is similar to the User type ,but does provide all the fields.
 */
export type MinimalUser = {
  id: string;
  username: string;
  email: string;
};

export type User = {
  id: string;
  businessUnitId: string;
  organizationId: string;
  username: string;
  name: string;
  email: string;
  dateJoined: string;
  isSuperuser: boolean;
  isAdmin: boolean;
  status: StatusChoiceProps;
  timezone: TimezoneChoices;
  profilePicUrl?: string;
  thumbnailUrl?: string;
  PhoneNumber?: string;
  userPermissions: string[];
};

export type UserFormValues = {
  organization: string;
  username: string;
  department?: string;
  email: string;
  isSuperuser: boolean;
};

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
  level: string;
  recipient: string;
  unread: boolean;
  actorContentType: number;
  actorObjectId: string;
  verb: string;
  description: string;
  targetContentType: number;
  targetObjectId: string;
  target_object_id: string;
  actionObjectContentType: number;
  actionObjectObjectId: string;
  timestamp: string;
  public: boolean;
  deleted: boolean;
  emailed: boolean;
  data: string;
  slug: number;
  actor: string;
  actionObject: string;
};

export type UserNotification = {
  unreadCount: number;
  unreadList: Notification[];
};

export type GroupType = {
  id: string;
  name: string;
  codename: string;
};
