/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { clamp } from "@/lib/utils";
import { api } from "@/services/api";
import { dockerAPI } from "@/services/docker";
import { useContainerLogStore } from "@/stores/docker-store";
import { ContainerStats } from "@/types/docker";
import { Activity, Download } from "lucide-react";

import { lazy, Suspense } from "react";
import { OverviewTabContent } from "./contailer-overview";
import {
  CommandDetails,
  ContainerLabels,
  EnvironmentVariables,
} from "./container-detail-components";
import { NetworkTabContent } from "./container-network";

const StatsTabContent = lazy(() =>
  import("./container-stats").then((module) => ({
    default: module.StatsTabContent,
  })),
);

interface ContainerDetailsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ContainerDetailsDialog({
  open,
  onOpenChange,
}: ContainerDetailsDialogProps) {
  const selectedContainer = useContainerLogStore.get("selectedContainer");
  const [stats, setStats] = useState<ContainerStats | null>(null);
  const [live, setLive] = useState(true);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const [cpuHistory, setCpuHistory] = useState<number[]>([]);
  const [memHistory, setMemHistory] = useState<number[]>([]);
  const historyCap = 60;

  const prevStatsRef = useRef<ContainerStats | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const isRunning = selectedContainer?.State?.toLowerCase() === "running";

  // Clean up on dialog close
  useEffect(() => {
    if (!open) {
      // Reset state when dialog closes
      setStats(null);
      setCpuHistory([]);
      setMemHistory([]);
      prevStatsRef.current = null;
      setLive(true); // Reset live mode for next opening

      // Ensure EventSource is closed
      if (eventSourceRef.current) {
        console.log("Closing stats stream on dialog close");
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    }
  }, [open]);

  const { data: details, refetch: refetchDetails } = useQuery({
    queryKey: ["docker", "container", selectedContainer?.Id],
    queryFn: () => api.docker.inspectContainer(selectedContainer?.Id ?? ""),
    enabled: open,
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  });

  useEffect(() => {
    // Clean up any existing connection
    if (eventSourceRef.current) {
      console.log("Closing existing stats stream");
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }

    if (open && isRunning && live && selectedContainer?.Id) {
      console.log("Starting stats stream for container:", selectedContainer.Id);
      eventSourceRef.current = dockerAPI.streamContainerStats(
        selectedContainer.Id,
        (newStats: ContainerStats) => {
          setStats(newStats);
          setCpuHistory((h) => {
            const next = [...h, clamp(newStats.cpuPercent, 0, 100)];
            return next.length > historyCap ? next.slice(-historyCap) : next;
          });
          const memPct = clamp(newStats.memPercent, 0, 100);
          setMemHistory((h) => {
            const next = [...h, memPct];
            return next.length > historyCap ? next.slice(-historyCap) : next;
          });
          prevStatsRef.current = newStats;
        },
        (error: string) => {
          console.error("Stats stream error:", error);
          setLive(false); // Disable live mode on error
        },
        () => {
          console.log("Connected to stats stream");
        },
      );
    }

    return () => {
      if (eventSourceRef.current) {
        console.log("Closing stats stream in cleanup");
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [open, isRunning, live, selectedContainer?.Id]);

  const handleCopy = useCallback(async (text: string, key: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      const t = setTimeout(() => setCopiedKey(null), 1200);
      return () => clearTimeout(t);
    } catch {
      // noop - if copy fails, do nothing
    }
  }, []);

  const exportInspect = useCallback(() => {
    const data = details ?? {};
    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `container-${selectedContainer?.Id.slice(0, 12)}-inspect.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [details, selectedContainer?.Id]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent withClose={false} className="min-w-[800px]">
        <DialogHeader>
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogTitle className="flex items-center gap-2">
                Container Details
                <Badge withDot={false} variant="purple">
                  {selectedContainer?.Id.slice(0, 12)}
                </Badge>
              </DialogTitle>
              <DialogDescription className="truncate">
                {selectedContainer?.Names?.[0]?.replace("/", "")} ·{" "}
                {selectedContainer?.Image}
              </DialogDescription>
            </div>
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-2">
                <Switch
                  id="live"
                  size="sm"
                  checked={live}
                  onCheckedChange={(v) => setLive(v)}
                  disabled={!isRunning}
                />
                <Label htmlFor="live">Live stats</Label>
              </div>
              <Separator orientation="vertical" className="h-6" />
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="outline" onClick={() => refetchDetails()}>
                    <Activity className="size-4" />
                    Refresh inspect
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Refetch the latest inspect data</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="outline" onClick={exportInspect}>
                    <Download className="size-4" />
                    Export JSON
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Download docker inspect as JSON</TooltipContent>
              </Tooltip>
            </div>
          </div>
        </DialogHeader>

        <Tabs defaultValue="overview" className="mt-2">
          <TabsList className="before:bg-border relative h-auto w-full gap-0.5 bg-transparent p-0 before:absolute before:inset-x-0 before:bottom-0 before:h-px">
            <TabsTrigger
              value="overview"
              className="bg-muted overflow-hidden rounded-b-none border-x border-t py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
            >
              Overview
            </TabsTrigger>
            <TabsTrigger
              value="stats"
              className="bg-muted overflow-hidden rounded-b-none border-x border-t py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
            >
              Stats
            </TabsTrigger>
            <TabsTrigger
              value="config"
              className="bg-muted overflow-hidden rounded-b-none border-x border-t py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
            >
              Configuration
            </TabsTrigger>
            <TabsTrigger
              value="network"
              className="bg-muted overflow-hidden rounded-b-none border-x border-t py-2 data-[state=active]:z-10 data-[state=active]:shadow-none"
            >
              Network
            </TabsTrigger>
          </TabsList>

          <ScrollArea className="h-[560px] px-4">
            <TabsContent value="overview" className="space-y-4">
              <OverviewTabContent
                details={details}
                handleCopy={handleCopy}
                copiedKey={copiedKey}
              />
            </TabsContent>

            <TabsContent value="stats" className="space-y-4">
              <Suspense
                fallback={
                  <div className="flex items-center justify-center h-[400px]">
                    <div className="animate-pulse text-muted-foreground">
                      Loading stats...
                    </div>
                  </div>
                }
              >
                <StatsTabContent
                  stats={stats}
                  cpuHistory={cpuHistory}
                  memHistory={memHistory}
                  prevStatsRef={prevStatsRef}
                  live={live}
                />
              </Suspense>
            </TabsContent>

            {/* CONFIG */}
            <TabsContent value="config" className="space-y-4">
              <EnvironmentVariables
                details={details}
                handleCopy={handleCopy}
                copiedKey={copiedKey}
              />

              <CommandDetails
                details={details}
                handleCopy={handleCopy}
                copiedKey={copiedKey}
              />

              <ContainerLabels handleCopy={handleCopy} copiedKey={copiedKey} />
            </TabsContent>

            {/* NETWORK */}
            <TabsContent value="network" className="space-y-4">
              <NetworkTabContent
                handleCopy={handleCopy}
                copiedKey={copiedKey}
              />
            </TabsContent>
          </ScrollArea>
        </Tabs>

        <DialogFooter className="mt-1">
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
