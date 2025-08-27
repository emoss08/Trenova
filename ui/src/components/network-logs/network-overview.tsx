import { DockerNetwork } from "@/types/docker";
import { AlertTriangle, Globe, Network, Server, Shield } from "lucide-react";
import { useMemo } from "react";
import {
  Bar,
  BarChart,
  Tooltip as RechartsTooltip,
  ResponsiveContainer,
  XAxis,
} from "recharts";
import { Badge } from "../ui/badge";
import { Fact } from "./network-details-dialog";

export function NetworkOverview({
  network,
}: {
  network: DockerNetwork | null;
}) {
  const stats = useMemo(() => summarizeNetwork(network), [network]);
  const warnings = useMemo(() => buildWarnings(stats), [stats]);

  return (
    <div className="border-b border-border pb-2">
      <div className="text-sm pb-2 font-semibold">Overview</div>
      <div className="grid gap-3 sm:grid-cols-3 text-sm">
        <div className="rounded-md border p-2">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Network className="size-4" />
            Driver
          </div>
          <div className="mt-1 font-medium capitalize">
            {network?.Driver ?? "—"}
          </div>
        </div>
        <div className="rounded-md border p-2">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Shield className="size-4" />
            Scope
          </div>
          <div className="mt-1">
            <Badge
              withDot={false}
              variant={network?.Scope === "local" ? "secondary" : "active"}
            >
              {network?.Scope ?? "—"}
            </Badge>
          </div>
        </div>
        <div className="rounded-md border p-2">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Server className="size-4" />
            Containers
          </div>
          <div className="mt-1 font-medium">{stats.totalContainers}</div>
        </div>
        <div className="sm:col-span-3 grid gap-3 sm:grid-cols-3">
          <Fact
            label="IPv4"
            value={stats.ipv4Count}
            icon={<Globe className="size-4" />}
          />
          <Fact
            label="IPv6"
            value={stats.ipv6Count}
            icon={<Globe className="size-4" />}
          />
          <div className="rounded-md border p-2">
            <div className="flex items-center gap-2 text-muted-foreground">
              <AlertTriangle className="size-4" />
              Issues
            </div>
            <div className="mt-1 flex flex-wrap gap-2">
              {warnings.length ? (
                warnings.map((w, i) => (
                  <Badge
                    withDot={false}
                    key={i}
                    variant="warning"
                    className="text-[10px]"
                  >
                    {w}
                  </Badge>
                ))
              ) : (
                <Badge withDot={false} variant="active" className="text-[10px]">
                  None detected
                </Badge>
              )}
            </div>
          </div>
        </div>
        {stats.subnetData.length ? (
          <div className="sm:col-span-3">
            <div className="text-xs text-muted-foreground mb-1">
              Containers per subnet
            </div>
            <div className="h-28 w-full rounded-md border">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart
                  data={stats.subnetData}
                  margin={{ top: 8, right: 8, bottom: 0, left: 8 }}
                >
                  <XAxis dataKey="key" hide />
                  <RechartsTooltip
                    cursor={false}
                    formatter={(v: any, _n: any, p: any) => [
                      v,
                      p?.payload?.key,
                    ]}
                  />
                  <Bar
                    dataKey="value"
                    radius={[4, 4, 0, 0]}
                    fill="var(--primary)"
                  />
                </BarChart>
              </ResponsiveContainer>
            </div>
            <div className="mt-1 flex flex-wrap gap-1">
              {stats.subnetData.map((s) => (
                <Badge
                  withDot={false}
                  key={s.key}
                  variant="outline"
                  className="text-[10px]"
                >
                  {s.key}: {s.value}
                </Badge>
              ))}
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function summarizeNetwork(network: DockerNetwork | null) {
  const totalContainers = network?.Containers
    ? Object.keys(network.Containers).length
    : 0;
  let ipv4Count = 0;
  let ipv6Count = 0;
  const v4Map = new Map<string, string[]>();
  const v6Map = new Map<string, string[]>();

  if (network?.Containers) {
    for (const [id, c] of Object.entries(network.Containers) as [
      string,
      any,
    ][]) {
      const v4 = stripCidrV4(c.IPv4Address);
      const v6 = stripCidrV6(c.IPv6Address);
      if (v4) {
        ipv4Count++;
        v4Map.set(v4, [...(v4Map.get(v4) || []), c.Name || id.slice(0, 12)]);
      }
      if (v6) {
        ipv6Count++;
        v6Map.set(v6, [...(v6Map.get(v6) || []), c.Name || id.slice(0, 12)]);
      }
    }
  }

  // subnet distribution (IPv4)
  const subnets = (network?.IPAM?.Config || [])
    .map((c: any) => String(c.Subnet || ""))
    .filter(Boolean);

  const subnetCounts: Record<string, number> = {};
  if (subnets.length && network?.Containers) {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    for (const [_id, c] of Object.entries(network.Containers) as [
      string,
      any,
    ][]) {
      const ip = stripCidrV4(c.IPv4Address);
      if (!ip) continue;
      const hit = subnets.find((cidr) => isIpInCidr(ip, cidr));
      const key = hit || "other";
      subnetCounts[key] = (subnetCounts[key] || 0) + 1;
    }
  }

  const subnetData = Object.entries(subnetCounts).map(([key, value]) => ({
    key,
    value,
  }));

  // duplicates
  const v4Dup = [...v4Map.entries()].filter(([, arr]) => arr.length > 1);
  const v6Dup = [...v6Map.entries()].filter(([, arr]) => arr.length > 1);

  return {
    totalContainers,
    ipv4Count,
    ipv6Count,
    v4Dup,
    v6Dup,
    subnetData,
  } as const;
}

function buildWarnings(stats: ReturnType<typeof summarizeNetwork>) {
  const out: string[] = [];
  if (stats.v4Dup.length) out.push(`${stats.v4Dup.length} duplicate IPv4`);
  if (stats.v6Dup.length) out.push(`${stats.v6Dup.length} duplicate IPv6`);
  if (!stats.totalContainers) out.push("no containers");
  return out;
}

function stripCidrV4(ip?: string | null) {
  if (!ip) return "";
  const m = String(ip).match(/^(\d{1,3}(?:\.\d{1,3}){3})/);
  return m ? m[1] : "";
}

function stripCidrV6(ip?: string | null) {
  if (!ip) return "";
  const m = String(ip).match(/^([a-fA-F0-9:]+)/);
  return m ? m[1] : "";
}

function ipToInt(ip: string) {
  return (
    ip
      .split(".")
      .reduce((acc, o) => (acc << 8) + (parseInt(o, 10) & 255), 0) >>> 0
  );
}

function isIpInCidr(ip: string, cidr: string) {
  const m = cidr.match(/^(\d{1,3}(?:\.\d{1,3}){3})\/(\d{1,2})$/);
  if (!m) return false;
  const base = ipToInt(m[1]);
  const bits = Math.max(0, Math.min(32, parseInt(m[2], 10)));
  const mask = bits === 0 ? 0 : (~0 << (32 - bits)) >>> 0;
  return (ipToInt(ip) & mask) === (base & mask);
}
