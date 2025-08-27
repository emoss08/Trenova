import { formatBytes, truncateText } from "@/lib/utils";
import { useContainerLogStore } from "@/stores/docker-store";
import { ContainerInspect } from "@/types/docker";
import { Box, Folder, HardDrive, Network, Tag } from "lucide-react";
import { useMemo } from "react";
import { Badge } from "../ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../ui/card";
import { ScrollArea, ScrollAreaShadow } from "../ui/scroll-area";
import { Separator } from "../ui/separator";
import { CopyIcon, KV, Mono } from "./container-detail-components";

/**
 * OverviewTabContent (modernized)
 * - Subtle gradient cards
 * - Better grouping & readability
 * - Copy actions for key fields
 * - Robust date handling (Docker seconds vs ISO)
 * - Reactive store subscription (no `.get()`)
 * - Optional sections: Ports, Mounts, Networks, Labels
 */

export function OverviewTabContent({
  details,
  copiedKey,
  handleCopy,
}: {
  details?: ContainerInspect;
  copiedKey: string | null;
  handleCopy: (text: string, key: string) => void;
}) {
  // Subscribe so changes re-render consistently
  const selectedContainer = useContainerLogStore.get("selectedContainer");
  const state = selectedContainer?.State?.toLowerCase();

  const createdAt = useMemo(
    () => formatDockerDate(selectedContainer?.Created),
    [selectedContainer?.Created],
  );

  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        {/* Container Information */}
        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Box className="h-4 w-4" /> Container Information
            </CardTitle>
            <CardDescription>Core identity & lifecycle</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <KV label="ID">
              <Mono>{shortId(selectedContainer?.Id)}</Mono>
              {selectedContainer?.Id && (
                <CopyIcon
                  ariaLabel="Copy container ID"
                  onClick={() =>
                    handleCopy(String(selectedContainer.Id), "container-id")
                  }
                  active={copiedKey === "container-id"}
                />
              )}
            </KV>
            <KV label="Name">
              {selectedContainer?.Names?.[0]?.replace("/", "") || "—"}
              {selectedContainer?.Names?.[0] && (
                <CopyIcon
                  ariaLabel="Copy name"
                  onClick={() =>
                    handleCopy(
                      selectedContainer.Names[0].replace("/", ""),
                      "container-name",
                    )
                  }
                  active={copiedKey === "container-name"}
                />
              )}
            </KV>
            <KV label="Image">
              {selectedContainer?.Image || "—"}
              {selectedContainer?.Image && (
                <CopyIcon
                  ariaLabel="Copy image"
                  onClick={() =>
                    handleCopy(
                      String(selectedContainer.Image),
                      "container-image",
                    )
                  }
                  active={copiedKey === "container-image"}
                />
              )}
            </KV>
            <KV label="Status">
              <StatusBadge state={state} title={selectedContainer?.Status} />
            </KV>
            <KV label="Created">{createdAt}</KV>
            {details?.Config?.Hostname && (
              <KV label="Hostname">
                <Mono>{details.Config.Hostname}</Mono>
                <CopyIcon
                  ariaLabel="Copy hostname"
                  onClick={() =>
                    handleCopy(details.Config.Hostname!, "hostname")
                  }
                  active={copiedKey === "hostname"}
                />
              </KV>
            )}
          </CardContent>
        </Card>
        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <HardDrive className="h-4 w-4" /> Resource Footprint
            </CardTitle>
            <CardDescription>On-disk sizes (approximate)</CardDescription>
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
            {details?.HostConfig?.RestartPolicy?.Name && (
              <KV label="Restart Policy">
                {details.HostConfig.RestartPolicy.Name}
                {details.HostConfig.RestartPolicy.MaximumRetryCount
                  ? ` (x${details.HostConfig.RestartPolicy.MaximumRetryCount})`
                  : null}
              </KV>
            )}
          </CardContent>
        </Card>
      </div>
      {Array.isArray((selectedContainer as any)?.Ports) &&
      (selectedContainer as any).Ports.length > 0 ? (
        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Network className="h-4 w-4" /> Ports
            </CardTitle>
            <CardDescription>Published & exposed ports</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {(selectedContainer as any).Ports.map((p: any, idx: number) => (
                <PortPill key={idx} p={p} />
              ))}
            </div>
          </CardContent>
        </Card>
      ) : null}
      {selectedContainer?.Mounts?.length ? (
        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Folder className="h-4 w-4" /> Mounts
            </CardTitle>
            <CardDescription>Volumes and bind mounts</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <ScrollArea className="h-72 px-4 pb-2">
              <div className="flex flex-col gap-3 text-sm">
                {selectedContainer.Mounts.map((m, idx) => (
                  <div key={idx} className="rounded-md border p-3 bg-card/50">
                    <div className="mb-2 flex items-center gap-2">
                      <Badge
                        withDot={false}
                        variant="outline"
                        className="capitalize"
                      >
                        {m.Type}
                      </Badge>
                      {m.RW !== undefined && (
                        <Badge
                          withDot={false}
                          variant={m.RW ? ("active" as any) : "secondary"}
                          className="capitalize"
                        >
                          {m.RW ? "read-write" : "read-only"}
                        </Badge>
                      )}
                    </div>
                    <KV label="Source">
                      <Mono>{m.Source}</Mono>
                      <CopyIcon
                        ariaLabel="Copy source"
                        onClick={() => handleCopy(m.Source, `mount-src-${idx}`)}
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
                    {m.Mode && <KV label="Mode">{m.Mode}</KV>}
                    {m.Name && <KV label="Name">{m.Name}</KV>}
                  </div>
                ))}
              </div>
              <ScrollAreaShadow />
            </ScrollArea>
          </CardContent>
        </Card>
      ) : null}
      {details?.NetworkSettings?.Networks &&
      Object.keys(details.NetworkSettings.Networks).length ? (
        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Network className="h-4 w-4" /> Networks
            </CardTitle>
            <CardDescription>Connected Docker networks</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {Object.entries(details.NetworkSettings.Networks).map(
                ([name, net]: any) => (
                  <div
                    key={name}
                    className="rounded-md border bg-card/50 p-3 text-sm"
                  >
                    <div className="mb-1 flex items-center gap-2">
                      <Badge
                        withDot={false}
                        variant="outline"
                        className="capitalize"
                      >
                        {name}
                      </Badge>
                    </div>
                    {net.IPAddress && (
                      <KV label="IP">
                        <Mono>{net.IPAddress}</Mono>
                        <CopyIcon
                          ariaLabel="Copy IP"
                          onClick={() =>
                            handleCopy(net.IPAddress, `net-ip-${name}`)
                          }
                          active={copiedKey === `net-ip-${name}`}
                        />
                      </KV>
                    )}
                    {net.Gateway && <KV label="Gateway">{net.Gateway}</KV>}
                    {net.MacAddress && (
                      <KV label="MAC">
                        <Mono>{net.MacAddress}</Mono>
                      </KV>
                    )}
                  </div>
                ),
              )}
            </div>
          </CardContent>
        </Card>
      ) : null}
      {details?.Config?.Labels && Object.keys(details.Config.Labels).length ? (
        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Tag className="h-4 w-4" /> Labels
            </CardTitle>
            <CardDescription>Metadata labels</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <ScrollArea className="h-72 px-4 pb-2">
              <div className="space-y-2 text-sm">
                {Object.entries(details.Config.Labels).map(([k, v]) => (
                  <div key={k} className="rounded-md border p-2 bg-card/50">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-xs text-muted-foreground">{k}</span>
                      <Separator orientation="vertical" className="h-4" />
                      <Mono className="truncate">
                        {truncateText(String(v), 80)}
                      </Mono>
                      <CopyIcon
                        ariaLabel="Copy label value"
                        onClick={() => handleCopy(String(v), `label-${k}`)}
                        active={copiedKey === `label-${k}`}
                      />
                    </div>
                  </div>
                ))}
              </div>
              <ScrollAreaShadow />
            </ScrollArea>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

function StatusBadge({ state, title }: { state?: string; title?: string }) {
  const variant: any =
    state === "running"
      ? "active"
      : state === "dead"
        ? "destructive"
        : "secondary";
  return (
    <Badge
      withDot={false}
      variant={variant}
      className="capitalize"
      title={title}
    >
      <span className="mr-1">
        <StatusDot state={state} />
      </span>
      {state || "unknown"}
    </Badge>
  );
}

function StatusDot({ state }: { state?: string }) {
  const color =
    state === "running"
      ? "bg-emerald-500"
      : state === "paused"
        ? "bg-amber-500"
        : state === "restarting"
          ? "bg-sky-500"
          : state === "dead" || state === "exited"
            ? "bg-rose-500"
            : "bg-muted-foreground";
  return <span className={`inline-block h-1.5 w-1.5 rounded-full ${color}`} />;
}

function PortPill({ p }: { p: any }) {
  const text = [
    p.PrivatePort ? `${p.PrivatePort}/${p.Type || "tcp"}` : null,
    "→",
    p.IP && p.PublicPort
      ? `${p.IP}:${p.PublicPort}`
      : p.PublicPort
        ? `${p.PublicPort}`
        : "—",
  ]
    .filter(Boolean)
    .join(" ");
  return (
    <Badge withDot={false} variant="outline" className="font-normal">
      {text}
    </Badge>
  );
}

function shortId(id?: string) {
  return id ? id.slice(0, 12) : "—";
}

function formatDockerDate(input?: number | string) {
  if (!input && input !== 0) return "—";
  let d: Date;
  if (typeof input === "number") {
    // Docker sometimes returns seconds since epoch for lists
    d = new Date(input < 1e12 ? input * 1000 : input);
  } else {
    const n = Number(input);
    if (!Number.isNaN(n) && n > 0) {
      d = new Date(n < 1e12 ? n * 1000 : n);
    } else {
      d = new Date(input);
    }
  }
  if (Number.isNaN(d.getTime())) return "—";
  return `${d.toLocaleString()} (${toRelativeTime(d)})`;
}

function toRelativeTime(date: Date) {
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const diffMs = date.getTime() - Date.now(); // negative if in the past
  const abs = Math.abs(diffMs);
  const minute = 60 * 1000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (abs < hour) return rtf.format(Math.round(diffMs / minute), "minute");
  if (abs < day) return rtf.format(Math.round(diffMs / hour), "hour");
  return rtf.format(Math.round(diffMs / day), "day");
}
