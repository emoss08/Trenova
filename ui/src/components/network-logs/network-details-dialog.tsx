/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { DockerNetwork } from "@/types/docker";
import { useCallback, useDeferredValue, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
    Collapsible,
    CollapsibleContent,
    CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from "@/components/ui/tooltip";

import { Check, Clipboard, Download, Search } from "lucide-react";

/** Props:
 * open should normally be (!!network)
 * onOpenChange(false) to close.
 */
export function NetworkDetailsDialog({
  network,
  open,
  onOpenChange,
}: {
  network: DockerNetwork | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [showRaw, setShowRaw] = useState(false);
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);

  const containers = useMemo(() => {
    if (!network?.Containers) return [];
    const entries = Object.entries(network.Containers) as [string, any][];
    if (!deferredQuery) return entries;
    const q = deferredQuery.toLowerCase();
    return entries.filter(([id, c]) => {
      const v4 = (c.IPv4Address || "").toLowerCase();
      const v6 = (c.IPv6Address || "").toLowerCase();
      return (
        id.toLowerCase().includes(q) ||
        (c.Name || "").toLowerCase().includes(q) ||
        v4.includes(q) ||
        v6.includes(q)
      );
    });
  }, [network, deferredQuery]);

  const copy = useCallback(async (text: string, key: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      setTimeout(() => setCopiedKey(null), 1200);
    } catch {
      /* noop */
    }
  }, []);

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
      <DialogContent className="max-w-2xl max-h-[85vh]">
        <DialogHeader className="sticky top-0 z-10 bg-background">
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

            <TooltipProvider delayDuration={200}>
              <div className="flex items-center gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button size="sm" variant="outline" onClick={exportJSON}>
                      <Download className="mr-2 h-4 w-4" />
                      Export JSON
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Download this network object</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      size="icon"
                      variant="outline"
                      onClick={() => network && copy(network.Id, "id")}
                      aria-label="Copy network ID"
                    >
                      {copiedKey === "id" ? (
                        <Check className="h-4 w-4" />
                      ) : (
                        <Clipboard className="h-4 w-4" />
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Copy ID</TooltipContent>
                </Tooltip>
              </div>
            </TooltipProvider>
          </div>
        </DialogHeader>

        <ScrollArea className="h-[520px] pr-4">
          <div className="space-y-4">
            {/* NETWORK INFORMATION */}
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">Network Information</CardTitle>
              </CardHeader>
              <CardContent className="space-y-1 text-sm">
                <KV label="ID">
                  <Mono>{network?.Id.slice(0, 12) ?? "—"}</Mono>
                  {network && (
                    <CopyIcon
                      ariaLabel="Copy network ID"
                      onClick={() => copy(network.Id, "id-info")}
                      active={copiedKey === "id-info"}
                    />
                  )}
                </KV>
                <KV label="Name">{network?.Name ?? "—"}</KV>
                <KV label="Driver">{network?.Driver ?? "—"}</KV>
                <KV label="Scope">
                  <Badge
                    variant={
                      network?.Scope === "local" ? "secondary" : "outline"
                    }
                  >
                    {network?.Scope ?? "—"}
                  </Badge>
                </KV>
                <KV label="Internal">{network?.Internal ? "Yes" : "No"}</KV>
                <KV label="Attachable">{network?.Attachable ? "Yes" : "No"}</KV>
                <KV label="IPv6 Enabled">
                  {network?.EnableIPv6 ? "Yes" : "No"}
                </KV>
                {network?.Created && (
                  <KV label="Created">{formatDate(network.Created)}</KV>
                )}
                {network?.Labels && Object.keys(network.Labels).length > 0 && (
                  <div className="pt-2">
                    <div className="text-xs text-muted-foreground mb-1">
                      Labels
                    </div>
                    <div className="space-y-1">
                      {Object.entries(network.Labels).map(([k, v]) => (
                        <div
                          key={k}
                          className="text-xs flex items-center justify-between gap-2"
                        >
                          <div className="truncate">
                            <span className="font-semibold">{k}: </span>
                            <span className="text-muted-foreground">
                              {String(v)}
                            </span>
                          </div>
                          <CopyIcon
                            ariaLabel="Copy label"
                            onClick={() => copy(`${k}=${v}`, `label-${k}`)}
                            active={copiedKey === `label-${k}`}
                          />
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* IPAM */}
            {network?.IPAM?.Config?.length ? (
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">IPAM Configuration</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  {network.IPAM.Config.map((c, idx) => (
                    <div key={idx} className="text-sm rounded-md border p-2">
                      {c.Subnet && (
                        <KV label="Subnet">
                          <Mono>{c.Subnet}</Mono>
                          <CopyIcon
                            ariaLabel="Copy subnet"
                            onClick={() => copy(c.Subnet!, `subnet-${idx}`)}
                            active={copiedKey === `subnet-${idx}`}
                          />
                        </KV>
                      )}
                      {c.Gateway && (
                        <KV label="Gateway">
                          <Mono>{c.Gateway}</Mono>
                          <CopyIcon
                            ariaLabel="Copy gateway"
                            onClick={() => copy(c.Gateway!, `gateway-${idx}`)}
                            active={copiedKey === `gateway-${idx}`}
                          />
                        </KV>
                      )}
                    </div>
                  ))}
                </CardContent>
              </Card>
            ) : null}

            {/* CONNECTED CONTAINERS */}
            {network?.Containers && (
              <Card>
                <CardHeader className="pb-2">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-sm">
                      Connected Containers
                      <span className="ml-2 text-xs text-muted-foreground">
                        ({Object.keys(network.Containers).length})
                      </span>
                    </CardTitle>
                    <div className="relative">
                      <Search className="pointer-events-none absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                      <Input
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        placeholder="Filter by name, ID, IPv4/IPv6…"
                        className="pl-8 w-64"
                        aria-label="Filter connected containers"
                        onKeyDown={(e) => e.key === "Escape" && setQuery("")}
                      />
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="space-y-2">
                  {containers.length ? (
                    containers.map(([id, c]) => (
                      <div
                        key={id}
                        className="text-sm rounded-md border p-2 space-y-1"
                      >
                        <div className="flex items-center justify-between">
                          <div className="font-medium">{c.Name}</div>
                          <div className="flex items-center gap-2">
                            <Mono>{id.slice(0, 12)}</Mono>
                            <CopyIcon
                              ariaLabel="Copy container ID"
                              onClick={() => copy(id, `cid-${id}`)}
                              active={copiedKey === `cid-${id}`}
                            />
                          </div>
                        </div>
                        {c.IPv4Address && (
                          <KV label="IPv4">
                            <Mono>{c.IPv4Address}</Mono>
                            <CopyIcon
                              ariaLabel="Copy IPv4"
                              onClick={() => copy(c.IPv4Address, `v4-${id}`)}
                              active={copiedKey === `v4-${id}`}
                            />
                          </KV>
                        )}
                        {c.IPv6Address && (
                          <KV label="IPv6">
                            <Mono>{c.IPv6Address}</Mono>
                            <CopyIcon
                              ariaLabel="Copy IPv6"
                              onClick={() => copy(c.IPv6Address, `v6-${id}`)}
                              active={copiedKey === `v6-${id}`}
                            />
                          </KV>
                        )}
                        {c.MacAddress && (
                          <KV label="MAC">
                            <Mono>{c.MacAddress}</Mono>
                          </KV>
                        )}
                      </div>
                    ))
                  ) : (
                    <div className="text-xs text-muted-foreground">
                      No matching containers
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {/* OPTIONS */}
            {network?.Options && Object.keys(network.Options).length > 0 ? (
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">Options</CardTitle>
                </CardHeader>
                <CardContent className="space-y-1">
                  {Object.entries(network.Options).map(([k, v]) => (
                    <div
                      key={k}
                      className="text-xs flex items-center justify-between gap-2"
                    >
                      <div className="space-x-2 truncate">
                        <span className="font-semibold">{k}:</span>
                        <span className="text-muted-foreground">
                          {String(v)}
                        </span>
                      </div>
                      <CopyIcon
                        ariaLabel="Copy option"
                        onClick={() => copy(`${k}=${v}`, `opt-${k}`)}
                        active={copiedKey === `opt-${k}`}
                      />
                    </div>
                  ))}
                </CardContent>
              </Card>
            ) : null}

            {/* RAW JSON */}
            <Collapsible open={showRaw} onOpenChange={setShowRaw}>
              <CollapsibleTrigger asChild>
                <Button variant="ghost" size="sm" className="mt-1">
                  {showRaw ? "Hide" : "Show"} raw network JSON
                </Button>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <ScrollArea className="h-[240px] rounded-md border p-3 mt-2">
                  <pre className="text-xs leading-5">
                    {network ? JSON.stringify(network, null, 2) : "—"}
                  </pre>
                </ScrollArea>
              </CollapsibleContent>
            </Collapsible>

            <Separator />
            <div className="flex items-center justify-end gap-2">
              <Button variant="secondary" onClick={() => onOpenChange(false)}>
                Close
              </Button>
            </div>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}

/* -------------------- Small locals -------------------- */

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

const CopyIcon = ({
  onClick,
  ariaLabel,
  active,
}: {
  onClick: () => void;
  ariaLabel: string;
  active?: boolean;
}) => (
  <Button
    size="icon"
    variant="ghost"
    aria-label={ariaLabel}
    onClick={onClick}
    className="h-7 w-7 shrink-0"
  >
    {active ? <Check className="h-4 w-4" /> : <Clipboard className="h-4 w-4" />}
  </Button>
);

function formatDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString();
}
