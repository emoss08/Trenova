import type { User } from "@/types/user";
import { api } from "./api";

export type ListUsersParams = {
  limit?: number;
  offset?: number;
  query?: string;
};

export async function getUser(id: string): Promise<User> {
  return api.get<User>(`/users/${id}`);
}

export type UserActivity = {
  id: string;
  userId: string;
  activityType: string;
  description: string;
  metadata?: Record<string, unknown>;
  ipAddress?: string;
  userAgent?: string;
  timestamp: number;
};

export async function resetUserPassword(userId: string): Promise<void> {
  return api.post(`/users/${userId}/reset-password`, {});
}

export async function unlockUser(userId: string): Promise<User> {
  return api.post<User>(`/users/${userId}/unlock`, {});
}

export async function lockUser(userId: string): Promise<User> {
  return api.post<User>(`/users/${userId}/lock`, {});
}
