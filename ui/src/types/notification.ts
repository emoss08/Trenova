export type UpdateType =
  | "status_change"
  | "assignment"
  | "location_change"
  | "document_upload"
  | "price_change"
  | "compliance_change"
  | "general";

export type NotificationChannel = "user" | "role" | "global";

export type NotificationPriority = "critical" | "high" | "medium" | "low";

export type DeliveryStatus = "pending" | "delivered" | "failed" | "expired";

export interface NotificationPreference {
  id: string;
  userId: string;
  organizationId: string;
  businessUnitId: string;
  resource: string;
  updateTypes: UpdateType[];
  notifyOnAllUpdates: boolean;
  notifyOnlyOwnedRecords: boolean;
  excludedUserIds: string[];
  includedRoleIds: string[];
  preferredChannels: NotificationChannel[];
  quietHoursEnabled: boolean;
  quietHoursStart?: string; // Time in HH:mm format
  quietHoursEnd?: string; // Time in HH:mm format
  timezone: string;
  batchNotifications: boolean;
  batchIntervalMinutes: number;
  isActive: boolean;
  version: number;
  createdAt: number;
  updatedAt: number;
}

export interface NotificationHistory {
  id: string;
  notificationId: string;
  userId: string;
  organizationId: string;
  businessUnitId: string;
  entityType?: string;
  entityId?: string;
  updateType?: UpdateType;
  updatedById?: string;
  title: string;
  message: string;
  priority: NotificationPriority;
  channel: NotificationChannel;
  eventType: string;
  data?: Record<string, any>;
  deliveryStatus: DeliveryStatus;
  deliveredAt?: number;
  failureReason?: string;
  retryCount: number;
  readAt?: number;
  dismissedAt?: number;
  clickedAt?: number;
  actionTaken?: string;
  groupId?: string;
  groupPosition?: number;
  createdAt: number;
  expiresAt?: number;
}

export interface NotificationRateLimit {
  id: string;
  organizationId: string;
  name: string;
  description?: string;
  resource?: string;
  eventType?: string;
  priority?: NotificationPriority;
  maxNotifications: number;
  period: "minute" | "hour" | "day";
  applyToAllUsers: boolean;
  userId?: string;
  roleId?: string;
  isActive: boolean;
  version: number;
  createdAt: number;
  updatedAt: number;
}

export interface CreateNotificationPreferenceInput {
  resource: string;
  updateTypes?: UpdateType[];
  notifyOnAllUpdates?: boolean;
  notifyOnlyOwnedRecords?: boolean;
  excludedUserIds?: string[];
  includedRoleIds?: string[];
  preferredChannels: NotificationChannel[];
  quietHoursEnabled?: boolean;
  quietHoursStart?: string;
  quietHoursEnd?: string;
  timezone?: string;
  batchNotifications?: boolean;
  batchIntervalMinutes?: number;
}

export interface UpdateNotificationPreferenceInput extends Partial<CreateNotificationPreferenceInput> {
  isActive?: boolean;
}

export interface NotificationPreferenceResponse {
  data: NotificationPreference;
}

export interface NotificationPreferenceListResponse {
  data: NotificationPreference[];
  totalCount: number;
}

export interface NotificationHistoryListResponse {
  data: NotificationHistory[];
  totalCount: number;
  hasMore: boolean;
}

// Resource types that support notifications
export const NOTIFICATION_RESOURCES = [
  "shipment",
  "worker",
  "customer",
  "tractor",
  "trailer",
  "location",
  "commodity",
] as const;

export type NotificationResource = typeof NOTIFICATION_RESOURCES[number];

// Update type display names
export const UPDATE_TYPE_LABELS: Record<UpdateType, string> = {
  status_change: "Status Changes",
  assignment: "Assignments",
  location_change: "Location Changes",
  document_upload: "Document Uploads",
  price_change: "Price Changes",
  compliance_change: "Compliance Changes",
  general: "General Updates",
};

// Priority colors and icons
export const PRIORITY_CONFIG = {
  critical: {
    color: "text-red-600",
    bgColor: "bg-red-50",
    borderColor: "border-red-200",
    icon: "AlertTriangle",
  },
  high: {
    color: "text-orange-600",
    bgColor: "bg-orange-50",
    borderColor: "border-orange-200",
    icon: "AlertCircle",
  },
  medium: {
    color: "text-yellow-600",
    bgColor: "bg-yellow-50",
    borderColor: "border-yellow-200",
    icon: "Info",
  },
  low: {
    color: "text-blue-600",
    bgColor: "bg-blue-50",
    borderColor: "border-blue-200",
    icon: "Bell",
  },
} as const;