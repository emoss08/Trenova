/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { formatBytes } from "@/lib/utils";
import { api } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  Database,
  HardDrive,
  Layers,
  Network,
  Package,
} from "lucide-react";

export function DockerOverview() {
  const { data: systemInfo, isLoading: loadingInfo } = useQuery({
    queryKey: ["docker", "system-info"],
    queryFn: api.docker.getSystemInfo,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  const { data: diskUsage, isLoading: loadingDisk } = useQuery({
    queryKey: ["docker", "disk-usage"],
    queryFn: api.docker.getDiskUsage,
    refetchInterval: 30000,
  });

  if (loadingInfo || loadingDisk) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-6">
        {[...Array(6)].map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-4 w-4" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-7 w-16" />
              <Skeleton className="mt-1 h-3 w-24" />
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-6">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Containers</CardTitle>
          <Package className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {systemInfo?.ContainersRunning || 0}
          </div>
          <p className="text-xs text-muted-foreground">
            {systemInfo?.Containers || 0} total
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Images</CardTitle>
          <Layers className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{systemInfo?.Images || 0}</div>
          <p className="text-xs text-muted-foreground">
            {formatBytes(diskUsage?.LayersSize || 0)}
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Volumes</CardTitle>
          <Database className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {diskUsage?.Volumes?.length || 0}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Networks</CardTitle>
          <Network className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {systemInfo?.Plugins?.Network?.length || 0}
          </div>
          <p className="text-xs text-muted-foreground">drivers</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">CPU</CardTitle>
          <Activity className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{systemInfo?.NCPU || 0}</div>
          <p className="text-xs text-muted-foreground">cores available</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Memory</CardTitle>
          <HardDrive className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {formatBytes(systemInfo?.MemTotal || 0)}
          </div>
          <p className="text-xs text-muted-foreground">total available</p>
        </CardContent>
      </Card>
    </div>
  );
}
