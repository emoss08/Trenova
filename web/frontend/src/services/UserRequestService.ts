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

import axios from "@/lib/axiosConfig";
import { User, UserNotification, UserReportResponse } from "@/types/accounts";

/**
 * Fetches the details of the user with the specified ID.
 * @param id - The ID of the user to fetch details for.
 * @returns A promise that resolves to the user's details.
 */
export async function getUserDetails(id: string): Promise<User> {
  const response = await axios.get(`/users/${id}/`);
  return response.data;
}

/**
 * Fetches users from the server.
 * @returns A promise that resolves to an array of users.
 */
export async function getUsers(): Promise<Array<User>> {
  const response = await axios.get("/users/", {
    params: {
      limit: 100,
    },
  });
  return response.data.results;
}

/**
 * Fetches user reports from the server.
 * @returns A promise that resolves to an array of user reports.
 */
export async function getUserReports(): Promise<UserReportResponse> {
  const response = await axios.get("/user_reports/");
  return response.data;
}

/**
 * Fetches the current user's notifications from the server.
 * @returns A promise that resolves to the user's notifications.
 */
export async function getUserNotifications(
  markAsRead: boolean,
): Promise<UserNotification> {
  const response = await axios.get("/user-notifications/", {
    params: {
      maxAmount: 10,
      markAsRead: markAsRead,
    },
  });
  return response.data;
}

/**
 * Posts a user profile picture to the server.
 * @param profilePicture Profile picture to be uploaded
 * @returns A promise that resolves to the user's details.
 */
export async function postUserProfilePicture(
  profilePicture: File,
): Promise<User> {
  const formData = new FormData();
  formData.append("profilePicture", profilePicture);
  const response = await axios.post("users/upload-profile-picture/", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });

  return response.data;
}

export async function clearProfilePicture(): Promise<void> {
  await axios.post("users/clear-profile-pic/");
}

export async function getAuthenticatedUser(): Promise<User> {
  const response = await axios.get("/users/me/", {
    withCredentials: true,
  });
  return response.data;
}
