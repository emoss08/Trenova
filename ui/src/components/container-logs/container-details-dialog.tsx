/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { formatBytes } from "@/lib/utils";
import { api } from "@/services/api";
import { dockerAPI } from "@/services/docker";
import { useContainerLogStore } from "@/stores/docker-store";
import { ContainerStats } from "@/types/docker";
import { Activity, Download, Gauge, HardDrive, Network } from "lucide-react";
import {
  CommandDetails,
  ContainerLabels,
  CopyIcon,
  EnvironmentVariables,
} from "./container-detail-components";

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

  const memPct = clamp(stats?.memPercent ?? 0, 0, 100);
  const cpuPct = clamp(stats?.cpuPercent ?? 0, 0, 100);

  const ioRates = useMemo(() => {
    const prev = prevStatsRef.current;
    if (!stats || !prev) return null;
    const dt = Math.max(
      1,
      (new Date(stats.timestamp).getTime() -
        new Date(prev.timestamp).getTime()) /
        1000,
    );
    const rate = (a: number, b: number) => (a - b) / dt;
    return {
      netInPerSec: Math.max(0, rate(stats.netInput, prev.netInput)),
      netOutPerSec: Math.max(0, rate(stats.netOutput, prev.netOutput)),
      blkInPerSec: Math.max(0, rate(stats.blockInput, prev.blockInput)),
      blkOutPerSec: Math.max(0, rate(stats.blockOutput, prev.blockOutput)),
    };
  }, [stats]);

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

  const createdAt = useMemo(() => {
    try {
      return new Date(selectedContainer?.Created ?? 0 * 1000).toLocaleString();
    } catch {
      return "—";
    }
  }, [selectedContainer?.Created]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent withClose={false} className="max-w-4xl max-h-[85vh]">
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

          <ScrollArea className="h-[520px] px-4">
            <TabsContent value="overview" className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm">
                      Container Information
                    </CardTitle>
                    <CardDescription>Core identity & lifecycle</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-2 text-sm">
                    <KV label="ID">
                      <Mono>{selectedContainer?.Id.slice(0, 12)}</Mono>
                    </KV>
                    <KV label="Name">
                      {selectedContainer?.Names?.[0]?.replace("/", "") || "—"}
                    </KV>
                    <KV label="Image">{selectedContainer?.Image || "—"}</KV>
                    <KV label="Status">
                      <Badge
                        variant={isRunning ? ("active" as any) : "secondary"}
                        className="capitalize"
                        title={selectedContainer?.Status}
                      >
                        {selectedContainer?.State}
                      </Badge>
                    </KV>
                    <KV label="Created">{createdAt}</KV>
                    {details?.Config?.Hostname && (
                      <KV label="Hostname">
                        <Mono>{details.Config.Hostname}</Mono>
                      </KV>
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm">
                      Resource Footprint
                    </CardTitle>
                    <CardDescription>
                      On-disk sizes (approximate)
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-2 text-sm">
                    <KV label="RW Size">
                      {formatBytes(selectedContainer?.SizeRw || 0)}
                    </KV>
                    <KV label="RootFS Size">
                      {formatBytes(selectedContainer?.SizeRootFs || 0)}
                    </KV>
                    {details?.GraphDriver?.Name && (
                      <KV label="Driver">{details.GraphDriver.Name}</KV>
                    )}
                  </CardContent>
                </Card>
              </div>

              {selectedContainer?.Mounts?.length ? (
                <div className="bg-muted p-4 rounded-md border">
                  <div className="pb-2">
                    <h3 className="text-sm">Mounts</h3>
                    <p className="text-xs text-muted-foreground">
                      Volumes and bind mounts
                    </p>
                  </div>
                  <div className="flex flex-col gap-2 text-sm">
                    {selectedContainer?.Mounts.map((m, idx) => (
                      <div key={idx}>
                        <KV label="Type">{m.Type}</KV>
                        <KV label="Source">
                          <Mono>{m.Source}</Mono>
                          <CopyIcon
                            ariaLabel="Copy source"
                            onClick={() =>
                              handleCopy(m.Source, `mount-src-${idx}`)
                            }
                            active={copiedKey === `mount-src-${idx}`}
                          />
                        </KV>
                        <KV label="Destination">
                          <Mono>{m.Destination}</Mono>
                          <CopyIcon
                            ariaLabel="Copy destination"
                            onClick={() =>
                              handleCopy(m.Destination, `mount-dst-${idx}`)
                            }
                            active={copiedKey === `mount-dst-${idx}`}
                          />
                        </KV>
                        {m.RW !== undefined && (
                          <KV label="RW">{String(m.RW)}</KV>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
            </TabsContent>

            {/* STATS */}
            <TabsContent value="stats" className="space-y-4">
              {isRunning ? (
                stats ? (
                  <div className="space-y-4">
                    <div className="grid gap-4 md:grid-cols-2">
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="flex items-center gap-2 text-sm">
                            <Gauge className="h-4 w-4" /> CPU Usage
                          </CardTitle>
                          <CardDescription>
                            {cpuPct.toFixed(2)}%
                          </CardDescription>
                        </CardHeader>
                        <CardContent>
                          <Progress value={cpuPct} className="h-2" />
                          <div className="mt-3">
                            <Sparkline
                              data={cpuHistory}
                              ariaLabel="CPU trend (last 60s)"
                            />
                          </div>
                        </CardContent>
                      </Card>

                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="flex items-center gap-2 text-sm">
                            <HardDrive className="h-4 w-4" /> Memory Usage
                          </CardTitle>
                          <CardDescription>
                            {formatBytes(stats.memUsage)} /{" "}
                            {formatBytes(stats.memLimit)}
                          </CardDescription>
                        </CardHeader>
                        <CardContent>
                          <Progress value={memPct} className="h-2" />
                          <div className="mt-3">
                            <Sparkline
                              data={memHistory}
                              ariaLabel="Memory trend (last 60s)"
                            />
                          </div>
                        </CardContent>
                      </Card>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="flex items-center gap-2 text-sm">
                            <Network className="h-4 w-4" /> Network I/O
                          </CardTitle>
                          <CardDescription>
                            In {formatBytes(stats.netInput)} · Out{" "}
                            {formatBytes(stats.netOutput)}
                          </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-2 text-sm">
                          <KV label="In/sec">
                            {ioRates
                              ? `${formatBytes(ioRates.netInPerSec)}/s`
                              : "—"}
                          </KV>
                          <KV label="Out/sec">
                            {ioRates
                              ? `${formatBytes(ioRates.netOutPerSec)}/s`
                              : "—"}
                          </KV>
                        </CardContent>
                      </Card>

                      <Card>
                        <CardHeader className="pb-2">
                          <CardTitle className="flex items-center gap-2 text-sm">
                            <HardDrive className="h-4 w-4" /> Block I/O
                          </CardTitle>
                          <CardDescription>
                            Read {formatBytes(stats.blockInput)} · Write{" "}
                            {formatBytes(stats.blockOutput)}
                          </CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-2 text-sm">
                          <KV label="Read/sec">
                            {ioRates
                              ? `${formatBytes(ioRates.blkInPerSec)}/s`
                              : "—"}
                          </KV>
                          <KV label="Write/sec">
                            {ioRates
                              ? `${formatBytes(ioRates.blkOutPerSec)}/s`
                              : "—"}
                          </KV>
                        </CardContent>
                      </Card>
                    </div>

                    <div className="text-xs text-muted-foreground text-right">
                      Updated {new Date(stats.timestamp).toLocaleTimeString()}
                      {live && (
                        <span className="ml-2 align-middle inline-flex items-center gap-1">
                          <span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" />
                          live
                        </span>
                      )}
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <SkeletonRow />
                    <SkeletonRow />
                    <div className="text-xs text-muted-foreground text-center">
                      Waiting for stats…
                    </div>
                  </div>
                )
              ) : (
                <div className="text-center text-muted-foreground py-8">
                  Container must be running to view stats
                </div>
              )}
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
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">Port Mappings</CardTitle>
                  <CardDescription>Published container ports</CardDescription>
                </CardHeader>
                <CardContent>
                  {selectedContainer?.Ports?.length ? (
                    <div className="space-y-2">
                      {selectedContainer?.Ports.map((p, idx) => (
                        <div
                          key={idx}
                          className="flex items-center justify-between text-sm rounded-md border p-2"
                        >
                          <span>
                            {p.PrivatePort}/{p.Type}
                          </span>
                          {p.PublicPort ? (
                            <div className="flex items-center gap-2">
                              <span className="text-muted-foreground">→</span>
                              <Mono>{`${p.IP || "0.0.0.0"}:${p.PublicPort}`}</Mono>
                              <CopyIcon
                                ariaLabel="Copy mapping"
                                onClick={() =>
                                  handleCopy(
                                    `${p.IP || "0.0.0.0"}:${p.PublicPort}`,
                                    `port-${idx}`,
                                  )
                                }
                                active={copiedKey === `port-${idx}`}
                              />
                            </div>
                          ) : (
                            <span className="text-muted-foreground">
                              not published
                            </span>
                          )}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      No port mappings
                    </p>
                  )}
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">Networks</CardTitle>
                  <CardDescription>Connected Docker networks</CardDescription>
                </CardHeader>
                <CardContent className="space-y-2">
                  {Object.entries(
                    selectedContainer?.NetworkSettings?.Networks || {},
                  ).map(([name, network]: [string, any]) => (
                    <div
                      key={name}
                      className="text-sm rounded-md border p-2 space-y-1"
                    >
                      <div className="font-semibold">{name}</div>
                      {network.IPAddress && (
                        <KV label="IP Address">
                          <Mono>{network.IPAddress}</Mono>
                          <CopyIcon
                            ariaLabel="Copy IP"
                            onClick={() =>
                              handleCopy(network.IPAddress, `ip-${name}`)
                            }
                            active={copiedKey === `ip-${name}`}
                          />
                        </KV>
                      )}
                      {network.MacAddress && (
                        <KV label="MAC">
                          <Mono>{network.MacAddress}</Mono>
                        </KV>
                      )}
                    </div>
                  ))}
                </CardContent>
              </Card>
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

/* ---------- Small utilities/components (local) ---------- */

function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}

const Mono = ({ children }: { children: React.ReactNode }) => (
  <span className="font-mono text-xs">{children}</span>
);

const KV = ({
  label,
  children,
}: {
  label: string;
  children?: React.ReactNode;
}) => (
  <div className="flex items-start justify-between gap-3 py-1">
    <span className="text-muted-foreground">{label}:</span>
    <div className="min-w-0 text-right">{children ?? "—"}</div>
  </div>
);

const Sparkline = ({
  data,
  width,
  height,
  ariaLabel,
}: {
  data: number[];
  width?: number;
  height?: number;
  ariaLabel?: string;
}) => {
  const pad = 2;
  if (!data?.length) {
    return <div className="text-xs text-muted-foreground">No data</div>;
  }
  const max = Math.max(...data, 1);
  const min = Math.min(...data, 0);
  const range = Math.max(1, max - min);
  const step = ((width ?? 320) - pad * 2) / Math.max(1, data.length - 1);
  const points = data
    .map((d, i) => {
      const x = pad + i * step;
      const y = pad + (1 - (d - min) / range) * ((height ?? 48) - pad * 2);
      return `${x},${y}`;
    })
    .join(" ");

  return (
    <svg
      width={width}
      height={height}
      role="img"
      aria-label={ariaLabel}
      className="block text-primary/80"
    >
      <polyline
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        points={points}
      />
    </svg>
  );
};

const SkeletonRow = () => (
  <div className="grid gap-4 md:grid-cols-2">
    <div className="rounded-md border p-4">
      <div className="h-4 w-24 rounded bg-muted animate-pulse mb-3" />
      <div className="h-2 w-full rounded bg-muted animate-pulse" />
      <div className="h-8 w-full rounded bg-muted animate-pulse mt-3" />
    </div>
    <div className="rounded-md border p-4">
      <div className="h-4 w-24 rounded bg-muted animate-pulse mb-3" />
      <div className="h-2 w-full rounded bg-muted animate-pulse" />
      <div className="h-8 w-full rounded bg-muted animate-pulse mt-3" />
    </div>
  </div>
);
