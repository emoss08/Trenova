import { clamp, formatBytes } from "@/lib/utils";
import { useContainerLogStore } from "@/stores/docker-store";
import { ContainerStats } from "@/types/docker";
import {
  Activity,
  ArrowDownRight,
  ArrowUpRight,
  Gauge,
  HardDrive,
  Network,
} from "lucide-react";
import { useMemo } from "react";
import {
  Area,
  AreaChart,
  PolarAngleAxis,
  RadialBar,
  RadialBarChart,
  Tooltip as RechartsTooltip,
  ResponsiveContainer,
} from "recharts";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../ui/card";
import { Progress } from "../ui/progress";
import { KV } from "./container-detail-components";

export function StatsTabContent({
  stats,
  cpuHistory,
  memHistory,
  prevStatsRef,
  live,
}: {
  stats: ContainerStats | null;
  cpuHistory: number[];
  memHistory: number[];
  prevStatsRef: React.RefObject<ContainerStats | null>;
  live: boolean;
}) {
  // Subscribe so status changes re-render this component.
  const selectedContainer = useContainerLogStore.get("selectedContainer");

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
    } as const;
  }, [stats, prevStatsRef]);

  const [cpuTrend, memTrend] = [useTrend(cpuHistory), useTrend(memHistory)];

  const isRunning = selectedContainer?.State?.toLowerCase() === "running";

  if (!isRunning) {
    return (
      <div className="text-center text-muted-foreground py-10">
        Container must be running to view stats
      </div>
    );
  }

  if (!stats) {
    return (
      <div className="space-y-4">
        <SkeletonRow />
        <SkeletonRow />
        <div className="text-xs text-muted-foreground text-center">
          Waiting for stats…
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Top: CPU / Memory */}
      <div className="grid gap-4 md:grid-cols-2">
        <MetricCard
          title="CPU Usage"
          icon={<Gauge className="h-4 w-4" />}
          description={`${cpuPct.toFixed(1)}%`}
          gaugeValue={cpuPct}
          trend={cpuTrend}
          footer={
            <MiniArea data={cpuHistory} ariaLabel="CPU trend (last 60s)" />
          }
        />

        <MetricCard
          title="Memory Usage"
          icon={<HardDrive className="h-4 w-4" />}
          description={`${formatBytes(stats.memUsage)} / ${formatBytes(stats.memLimit)}`}
          gaugeValue={memPct}
          trend={memTrend}
          footer={
            <MiniArea data={memHistory} ariaLabel="Memory trend (last 60s)" />
          }
        />
      </div>

      {/* Bottom: Network / Block I/O */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
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
              {ioRates ? `${formatBytes(ioRates.netInPerSec)}/s` : "—"}
            </KV>
            <KV label="Out/sec">
              {ioRates ? `${formatBytes(ioRates.netOutPerSec)}/s` : "—"}
            </KV>
            {ioRates && (
              <div className="pt-1">
                <MiniTwinBars
                  a={ioRates.netInPerSec}
                  b={ioRates.netOutPerSec}
                  aLabel="In"
                  bLabel="Out"
                />
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
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
              {ioRates ? `${formatBytes(ioRates.blkInPerSec)}/s` : "—"}
            </KV>
            <KV label="Write/sec">
              {ioRates ? `${formatBytes(ioRates.blkOutPerSec)}/s` : "—"}
            </KV>
            {ioRates && (
              <div className="pt-1">
                <MiniTwinBars
                  a={ioRates.blkInPerSec}
                  b={ioRates.blkOutPerSec}
                  aLabel="Read"
                  bLabel="Write"
                />
              </div>
            )}
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
  );
}

function MetricCard({
  title,
  icon,
  description,
  gaugeValue,
  trend,
  footer,
}: {
  title: string;
  icon: React.ReactNode;
  description?: string;
  gaugeValue: number; // 0..100
  trend: "up" | "down" | "flat";
  footer?: React.ReactNode;
}) {
  return (
    <Card className="border border-border/60 bg-gradient-to-br from-background to-muted/30 dark:from-muted/10 dark:to-background/20">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-sm">
          {icon} {title}
          <TrendPill trend={trend} />
        </CardTitle>
        {description ? <CardDescription>{description}</CardDescription> : null}
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-4">
          <RadialGauge value={gaugeValue} />
          <div className="flex-1">
            <Progress value={gaugeValue} className="h-2" />
            <div className="mt-3">{footer}</div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function TrendPill({ trend }: { trend: "up" | "down" | "flat" }) {
  const cls =
    trend === "up"
      ? "text-emerald-600 dark:text-emerald-400"
      : trend === "down"
        ? "text-rose-600 dark:text-rose-400"
        : "text-muted-foreground";
  return (
    <span className={`inline-flex items-center gap-1 text-xs ${cls}`}>
      {trend === "up" ? (
        <ArrowUpRight className="h-3.5 w-3.5" />
      ) : trend === "down" ? (
        <ArrowDownRight className="h-3.5 w-3.5" />
      ) : (
        <Activity className="h-3.5 w-3.5" />
      )}
      {trend}
    </span>
  );
}

function RadialGauge({ value, size = 96 }: { value: number; size?: number }) {
  // Clamp to [0,100] for safety
  const v = Math.max(0, Math.min(100, value));
  const data = useMemo(() => [{ name: "value", value: v }], [v]);

  return (
    <div className="relative" style={{ width: size, height: size }}>
      <ResponsiveContainer width="100%" height="100%">
        <RadialBarChart
          data={data}
          startAngle={220}
          endAngle={-40}
          innerRadius="72%"
          outerRadius="100%"
          cx="50%"
          cy="50%"
        >
          <defs>
            <linearGradient id="gaugeFill" x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="var(--primary)" stopOpacity="0.85" />
              <stop
                offset="100%"
                stopColor="var(--primary)"
                stopOpacity="0.55"
              />
            </linearGradient>
          </defs>
          <PolarAngleAxis type="number" domain={[0, 100]} tick={false} />
          <RadialBar
            dataKey="value"
            cornerRadius={8}
            background
            fill="url(#gaugeFill)"
            className="[&_.recharts-radial-bar-background-sector]:fill-muted/60"
          />
        </RadialBarChart>
      </ResponsiveContainer>
      <div className="absolute inset-0 grid place-items-center">
        <span className="text-xs text-muted-foreground">{v.toFixed(0)}%</span>
      </div>
    </div>
  );
}

function MiniArea({ data, ariaLabel }: { data: number[]; ariaLabel?: string }) {
  const chartData = useMemo(
    () => data?.map((v, i) => ({ i, v })) ?? [],
    [data],
  );
  if (!chartData.length)
    return <div className="text-xs text-muted-foreground">No data</div>;

  return (
    <div className="h-14">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={chartData} aria-label={ariaLabel}>
          <defs>
            <linearGradient id="areaFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--primary)" stopOpacity={0.35} />
              <stop
                offset="100%"
                stopColor="var(--primary)"
                stopOpacity={0.05}
              />
            </linearGradient>
          </defs>
          <Area
            type="monotone"
            dataKey="v"
            stroke="var(--muted-foreground)"
            strokeWidth={1.5}
            fill="url(#areaFill)"
            isAnimationActive={false}
          />
          <RechartsTooltip cursor={false} content={<SimpleTooltip />} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

function SimpleTooltip({ active, payload }: any) {
  if (!active || !payload?.length) return null;
  const v = payload[0]?.value as number;
  return (
    <div className="rounded-md bg-popover text-popover-foreground shadow p-1.5 text-xs">
      {v.toFixed(2)}
    </div>
  );
}

function MiniTwinBars({
  a,
  b,
  aLabel,
  bLabel,
}: {
  a: number;
  b: number;
  aLabel: string;
  bLabel: string;
}) {
  const max = Math.max(a, b, 1);
  const aPct = (a / max) * 100;
  const bPct = (b / max) * 100;
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-2">
        <span className="w-10 shrink-0 text-xs text-muted-foreground">
          {aLabel}
        </span>
        <div className="h-1.5 flex-1 rounded bg-muted">
          <div
            className="h-full rounded bg-primary/80"
            style={{ width: `${aPct}%` }}
          />
        </div>
      </div>
      <div className="flex items-center gap-2">
        <span className="w-10 shrink-0 text-xs text-muted-foreground">
          {bLabel}
        </span>
        <div className="h-1.5 flex-1 rounded bg-muted">
          <div
            className="h-full rounded bg-primary/80"
            style={{ width: `${bPct}%` }}
          />
        </div>
      </div>
    </div>
  );
}

function useTrend(series?: number[]): "up" | "down" | "flat" {
  return useMemo(() => {
    if (!series || series.length < 2) return "flat";
    const n = Math.min(8, series.length);
    const last = series[series.length - 1];
    const prevAvg = series.slice(-n, -1).reduce((s, v) => s + v, 0) / (n - 1);
    const delta = last - prevAvg;
    if (Math.abs(delta) < 0.5) return "flat"; // small threshold to avoid noise
    return delta > 0 ? "up" : "down";
  }, [series]);
}

const SkeletonRow = () => (
  <div className="grid gap-4 md:grid-cols-2">
    <div className="rounded-md border p-4">
      <div className="h-4 w-24 rounded bg-muted animate-pulse mb-3" />
      <div className="h-2 w-full rounded bg-muted animate-pulse" />
      <div className="h-14 w-full rounded bg-muted animate-pulse mt-3" />
    </div>
    <div className="rounded-md border p-4">
      <div className="h-4 w-24 rounded bg-muted animate-pulse mb-3" />
      <div className="h-2 w-full rounded bg-muted animate-pulse" />
      <div className="h-14 w-full rounded bg-muted animate-pulse mt-3" />
    </div>
  </div>
);
