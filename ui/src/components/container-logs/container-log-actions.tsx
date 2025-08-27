/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { cn } from "@/lib/utils";
import { useContainerLogStore } from "@/stores/docker-store";
import { Check, Clipboard, Download, RefreshCw } from "lucide-react";
import { useCallback } from "react";

type ContainerLogActionsProps = {
  logs?: string[];
  filteredLogs: string[];
  needle: string;
  containerId: string;
  refetch: () => void;
  isRefetching: boolean;
};

export function ContainerLogActions({
  logs,
  filteredLogs,
  needle,
  containerId,
  refetch,
  isRefetching,
}: ContainerLogActionsProps) {
  const [copied, setCopied] = useContainerLogStore.use("copied");

  const handleRefresh = useCallback(() => {
    void refetch();
  }, [refetch]);

  const handleDownload = useCallback(
    (which: "all" | "filtered") => {
      const lines = which === "filtered" && needle ? filteredLogs : logs;
      if (!lines?.length) return;
      const blob = new Blob([lines.join("\n")], { type: "text/plain" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `container-${containerId.slice(0, 12)}-logs${
        which === "filtered" ? "-filtered" : ""
      }.txt`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    },
    [logs, filteredLogs, needle, containerId],
  );

  const handleCopyAll = useCallback(
    async (which: "all" | "filtered") => {
      const lines = which === "filtered" && needle ? filteredLogs : logs;
      if (!lines?.length) return;
      await navigator.clipboard.writeText(lines.join("\n"));
      setCopied(true);
      const t = setTimeout(() => setCopied(false), 1200);
      return () => clearTimeout(t);
    },
    [logs, filteredLogs, needle, setCopied],
  );

  return (
    <div className="flex items-center gap-2">
      <TooltipProvider delayDuration={200}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              size="icon"
              variant="outline"
              onClick={handleRefresh}
              aria-label="Refresh logs"
            >
              <RefreshCw
                className={cn("size-4", isRefetching ? "animate-spin" : "")}
              />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Refresh now</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              size="icon"
              variant="outline"
              onClick={() => handleDownload(needle ? "filtered" : "all")}
              aria-label="Download logs"
            >
              <Download className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            Download {needle ? "filtered" : "all"} as .txt
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              size="icon"
              variant="outline"
              onClick={() => void handleCopyAll(needle ? "filtered" : "all")}
              aria-label="Copy logs"
            >
              {copied ? (
                <Check className="h-4 w-4" />
              ) : (
                <Clipboard className="size-4" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {copied ? "Copied!" : `Copy ${needle ? "filtered" : "all"}`}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}
