/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { API_URL } from "@/constants/env";
import { http } from "@/lib/http-client";
import {
  Container,
  ContainerInspect,
  ContainerStats,
  DiskUsage,
  DockerImage,
  DockerNetwork,
  DockerSystemInfo,
  DockerVolume,
  SystemPruneReport,
} from "@/types/docker";

export const dockerAPI = {
  // Container operations
  listContainers: async (all = false) => {
    const response = await http.get<{ containers: Container[] }>(
      `/docker/containers?all=${all}`,
    );
    return response.data.containers;
  },

  inspectContainer: async (id: string) => {
    const response = await http.get<ContainerInspect>(
      `/docker/containers/${id}`,
    );
    return response.data;
  },

  startContainer: async (id: string) => {
    const response = await http.post<{ message: string }>(
      `/docker/containers/${id}/start`,
    );
    return response.data;
  },

  stopContainer: async (id: string, timeout?: number) => {
    const params = timeout ? `?timeout=${timeout}` : "";
    const response = await http.post<{ message: string }>(
      `/docker/containers/${id}/stop${params}`,
    );
    return response.data;
  },

  restartContainer: async (id: string, timeout?: number) => {
    const params = timeout ? `?timeout=${timeout}` : "";
    const response = await http.post<{ message: string }>(
      `/docker/containers/${id}/restart${params}`,
    );
    return response.data;
  },

  removeContainer: async (id: string, force = false) => {
    const response = await http.delete<{ message: string }>(
      `/docker/containers/${id}?force=${force}`,
    );
    return response.data;
  },

  getContainerLogs: async (
    id: string,
    tail = "100",
    follow = false,
  ): Promise<string[]> => {
    const response = await http.get<{ logs: string[] }>(
      `/docker/containers/${id}/logs?tail=${tail}&follow=${follow}`,
    );
    return response.data.logs;
  },

  getContainerStats: async (id: string) => {
    const response = await http.get<ContainerStats>(
      `/docker/containers/${id}/stats`,
    );
    return response.data;
  },

  // SSE stream for container stats
  streamContainerStats: (
    id: string,
    onStats: (stats: ContainerStats) => void,
    onError?: (error: string) => void,
  ) => {
    const eventSource = new EventSource(
      `${API_URL}/docker/containers/${id}/stats/stream`,
    );

    eventSource.addEventListener("stats", (event) => {
      const stats = JSON.parse(event.data) as ContainerStats;
      onStats(stats);
    });

    eventSource.addEventListener("error", (event) => {
      if (onError && event.data) {
        const error = JSON.parse(event.data as string) as { error: string };
        onError(error.error);
      }
    });

    eventSource.onerror = () => {
      if (onError) {
        onError("Connection lost");
      }
      eventSource.close();
    };

    return eventSource;
  },

  // Image operations
  listImages: async () => {
    const response = await http.get<{ images: DockerImage[] }>(
      "/docker/images",
    );
    return response.data.images;
  },

  pullImage: async (imageName: string) => {
    const response = await http.post<{ message: string }>(
      "/docker/images/pull",
      { imageName },
    );
    return response.data;
  },

  removeImage: async (id: string, force = false) => {
    const response = await http.delete<{ message: string }>(
      `/docker/images/${id}?force=${force}`,
    );
    return response.data;
  },

  // Volume operations
  listVolumes: async () => {
    const response = await http.get<{
      volumes: DockerVolume[];
      Warnings: string[];
    }>("/docker/volumes");
    return response.data;
  },

  createVolume: async (
    name: string,
    driver = "local",
    labels?: Record<string, string>,
  ) => {
    const response = await http.post<DockerVolume>("/docker/volumes", {
      name,
      driver,
      labels,
    });
    return response.data;
  },

  removeVolume: async (id: string, force = false) => {
    const response = await http.delete<{ message: string }>(
      `/docker/volumes/${id}?force=${force}`,
    );
    return response.data;
  },

  // Network operations
  listNetworks: async () => {
    const response = await http.get<{ networks: DockerNetwork[] }>(
      "/docker/networks",
    );
    return response.data.networks;
  },

  inspectNetwork: async (id: string) => {
    const response = await http.get<DockerNetwork>(`/docker/networks/${id}`);
    return response.data;
  },

  // System operations
  getSystemInfo: async () => {
    const response = await http.get<DockerSystemInfo>("/docker/system/info");
    return response.data;
  },

  getDiskUsage: async () => {
    const response = await http.get<DiskUsage>("/docker/system/disk-usage");
    return response.data;
  },

  pruneSystem: async () => {
    const response = await http.post<SystemPruneReport>("/docker/system/prune");
    return response.data;
  },
};
