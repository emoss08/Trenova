/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

export interface WebSocketMessage {
  type: string;
  data: any;
  timestamp?: string;
}

export interface NotificationMessage {
  id: string;
  organizationId: string;
  businessUnitId?: string;
  targetUserId?: string;
  targetRoleId?: string;
  eventType: string;
  priority: 'critical' | 'high' | 'medium' | 'low';
  channel: 'global' | 'user' | 'role';
  title: string;
  message: string;
  data?: Record<string, any>;
  relatedEntities?: RelatedEntity[];
  actions?: NotificationAction[];
  expiresAt?: number;
  deliveredAt?: number;
  readAt?: number;
  dismissedAt?: number;
  createdAt: number;
  updatedAt: number;
  deliveryStatus: 'pending' | 'delivered' | 'failed' | 'expired';
  retryCount: number;
  maxRetries: number;
  source: string;
  jobId?: string;
  correlationId?: string;
  tags?: string[];
  version: number;
}

export interface RelatedEntity {
  type: string;
  id: string;
  name?: string;
}

export interface NotificationAction {
  type: 'link' | 'button' | 'dismiss';
  label: string;
  url?: string;
  action?: string;
  variant?: 'default' | 'destructive' | 'outline' | 'secondary';
}

export interface WebSocketConnectionState {
  socket: WebSocket | null;
  isConnected: boolean;
  connectionState: 'disconnected' | 'connecting' | 'connected' | 'reconnecting';
  reconnectAttempts: number;
  lastError?: string;
}

export interface WebSocketSubscription {
  room: string;
  userId: string;
  organizationId: string;
  businessUnitId?: string;
  roles: string[];
}

export interface WebSocketConfig {
  url: string;
  reconnectInterval: number;
  maxReconnectAttempts: number;
  heartbeatInterval: number;
}