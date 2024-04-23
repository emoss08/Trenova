import { JobFunctionChoiceProps, TimezoneChoices } from "@/lib/choices";
import { StatusChoiceProps } from "@/types/index";

export type UserFavorite = {
  id: string;
  userID: string;
  created: string;
  pageLink: string;
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
  isSuperAdmin: boolean;
  isAdmin: boolean;
  status: StatusChoiceProps;
  timezone: TimezoneChoices;
  PhoneNumber?: string;
  userPermissions?: string[];
  profilePicUrl: string;
};

export type UserFormValues = {
  organization: string;
  username: string;
  department?: string;
  email: string;
  isSuperAdmin: boolean;
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
  userID: string;
  isRead: boolean;
  title: string;
  description: string;
  actionUrl: string;
  createdAt: string;
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
