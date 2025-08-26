import { DockerNetwork } from "@/types/docker";
import { Search } from "lucide-react";
import { useDeferredValue, useMemo, useState } from "react";
import {
  CopyIcon,
  KV,
  Mono,
} from "../container-details/container-detail-components";
import { Badge } from "../ui/badge";
import { Input } from "../ui/input";

export function NetworkInformation({
  network,
  copiedKey,
  copy,
}: {
  network: DockerNetwork | null;
  copiedKey: string | null;
  copy: (text: string, key: string) => void;
}) {
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);

  const entries = useMemo(() => {
    if (!network?.Containers) return [] as [string, any][];
    const e = Object.entries(network.Containers) as [string, any][];
    e.sort(
      (a, b) =>
        (a[1]?.Name || "").localeCompare(b[1]?.Name || "") ||
        a[0].localeCompare(b[0]),
    );
    return e;
  }, [network?.Containers]);

  const containers = useMemo(() => {
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
  }, [entries, deferredQuery]);

  return (
    <div className="flex flex-col gap-2 py-2">
      <div className="py-2">
        <div className="text-sm pb-2 font-semibold">Network Information</div>
        <div className="space-y-1 text-sm border rounded-md p-2">
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
              withDot={false}
              variant={network?.Scope === "local" ? "secondary" : "outline"}
            >
              {network?.Scope ?? "—"}
            </Badge>
          </KV>
          <KV label="Internal">{network?.Internal ? "Yes" : "No"}</KV>
          <KV label="Attachable">{network?.Attachable ? "Yes" : "No"}</KV>
          <KV label="IPv6 Enabled">{network?.EnableIPv6 ? "Yes" : "No"}</KV>
          {network?.Created && (
            <KV label="Created">{formatDate(network.Created)}</KV>
          )}
          {network?.Labels && Object.keys(network.Labels).length > 0 && (
            <div className="pt-2">
              <div className="text-xs text-muted-foreground mb-1">Labels</div>
              <div className="space-y-1">
                {Object.entries(network.Labels).map(([k, v]) => (
                  <div
                    key={k}
                    className="text-xs flex items-center justify-between gap-2"
                  >
                    <div className="truncate">
                      <span className="font-semibold">{k}: </span>
                      <span className="text-muted-foreground">{String(v)}</span>
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
        </div>
      </div>
      {network?.IPAM?.Config?.length ? (
        <div className="border-b border-border pb-2">
          <div className="text-sm pb-2 font-semibold">IPAM Configuration</div>
          <div className="space-y-2">
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
          </div>
        </div>
      ) : null}
      {network?.Containers && (
        <div className="border-b border-border pb-2">
          <div className="text-sm pb-2 font-semibold">
            <div className="flex items-center justify-between">
              <div className="text-sm">
                Connected Containers
                <span className="ml-2 text-xs text-muted-foreground">
                  ({Object.keys(network.Containers).length})
                </span>
              </div>
              <div className="relative">
                <Input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Filter by name, ID, IPv4/IPv6…"
                  className="pl-8 w-64"
                  aria-label="Filter connected containers"
                  icon={<Search className="size-4" />}
                  onKeyDown={(e) => e.key === "Escape" && setQuery("")}
                />
              </div>
            </div>
          </div>
          <div className="space-y-2">
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
          </div>
        </div>
      )}
      {network?.Options && Object.keys(network.Options).length > 0 ? (
        <div className="border-b border-border pb-2">
          <div className="text-sm pb-2 font-semibold">Options</div>
          <div className="space-y-1">
            {Object.entries(network.Options).map(([k, v]) => (
              <div
                key={k}
                className="text-xs flex items-center justify-between gap-2"
              >
                <div className="space-x-2 truncate">
                  <span className="font-semibold">{k}:</span>
                  <span className="text-muted-foreground">{String(v)}</span>
                </div>
                <CopyIcon
                  ariaLabel="Copy option"
                  onClick={() => copy(`${k}=${v}`, `opt-${k}`)}
                  active={copiedKey === `opt-${k}`}
                />
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString();
}
