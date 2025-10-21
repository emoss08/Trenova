/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ExternalLink } from "@/components/ui/link";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { api } from "@/services/api";
import { useContainerLogStore } from "@/stores/docker-store";
import { Container } from "@/types/docker";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Eye,
  LogOut,
  MoreVertical,
  Play,
  RotateCw,
  Square,
  Trash2,
} from "lucide-react";
import { useCallback } from "react";
import { toast } from "sonner";
import { Badge, BadgeProps } from "../ui/badge";

export function ContainerListTable({
  containers,
  isLoading,
}: {
  containers?: Container[];
  isLoading: boolean;
}) {
  const queryClient = useQueryClient();
  const startMutation = useMutation({
    mutationFn: api.docker.startContainer,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
      toast.success("Container started");
    },
    onError: (error: any) => {
      toast.error("Failed to start container", {
        description: error.response?.data?.message || error.message,
      });
    },
  });

  const stopMutation = useMutation({
    mutationFn: (id: string) => api.docker.stopContainer(id, 10),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
      toast.success("Container stopped");
    },
    onError: (error: any) => {
      toast.error("Failed to stop container", {
        description: error.response?.data?.message || error.message,
      });
    },
  });

  const restartMutation = useMutation({
    mutationFn: (id: string) => api.docker.restartContainer(id, 10),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
      toast.success("Container restarted");
    },
    onError: (error: any) => {
      toast.error("Failed to restart container", {
        description: error.response?.data?.message || error.message,
      });
    },
  });

  const removeMutation = useMutation({
    mutationFn: ({ id, force }: { id: string; force: boolean }) =>
      api.docker.removeContainer(id, force),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "containers"] });
      toast.success("Container removed");
    },
    onError: (error: any) => {
      toast.error("Failed to remove container", {
        description: error.response?.data?.message || error.message,
      });
    },
  });
  const getStatusBadge = (state: string) => {
    const statusMap: Record<
      string,
      { variant: BadgeProps["variant"]; label: string }
    > = {
      running: { variant: "info", label: "Running" },
      exited: { variant: "secondary", label: "Exited" },
      paused: { variant: "warning", label: "Paused" },
      restarting: { variant: "warning", label: "Restarting" },
      removing: { variant: "inactive", label: "Removing" },
      dead: { variant: "inactive", label: "Dead" },
      created: { variant: "active", label: "Created" },
    };

    const status = statusMap[state.toLowerCase()] || {
      variant: "default",
      label: state,
    };

    return (
      <Badge
        className="w-fit text-center"
        withDot={false}
        variant={status.variant}
      >
        {status.label}
      </Badge>
    );
  };

  const formatPorts = (ports: Container["Ports"]) => {
    if (!ports || ports.length === 0) return "-";

    return ports
      .map((p) => {
        if (p.PublicPort) {
          return `${p.PublicPort}:${p.PrivatePort}/${p.Type}`;
        }
        return `${p.PrivatePort}/${p.Type}`;
      })
      .join(", ");
  };

  const imageToDockerHub = useCallback((image: string): string | null => {
    // Handle empty or invalid input
    if (!image || typeof image !== "string") {
      return null;
    }

    // Split by first colon to separate tag
    const colonIndex = image.indexOf(":");
    let imageName: string;

    if (colonIndex === -1) {
      // No tag specified, use the full image name
      imageName = image;
    } else {
      imageName = image.substring(0, colonIndex);
    }

    // Handle official Docker images (no registry prefix)
    if (!imageName.includes("/")) {
      return `https://hub.docker.com/_/${imageName}`;
    }

    // Handle images with registry
    const parts = imageName.split("/");

    // If it looks like a private registry (contains dots or is localhost), skip
    if (parts[0].includes(".") || parts[0] === "localhost") {
      return null;
    }

    // Handle library images (docker.io/library/...)
    if (parts[0] === "docker.io" && parts[1] === "library") {
      const repoName = parts.slice(2).join("/");
      return `https://hub.docker.com/_/${repoName}`;
    }

    // Handle regular user/organization images
    if (parts.length >= 2) {
      const repoName = parts.join("/");
      return `https://hub.docker.com/r/${repoName}`;
    }

    return null;
  }, []);

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Image</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Ports</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {isLoading ? (
          <TableRow>
            <TableCell colSpan={6} className="text-center">
              Loading containers...
            </TableCell>
          </TableRow>
        ) : containers?.length === 0 ? (
          <TableRow>
            <TableCell colSpan={6} className="text-center">
              No containers found
            </TableCell>
          </TableRow>
        ) : (
          containers?.map((container) => (
            <TableRow key={container.Id}>
              <TableCell className="font-medium">
                {container.Names[0]?.replace("/", "")}
              </TableCell>
              <TableCell>
                <ExternalLink href={imageToDockerHub(container.Image) || ""}>
                  {container.Image}
                </ExternalLink>
              </TableCell>
              <TableCell>
                <div className="flex flex-col gap-1">
                  {getStatusBadge(container.State)}
                  <span className="text-xs text-muted-foreground">
                    {container.Status}
                  </span>
                </div>
              </TableCell>
              <TableCell className="text-xs">
                {formatPorts(container.Ports)}
              </TableCell>
              <TableCell>
                {new Date(container.Created * 1000).toLocaleDateString()}
              </TableCell>
              <TableCell className="text-right">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 w-8 p-0">
                      <span className="sr-only">Open menu</span>
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel>Actions</DropdownMenuLabel>
                    <DropdownMenuItem
                      title="View details"
                      startContent={<Eye className="mr-2 h-4 w-4" />}
                      onClick={() =>
                        useContainerLogStore.set("selectedContainer", container)
                      }
                    />
                    <DropdownMenuItem
                      title="View logs"
                      startContent={<LogOut className="mr-2 h-4 w-4" />}
                      onClick={() =>
                        useContainerLogStore.set("showLogs", container.Id)
                      }
                    />
                    <DropdownMenuSeparator />
                    {container.State.toLowerCase() === "running" ? (
                      <DropdownMenuItem
                        title="Stop container"
                        startContent={<Square className="mr-2 h-4 w-4" />}
                        onClick={() => stopMutation.mutate(container.Id)}
                      />
                    ) : (
                      <DropdownMenuItem
                        title="Start container"
                        startContent={<Play className="mr-2 h-4 w-4" />}
                        onClick={() => startMutation.mutate(container.Id)}
                      />
                    )}
                    <DropdownMenuItem
                      title="Restart container"
                      startContent={<RotateCw className="mr-2 h-4 w-4" />}
                      onClick={() => restartMutation.mutate(container.Id)}
                    />
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      className="text-destructive"
                      title="Remove container"
                      startContent={<Trash2 className="mr-2 h-4 w-4" />}
                      onClick={() =>
                        removeMutation.mutate({
                          id: container.Id,
                          force: false,
                        })
                      }
                    />
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  );
}
