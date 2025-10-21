/*
 * Copyright 2025 Eric Moss
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

export class DockerAPI {
  async listContainers(all = false): Promise<Container[]> {
    const response = await http.get<{ containers: Container[] }>(
      `/docker/containers?all=${all}`,
    );
    return response.data.containers;
  }

  async inspectContainer(id: string): Promise<ContainerInspect> {
    const response = await http.get<ContainerInspect>(
      `/docker/containers/${id}`,
    );
    return response.data;
  }

  async startContainer(id: string): Promise<{ message: string }> {
    const response = await http.post<{ message: string }>(
      `/docker/containers/${id}/start`,
    );
    return response.data;
  }

  async stopContainer(
    id: string,
    timeout?: number,
  ): Promise<{ message: string }> {
    const params = timeout ? `?timeout=${timeout}` : "";
    const response = await http.post<{ message: string }>(
      `/docker/containers/${id}/stop${params}`,
    );
    return response.data;
  }

  async restartContainer(
    id: string,
    timeout?: number,
  ): Promise<{ message: string }> {
    const params = timeout ? `?timeout=${timeout}` : "";
    const response = await http.post<{ message: string }>(
      `/docker/containers/${id}/restart${params}`,
    );
    return response.data;
  }

  async removeContainer(
    id: string,
    force = false,
  ): Promise<{ message: string }> {
    const response = await http.delete<{ message: string }>(
      `/docker/containers/${id}?force=${force}`,
    );
    return response.data;
  }

  async getContainerLogs(
    id: string,
    tail = "100",
    follow = false,
  ): Promise<string[]> {
    const response = await http.get<{ logs: string[] }>(
      `/docker/containers/${id}/logs?tail=${tail}&follow=${follow}`,
    );
    return response.data.logs;
  }

  async getContainerStats(id: string): Promise<ContainerStats> {
    const response = await http.get<ContainerStats>(
      `/docker/containers/${id}/stats`,
    );
    return response.data;
  }

  streamContainerStats(
    id: string,
    onStats: (stats: ContainerStats) => void,
    onError?: (error: string) => void,
    onConnected?: () => void,
  ): EventSource {
    const eventSource = new EventSource(
      `${API_URL}/docker/containers/${id}/stats/stream`,
      {
        withCredentials: true,
      },
    );

    // Handle connected event
    eventSource.addEventListener("connected", () => {
      onConnected?.();
    });

    // Handle stats event
    eventSource.addEventListener("stats", (event) => {
      try {
        const stats = JSON.parse(event.data) as ContainerStats;
        onStats(stats);
      } catch (error) {
        console.error("Failed to parse stats data:", error);
        onError?.("Failed to parse stats data");
      }
    });

    // Handle server error event
    eventSource.addEventListener("error", (event: MessageEvent) => {
      if (event.data) {
        try {
          const error = JSON.parse(event.data) as { error: string };
          onError?.(error.error);
        } catch {
          onError?.("Unknown error occurred");
        }
      }
    });

    // Handle connection errors
    eventSource.onerror = () => {
      if (eventSource.readyState === EventSource.CLOSED) {
        onError?.("Connection closed");
        eventSource.close();
      } else if (eventSource.readyState === EventSource.CONNECTING) {
        // Attempting to reconnect
        console.log("Reconnecting to stats stream...");
      }
    };

    return eventSource;
  }

  async listImages(): Promise<DockerImage[]> {
    const response = await http.get<{ images: DockerImage[] }>(
      "/docker/images",
    );
    return response.data.images;
  }

  async pullImage(imageName: string): Promise<{ message: string }> {
    const response = await http.post<{ message: string }>(
      "/docker/images/pull",
      { imageName },
    );
    return response.data;
  }

  async removeImage(id: string, force = false): Promise<{ message: string }> {
    const response = await http.delete<{ message: string }>(
      `/docker/images/${id}?force=${force}`,
    );
    return response.data;
  }

  async listVolumes(): Promise<{
    volumes: DockerVolume[];
    Warnings: string[];
  }> {
    const response = await http.get<{
      volumes: DockerVolume[];
      Warnings: string[];
    }>("/docker/volumes");
    return response.data;
  }

  async createVolume(
    name: string,
    driver = "local",
    labels?: Record<string, string>,
  ): Promise<DockerVolume> {
    const response = await http.post<DockerVolume>("/docker/volumes", {
      name,
      driver,
      labels,
    });
    return response.data;
  }

  async removeVolume(id: string, force = false): Promise<{ message: string }> {
    const response = await http.delete<{ message: string }>(
      `/docker/volumes/${id}?force=${force}`,
    );
    return response.data;
  }

  async listNetworks(): Promise<DockerNetwork[]> {
    const response = await http.get<{ networks: DockerNetwork[] }>(
      "/docker/networks",
    );
    return response.data.networks;
  }

  async inspectNetwork(id: string): Promise<DockerNetwork> {
    const response = await http.get<DockerNetwork>(`/docker/networks/${id}`);
    return response.data;
  }

  async removeNetwork(id: string): Promise<{ message: string }> {
    const response = await http.delete<{ message: string }>(
      `/docker/networks/${id}`,
    );
    return response.data;
  }

  async getSystemInfo(): Promise<DockerSystemInfo> {
    const response = await http.get<DockerSystemInfo>("/docker/system/info");
    return response.data;
  }

  async getDiskUsage(): Promise<DiskUsage> {
    const response = await http.get<DiskUsage>("/docker/system/disk-usage");
    return response.data;
  }

  async pruneSystem(): Promise<SystemPruneReport> {
    const response = await http.post<SystemPruneReport>("/docker/system/prune");
    return response.data;
  }
}

export const dockerAPI = new DockerAPI();
