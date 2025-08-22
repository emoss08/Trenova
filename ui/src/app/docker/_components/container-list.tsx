/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { dockerAPI } from "@/services/docker";
import { useQuery } from "@tanstack/react-query";

import { ContainerDetailsDialog } from "@/components/container-logs/container-details-dialog";
import { ContainerListTable } from "@/components/container-logs/container-table";
import { ContainerTableHeader } from "@/components/container-logs/container-table-header";
import { useContainerLogStore } from "@/stores/docker-store";
import { ContainerLogsDialog } from "./container-logs-dialog";

export function ContainerList() {
  const showAll = useContainerLogStore.get("showAll");
  const [selectedContainer, setSelectedContainer] =
    useContainerLogStore.use("selectedContainer");
  const [showLogs, setShowLogs] = useContainerLogStore.use("showLogs");

  const {
    data: containers,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["docker", "containers", showAll],
    queryFn: () => dockerAPI.listContainers(showAll),
    refetchInterval: 5000, // Refresh every 5 seconds
  });

  return (
    <div className="flex flex-col gap-2">
      <ContainerTableHeader refetch={refetch} />
      <ContainerListTable containers={containers} isLoading={isLoading} />

      {selectedContainer && (
        <ContainerDetailsDialog
          open={!!selectedContainer}
          onOpenChange={(open) => !open && setSelectedContainer(null)}
        />
      )}

      {showLogs && (
        <ContainerLogsDialog
          containerId={showLogs}
          open={!!showLogs}
          onOpenChange={(open) => !open && setShowLogs(null)}
        />
      )}
    </div>
  );
}
