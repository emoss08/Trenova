/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { NetworkDetailsDialog } from "@/components/network-logs/network-details-dialog";
import { Badge, BadgeProps } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { upperFirst } from "@/lib/utils";
import { api } from "@/services/api";
import { DockerNetwork } from "@/types/docker";
import { useQuery } from "@tanstack/react-query";
import { Info, RefreshCw } from "lucide-react";
import { useState } from "react";

export function NetworkList() {
  const [selectedNetwork, setSelectedNetwork] = useState<DockerNetwork | null>(
    null,
  );

  const {
    data: networks,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["docker", "networks"],
    queryFn: api.docker.listNetworks,
    refetchInterval: 10000,
  });

  const getNetworkBadge = (driver: string) => {
    const driverMap: Record<
      string,
      { variant: BadgeProps["variant"]; label: string }
    > = {
      bridge: { variant: "info", label: "Bridge" },
      host: { variant: "indigo", label: "Host" },
      overlay: { variant: "outline", label: "Overlay" },
      macvlan: { variant: "outline", label: "Macvlan" },
      null: { variant: "inactive", label: "None" },
    };

    const config = driverMap[driver] || {
      variant: "outline",
      label: driver,
    };

    return (
      <Badge withDot={false} variant={config.variant}>
        {config.label}
      </Badge>
    );
  };

  const getScopeBadge = (scope: string) => {
    return (
      <Badge withDot={false} variant={scope === "local" ? "info" : "secondary"}>
        {upperFirst(scope)}
      </Badge>
    );
  };

  const formatSubnet = (network: DockerNetwork) => {
    if (!network.IPAM?.Config || network.IPAM.Config.length === 0) {
      return "-";
    }
    const subnet = network.IPAM.Config[0].Subnet;
    return subnet || "-";
  };

  const countContainers = (network: DockerNetwork) => {
    if (!network.Containers) return 0;
    return Object.keys(network.Containers).length;
  };

  const mapBadgeToContainerCount = (count: number): BadgeProps["variant"] => {
    if (count === 0) return "inactive";
    if (count >= 1 && count <= 3) return "warning";
    if (count >= 4) return "active";
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-end">
        <Button variant="outline" onClick={() => refetch()}>
          <RefreshCw className="size-4" />
          Refresh
        </Button>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Driver</TableHead>
            <TableHead>Scope</TableHead>
            <TableHead>Subnet</TableHead>
            <TableHead>Containers</TableHead>
            <TableHead>Created</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={7} className="text-center">
                Loading networks...
              </TableCell>
            </TableRow>
          ) : networks?.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="text-center">
                No networks found
              </TableCell>
            </TableRow>
          ) : (
            networks?.map((network) => (
              <TableRow key={network.Id}>
                <TableCell className="font-medium">
                  <div className="flex items-center gap-2">
                    {network.Name}
                    {network.Internal && (
                      <Badge variant="outline" className="text-xs">
                        Internal
                      </Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell>{getNetworkBadge(network.Driver)}</TableCell>
                <TableCell>{getScopeBadge(network.Scope)}</TableCell>
                <TableCell className="font-mono text-xs">
                  {formatSubnet(network)}
                </TableCell>
                <TableCell>
                  <Badge
                    withDot={false}
                    variant={mapBadgeToContainerCount(countContainers(network))}
                  >
                    {countContainers(network)}
                  </Badge>
                </TableCell>
                <TableCell>
                  {new Date(network.Created).toLocaleDateString()}
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setSelectedNetwork(network)}
                  >
                    <Info className="h-4 w-4" />
                  </Button>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <NetworkDetailsDialog
        network={selectedNetwork}
        open={!!selectedNetwork}
        onOpenChange={(open) => !open && setSelectedNetwork(null)}
      />
    </div>
  );
}
