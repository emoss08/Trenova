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
import { Download, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

export function ImageList() {
  const [pullDialogOpen, setPullDialogOpen] = useState(false);
  const [imageName, setImageName] = useState("");
  const queryClient = useQueryClient();

  const {
    data: images,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["docker", "images"],
    queryFn: api.docker.listImages,
  });

  const pullMutation = useMutation({
    mutationFn: api.docker.pullImage,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "images"] });
      toast.success(`Successfully pulled ${imageName}`);
      setPullDialogOpen(false);
      setImageName("");
    },
    onError: (error: any) => {
      toast.error("Failed to pull image", {
        description: error.response?.data?.message || error.message,
      });
    },
  });

  const removeMutation = useMutation({
    mutationFn: ({ id, force }: { id: string; force: boolean }) =>
      api.docker.removeImage(id, force),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["docker", "images"] });
      toast.success("Image removed");
    },
    onError: (error: any) => {
      toast.error("Failed to remove image", {
        description: error.response?.data?.message || error.message,
      });
    },
  });

  const formatRepoTag = (tags: string[]) => {
    if (!tags || tags.length === 0 || tags[0] === "<none>:<none>") {
      return <span className="text-muted-foreground">{"<none>"}</span>;
    }
    return tags[0];
  };

  const formatCreated = (timestamp: number) => {
    const date = new Date(timestamp * 1000);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) return "Today";
    if (days === 1) return "Yesterday";
    if (days < 7) return `${days} days ago`;
    if (days < 30) return `${Math.floor(days / 7)} weeks ago`;
    if (days < 365) return `${Math.floor(days / 30)} months ago`;
    return `${Math.floor(days / 365)} years ago`;
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-end">
        <div className="flex items-center gap-2">
          <Button variant="default" onClick={() => setPullDialogOpen(true)}>
            <Download className="size-4" />
            Pull Image
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
            <TableHead>Repository:Tag</TableHead>
            <TableHead>Image ID</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Size</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={5} className="text-center">
                Loading images...
              </TableCell>
            </TableRow>
          ) : images?.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="text-center">
                No images found
              </TableCell>
            </TableRow>
          ) : (
            images?.map((image) => (
              <TableRow key={image.Id}>
                <TableCell className="font-medium">
                  {formatRepoTag(image.RepoTags)}
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {image.Id.replace("sha256:", "").slice(0, 12)}
                </TableCell>
                <TableCell>{formatCreated(image.Created)}</TableCell>
                <TableCell>{formatBytes(image.Size)}</TableCell>
                <TableCell className="text-right">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() =>
                      removeMutation.mutate({ id: image.Id, force: false })
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

      <Dialog open={pullDialogOpen} onOpenChange={setPullDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Pull Docker Image</DialogTitle>
            <DialogDescription>
              Enter the name of the image to pull from Docker Hub or another
              registry.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="image-name">Image Name</Label>
              <Input
                id="image-name"
                placeholder="e.g., nginx:latest, redis:alpine"
                value={imageName}
                onChange={(e) => setImageName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && imageName) {
                    pullMutation.mutate(imageName);
                  }
                }}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setPullDialogOpen(false);
                setImageName("");
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={() => pullMutation.mutate(imageName)}
              disabled={!imageName || pullMutation.isPending}
            >
              {pullMutation.isPending ? "Pulling..." : "Pull Image"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
