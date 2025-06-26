/**
 * Live Mode Configuration Types
 *
 * This file contains the type definitions and configuration options
 * for implementing live mode (real-time updates) in data tables.
 */

import { API_ENDPOINTS } from "./server";

export interface LiveModeConfig {
  /** Whether live mode is enabled for this table */
  enabled: boolean;

  /** The SSE endpoint for real-time updates */
  endpoint: API_ENDPOINTS | string;

  /** Optional configuration options */
  options?: {
    /** How often to poll for updates when using polling fallback (in ms) */
    pollInterval?: number;

    /** Maximum number of reconnection attempts */
    maxReconnectAttempts?: number;

    /** Whether to show connection status indicator */
    showConnectionStatus?: boolean;

    /** Custom handler for new data */
    onNewData?: (data: any) => void;

    /** Custom error handler */
    onError?: (error: string) => void;
  };
}

export type LiveModeTableConfig = {
  enabled: boolean;
  endpoint: API_ENDPOINTS | string;
  autoRefresh?: boolean;
  showToggle?: boolean;
};

export interface LiveModeState {
  /** Whether currently connected to live updates */
  connected: boolean;

  /** Current error message, if any */
  error: string | null;

  /** Timestamp of last heartbeat received */
  lastHeartbeat: Date | null;

  /** Number of new items waiting to be refreshed */
  newItemsCount: number;

  /** Whether to show the new items banner */
  showNewItemsBanner: boolean;
}

export interface LiveModeActions {
  /** Manually refresh the data and clear new items counter */
  refreshData: () => void;

  /** Dismiss the new items banner without refreshing */
  dismissBanner: () => void;

  /** Manually connect to live updates */
  connect: () => void;

  /** Manually disconnect from live updates */
  disconnect: () => void;
}

/**
 * Complete live mode interface combining state and actions
 */
export type LiveModeInterface = LiveModeState & LiveModeActions;

/**
 * Server-Sent Events message types that can be received
 */
export interface SSEMessage {
  /** Event type identifier */
  type: "connected" | "new-entry" | "heartbeat" | "error";

  /** Message data payload */
  data: any;

  /** Optional timestamp */
  timestamp?: string;
}

/**
 * Configuration for different types of live updates
 */
export interface LiveUpdateStrategy {
  /** Strategy name */
  name: "sse" | "polling" | "websocket";

  /** Whether this strategy is available/supported */
  supported: boolean;

  /** Configuration specific to this strategy */
  config?: {
    /** For SSE: endpoint URL */
    endpoint?: string;

    /** For polling: interval in milliseconds */
    interval?: number;

    /** For WebSocket: socket URL */
    socketUrl?: string;
  };
}

/**
 * Standard live mode endpoints for different resources
 * Add new endpoints here as you implement live mode for other tables
 */
export const LIVE_MODE_ENDPOINTS = {
  AUDIT_LOGS: "/audit-logs/live",
  SHIPMENTS: "/shipments/live",
} as const;

/**
 * Helper type to ensure live mode endpoints are valid
 */
export type LiveModeEndpoint =
  (typeof LIVE_MODE_ENDPOINTS)[keyof typeof LIVE_MODE_ENDPOINTS];

/**
 * Utility function to create live mode configuration
 */
export function createLiveModeConfig(
  endpoint: LiveModeEndpoint | string,
  options?: LiveModeConfig["options"],
): LiveModeConfig {
  return {
    enabled: true,
    endpoint,
    options,
  };
}

/**
 * Default live mode configuration
 */
export const DEFAULT_LIVE_MODE_CONFIG: Required<LiveModeConfig> = {
  enabled: false,
  endpoint: "",
  options: {
    pollInterval: 2000,
    maxReconnectAttempts: 5,
    showConnectionStatus: true,
  },
};
