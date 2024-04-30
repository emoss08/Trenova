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
  const response = await axios.post("users/profile-picture/", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });

  return response.data;
}

export async function getAuthenticatedUser(): Promise<User> {
  const response = await axios.get("/users/me/", {
    withCredentials: true,
  });
  return response.data;
}
