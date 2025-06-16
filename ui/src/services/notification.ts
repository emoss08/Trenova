import { http } from "@/lib/http-client";
import type {
  CreateNotificationPreferenceInput,
  NotificationHistory,
  NotificationHistoryListResponse,
  NotificationPreference,
  NotificationPreferenceListResponse,
  NotificationPreferenceResponse,
  UpdateNotificationPreferenceInput,
} from "@/types/notification";

export class NotificationAPI {
  // Notification Preferences
  async getPreferences(params?: { 
    resource?: string; 
    isActive?: boolean 
  }): Promise<NotificationPreferenceListResponse> {
    const response = await http.get<NotificationPreferenceListResponse>(
      "/notification-preferences",
      { 
        params: params ? {
          resource: params.resource,
          isActive: params.isActive?.toString()
        } : undefined
      }
    );
    return response.data;
  }

  async getUserPreferences(userId: string): Promise<NotificationPreferenceListResponse> {
    const response = await http.get<NotificationPreferenceListResponse>(
      "/notification-preferences/user",
      { params: { userId } }
    );
    return response.data;
  }

  async getPreference(id: string): Promise<NotificationPreferenceResponse> {
    const response = await http.get<NotificationPreferenceResponse>(
      `/notification-preferences/${id}`
    );
    return response.data;
  }

  async createPreference(
    data: CreateNotificationPreferenceInput
  ): Promise<NotificationPreferenceResponse> {
    const response = await http.post<NotificationPreferenceResponse>(
      "/notification-preferences",
      data
    );
    return response.data;
  }

  async updatePreference(
    id: string,
    data: UpdateNotificationPreferenceInput
  ): Promise<NotificationPreferenceResponse> {
    const response = await http.put<NotificationPreferenceResponse>(
      `/notification-preferences/${id}`,
      data
    );
    return response.data;
  }

  async deletePreference(id: string): Promise<void> {
    await http.delete(`/notification-preferences/${id}`);
  }

  // Notification History
  async getHistory(params?: {
    limit?: number;
    offset?: number;
    unreadOnly?: boolean;
    resource?: string;
    priority?: string;
    startDate?: number;
    endDate?: number;
  }): Promise<NotificationHistoryListResponse> {
    const response = await http.get<NotificationHistoryListResponse>(
      "/notifications/history",
      { 
        params: params ? {
          limit: params.limit?.toString(),
          offset: params.offset?.toString(),
          unreadOnly: params.unreadOnly?.toString(),
          resource: params.resource,
          priority: params.priority,
          startDate: params.startDate?.toString(),
          endDate: params.endDate?.toString()
        } : undefined
      }
    );
    return response.data;
  }

  async markAsRead(notificationId: string): Promise<void> {
    await http.post(`/notifications/${notificationId}/read`);
  }

  async markAllAsRead(): Promise<void> {
    await http.post("/notifications/read-all");
  }

  async dismiss(notificationId: string): Promise<void> {
    await http.post(`/notifications/${notificationId}/dismiss`);
  }

  async getUnreadCount(): Promise<{ count: number }> {
    const response = await http.get<{ count: number }>("/notifications/unread-count");
    return response.data;
  }

  // Batch operations
  async markMultipleAsRead(notificationIds: string[]): Promise<void> {
    await http.post("/notifications/batch-read", { notificationIds });
  }

  async dismissMultiple(notificationIds: string[]): Promise<void> {
    await http.post("/notifications/batch-dismiss", { notificationIds });
  }
}