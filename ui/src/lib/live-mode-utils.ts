/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/**
 * Utility functions for implementing live mode in data tables
 */

import {
  createLiveModeConfig,
  LIVE_MODE_ENDPOINTS,
  LiveModeConfig,
} from "@/types/live-mode";

/**
 * Quick setup function for common live mode configurations
 */
export function setupLiveMode(
  endpoint: keyof typeof LIVE_MODE_ENDPOINTS | string,
  options?: {
    /** Custom polling interval in milliseconds (default: 2000) */
    pollInterval?: number;
    /** Enable/disable live mode (default: true) */
    enabled?: boolean;
    /** Custom new data handler */
    onNewData?: (data: any) => void;
    /** Custom error handler */
    onError?: (error: string) => void;
  },
): LiveModeConfig {
  const endpointUrl =
    typeof endpoint === "string" ? endpoint : LIVE_MODE_ENDPOINTS[endpoint];

  return createLiveModeConfig(endpointUrl, {
    pollInterval: options?.pollInterval || 2000,
    onNewData: options?.onNewData,
    onError: options?.onError,
  });
}

/**
 * Pre-configured live mode setups for common resources
 */
export const LiveModePresets = {
  /**
   * Setup for audit logs with optimized configuration
   */
  auditLogs: () => ({
    enabled: true,
    endpoint: LIVE_MODE_ENDPOINTS.AUDIT_LOGS,
    autoRefresh: true,
    showToggle: true,
    options: {
      batchWindow: 50, // Smaller batch window for audit logs
      debounceDelay: 300, // Shorter debounce for more responsive updates
      pollInterval: 2000,
      maxReconnectAttempts: 5,
      showConnectionStatus: true,
    },
  }),

  /**
   * Setup for shipments with optimized batching configuration
   */
  shipments: () => ({
    enabled: true,
    endpoint: LIVE_MODE_ENDPOINTS.SHIPMENTS,
    autoRefresh: true,
    showToggle: true,
    options: {
      batchWindow: 100, // Batch events within 100ms window
      debounceDelay: 500, // Wait 500ms after last event before refreshing
      pollInterval: 2000,
      maxReconnectAttempts: 5,
      showConnectionStatus: true,
    },
  }),

  /**
   * Setup for high-frequency updates (faster polling)
   */
  highFrequency: (endpoint: string) =>
    setupLiveMode(endpoint, {
      pollInterval: 1000, // 1 second
    }),

  /**
   * Setup for low-frequency updates (slower polling)
   */
  lowFrequency: (endpoint: string) =>
    setupLiveMode(endpoint, {
      pollInterval: 5000, // 5 seconds
    }),

  /**
   * Custom setup with logging
   */
  withLogging: (endpoint: string) => setupLiveMode(endpoint),
} as const;

/**
 * Utility to check if live mode is supported for a given resource
 */
export function isLiveModeSupported(resource: string): boolean {
  const supportedResources = Object.keys(LIVE_MODE_ENDPOINTS);
  return supportedResources.includes(resource.toUpperCase().replace("-", "_"));
}

/**
 * Get the live mode endpoint for a resource if supported
 */
export function getLiveModeEndpoint(resource: string): string | null {
  const key = resource
    .toUpperCase()
    .replace("-", "_") as keyof typeof LIVE_MODE_ENDPOINTS;
  return LIVE_MODE_ENDPOINTS[key] || null;
}

/**
 * Helper to create a custom live mode configuration with validation
 */
export function createCustomLiveMode(config: {
  endpoint: string;
  enabled?: boolean;
  pollInterval?: number;
  maxReconnectAttempts?: number;
  onNewData?: (data: any) => void;
  onError?: (error: string) => void;
}): LiveModeConfig {
  if (!config.endpoint) {
    throw new Error("Live mode endpoint is required");
  }

  if (config.pollInterval && config.pollInterval < 500) {
    console.warn(
      "Live mode poll interval less than 500ms may cause performance issues",
    );
  }

  return {
    enabled: config.enabled ?? true,
    endpoint: config.endpoint,
    options: {
      pollInterval: config.pollInterval || 2000,
      maxReconnectAttempts: config.maxReconnectAttempts || 5,
      showConnectionStatus: true,
      onNewData: config.onNewData,
      onError: config.onError,
    },
  };
}

/**
 * Environment-based live mode configuration
 */
export function getEnvironmentLiveMode(
  endpoint: string,
  environment: "development" | "staging" | "production" = "production",
): LiveModeConfig {
  const baseConfig = setupLiveMode(endpoint);

  switch (environment) {
    case "development":
      return {
        ...baseConfig,
        options: {
          ...baseConfig.options,
          pollInterval: 1000, // Faster updates in dev
        },
      };

    case "staging":
      return {
        ...baseConfig,
        options: {
          ...baseConfig.options,
          pollInterval: 1500,
          onError: (error) => console.warn("[STAGING] Live mode error:", error),
        },
      };

    case "production":
    default:
      return baseConfig;
  }
}

/**
 * Batch configuration for multiple tables
 */
export function configureLiveModeForTables(
  tables: Array<{
    name: string;
    endpoint: string;
    config?: Partial<LiveModeConfig["options"]>;
  }>,
): Record<string, LiveModeConfig> {
  return tables.reduce(
    (acc, table) => {
      acc[table.name] = createCustomLiveMode({
        endpoint: table.endpoint,
        ...table.config,
      });
      return acc;
    },
    {} as Record<string, LiveModeConfig>,
  );
}
