/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { DockerNetwork } from "@/types/docker";
import { useCallback, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { queries } from "@/lib/queries";
import { api } from "@/services/api";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Check, Clipboard, Download, Trash } from "lucide-react";
import { toast } from "sonner";
import { NetworkInformation } from "./network-information";
import { NetworkOverview } from "./network-overview";

export function NetworkDetailsDialog({
  network,
  open,
  onOpenChange,
}: {
  network: DockerNetwork | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const copy = useCallback(async (text: string, key: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      setTimeout(() => setCopiedKey(null), 1200);
    } catch {
      /* noop */
    }
  }, []);

  const removeNetwork = useMutation({
    mutationFn: (id: string) => api.docker.removeNetwork(id),
    onSuccess: () => {
      toast.success("Network removed");
      onOpenChange(false);
      queryClient.invalidateQueries({
        queryKey: queries.docker.listNetworks._def,
      });
    },
    onError: (error) => {
      toast.error("Failed to remove network", {
        description: error.message,
      });
    },
  });

  const exportJSON = useCallback(() => {
    if (!network) return;
    const blob = new Blob([JSON.stringify(network, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `network-${network.Id.slice(0, 12)}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [network]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent withClose={false} className="max-w-3xl max-h-[85vh]">
        <DialogHeader className="sticky top-0 z-10 bg-gradient-to-b from-background to-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/70">
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogTitle className="flex items-center gap-2">
                Network Details
                {network && (
                  <Badge variant="outline" className="font-mono">
                    {network.Id.slice(0, 12)}
                  </Badge>
                )}
                {network?.Internal && (
                  <Badge variant="outline" className="text-[10px]">
                    Internal
                  </Badge>
                )}
                {network?.Attachable && (
                  <Badge variant="secondary" className="text-[10px]">
                    Attachable
                  </Badge>
                )}
              </DialogTitle>
              <DialogDescription className="truncate">
                {network?.Name} · {network?.Driver} · {network?.Scope}
              </DialogDescription>
            </div>

            <div className="flex items-center gap-2">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="red"
                    onClick={() => network && removeNetwork.mutate(network.Id)}
                    disabled={removeNetwork.isPending}
                  >
                    <Trash className="size-4" />
                    Remove
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Remove this network</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="outline" onClick={exportJSON}>
                    <Download className="size-4" />
                    Export JSON
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Download this network object</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    className="w-4"
                    variant="outline"
                    onClick={() => network && copy(network.Id, "id")}
                    aria-label="Copy network ID"
                  >
                    {copiedKey === "id" ? (
                      <Check className="size-4" />
                    ) : (
                      <Clipboard className="size-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Copy ID</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </DialogHeader>
        <ScrollArea className="h-[520px]">
          <div className="p-4">
            <NetworkOverview network={network} />
            <NetworkInformation
              network={network}
              copiedKey={copiedKey}
              copy={copy}
            />
          </div>
        </ScrollArea>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="secondary">Close</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type FactProps = {
  label: React.ReactNode;
  value: React.ReactNode;
  icon?: React.ReactNode;
  className?: string;
};

export function Fact({ label, value, icon, className }: FactProps) {
  return (
    <div className={`rounded-md border p-2 ${className ?? ""}`}>
      <div className="flex items-center gap-2 text-muted-foreground">
        {icon}
        <span>{label}</span>
      </div>
      <div className="mt-1 font-medium">{value ?? "—"}</div>
    </div>
  );
}
