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

import { JobFunctionChoiceProps } from "@/utils/apps/accounts";
import { StatusChoiceProps } from "@/types";

export type UserProfile = {
  id: string;
  organization: string;
  firstName: string;
  lastName: string;
  user: string;
  jobTitle: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  zipCode: string;
  phoneNumber?: string;
  profilePicture: string;
  isPhoneVerified: boolean;
};

export type User = {
  id: string;
  username: string;
  organization: string;
  email: string;
  department?: string;
  dateJoined: string;
  isSuperuser: boolean;
  isStaff: boolean;
  isActive: boolean;
  groups: string[];
  userPermissions: string[];
  online: boolean;
  lastLogin: string;
  profile?: UserProfile;
};

export type UserFormValues = {
  organization: string;
  username: string;
  department?: string;
  email: string;
  isSuperuser: boolean;
  profile: {
    jobTitle: string;
    organization: string;
    firstName: string;
    lastName: string;
    addressLine1: string;
    addressLine2?: string;
    city: string;
    state: string;
    zipCode: string;
    phoneNumber?: string;
  };
};

export type JobTitle = {
  id: string;
  organization: string;
  name: string;
  description?: string | null;
  status: StatusChoiceProps;
  jobFunction: JobFunctionChoiceProps | "";
};

export type JobTitleFormValues = Omit<JobTitle, "id" | "organization">;

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
