/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatBytes } from "@/lib/utils";
import { api } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export function VolumeList() {
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [volumeName, setVolumeName] = useState("");
  const [driver, setDriver] = useState("local");
  const queryClient = useQueryClient();

  const {
    data: volumesData,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["docker", "volumes"],
    queryFn: api.docker.listVolumes,
  });

  const createMutation = useMutation({
    mutationFn: () => api.docker.createVolume(volumeName, driver),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "volumes"] });
      toast.success(`Successfully created volume ${volumeName}`);
      setCreateDialogOpen(false);
      setVolumeName("");
      setDriver("local");
    },
    onError: (error: any) => {
      toast.error("Failed to create volume", {
        description: error.response?.data?.message || error.message,
      });
    },
  });

  const removeMutation = useMutation({
    mutationFn: ({ id, force }: { id: string; force: boolean }) =>
      api.docker.removeVolume(id, force),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "volumes"] });
      toast.success("Volume removed");
    },
    onError: (error: any) => {
      toast.error("Failed to remove volume", {
        description: error.response?.data?.message || error.message,
      });
    },
  });

  const formatDate = (dateString?: string) => {
    if (!dateString) return "-";
    return new Date(dateString).toLocaleDateString();
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-end">
        <div className="flex items-center gap-2">
          <Button variant="default" onClick={() => setCreateDialogOpen(true)}>
            <Plus className="size-4" />
            Create Volume
          </Button>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="size-4" />
            Refresh
          </Button>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Driver</TableHead>
            <TableHead>Scope</TableHead>
            <TableHead>Size</TableHead>
            <TableHead>Created</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={6} className="text-center">
                Loading volumes...
              </TableCell>
            </TableRow>
          ) : volumesData?.volumes?.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="text-center">
                No volumes found
              </TableCell>
            </TableRow>
          ) : (
            volumesData?.volumes?.map((volume) => (
              <TableRow key={volume.Name}>
                <TableCell className="font-medium">
                  <div className="flex items-center gap-2">
                    {volume.Name.length > 30
                      ? `${volume.Name.slice(0, 30)}...`
                      : volume.Name}
                  </div>
                </TableCell>
                <TableCell>{volume.Driver}</TableCell>
                <TableCell>{volume.Scope}</TableCell>
                <TableCell>
                  {volume.size ? formatBytes(volume.size) : "-"}
                </TableCell>
                <TableCell>{formatDate(volume.CreatedAt)}</TableCell>
                <TableCell className="text-right">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() =>
                      removeMutation.mutate({
                        id: volume.Name,
                        force: false,
                      })
                    }
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
      {volumesData?.Warnings && volumesData.Warnings.length > 0 && (
        <div className="mt-4 p-3 bg-yellow-100 dark:bg-yellow-900/20 rounded-md">
          <p className="text-sm text-yellow-800 dark:text-yellow-200">
            Warnings: {volumesData.Warnings.join(", ")}
          </p>
        </div>
      )}

      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Docker Volume</DialogTitle>
            <DialogDescription>
              Create a new Docker volume for persistent data storage.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="volume-name">Volume Name</Label>
              <Input
                id="volume-name"
                placeholder="my-volume"
                value={volumeName}
                onChange={(e) => setVolumeName(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="driver">Driver</Label>
              <Select value={driver} onValueChange={setDriver}>
                <SelectTrigger id="driver">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="local">Local</SelectItem>
                  <SelectItem value="overlay">Overlay</SelectItem>
                  <SelectItem value="bridge">Bridge</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setCreateDialogOpen(false);
                setVolumeName("");
                setDriver("local");
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={!volumeName || createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create Volume"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
