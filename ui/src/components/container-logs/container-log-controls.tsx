/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */
import { Badge, BadgeProps } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { summarizeLevels } from "@/lib/docker-utils";

import { useContainerLogStore } from "@/stores/docker-store";
import { BarChart3, ScanText, Search } from "lucide-react";
import { useMemo } from "react";
import {
  Bar,
  BarChart,
  Tooltip as RechartsTooltip,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from "recharts";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";
import { Switch } from "../ui/switch";

export function ContainerLogControls({
  logs,
  filteredCount,
  handleJumpToLatest,
}: {
  logs?: string[];
  filteredCount: number;
  handleJumpToLatest: () => void;
}) {
  const levelStats = useMemo(() => summarizeLevels(logs ?? []), [logs]);
  const chartData = useMemo(
    () => [
      { key: "ERROR", value: levelStats.ERROR },
      { key: "WARN", value: levelStats.WARN },
      { key: "INFO", value: levelStats.INFO },
      { key: "DEBUG", value: levelStats.DEBUG },
    ],
    [levelStats],
  );

  return (
    <div className="flex flex-col p-2 gap-3">
      <ContainerLogControlActions handleJumpToLatest={handleJumpToLatest} />
      <ContainerLogControlsOuter>
        <ContainerLogControlBadges
          filteredCount={filteredCount}
          logs={logs}
          levelStats={levelStats}
        />
        <ContainerLogControlsInner>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart
              data={chartData}
              barCategoryGap={12}
              margin={{ top: 4, right: 0, bottom: 0, left: 0 }}
            >
              <defs>
                <linearGradient id="barFill" x1="0" y1="0" x2="0" y2="1">
                  <stop
                    offset="0%"
                    stopColor="var(--primary)"
                    stopOpacity={0.8}
                  />
                  <stop
                    offset="100%"
                    stopColor="var(--primary)"
                    stopOpacity={0.4}
                  />
                </linearGradient>
              </defs>
              <XAxis hide />
              <YAxis
                hide
                domain={[0, (dataMax: number) => Math.max(5, dataMax)]}
              />
              <RechartsTooltip cursor={false} content={<MiniTooltip />} />
              <Bar dataKey="value" fill="url(#barFill)" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </ContainerLogControlsInner>
      </ContainerLogControlsOuter>
    </div>
  );
}

function ContainerLogControlsOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border bg-card/60 p-2 text-xs w-full">
      {children}
    </div>
  );
}

function ContainerLogControlsInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="h-10 w-64 md:w-64 overflow-hidden">{children}</div>;
}

function ContainerLogControlActions({
  handleJumpToLatest,
}: {
  handleJumpToLatest: () => void;
}) {
  const [searchTerm, setSearchTerm] = useContainerLogStore.use("searchTerm");
  const [tail, setTail] = useContainerLogStore.use("tail");
  const [autoRefresh, setAutoRefresh] = useContainerLogStore.use("autoRefresh");
  const [wrap, setWrap] = useContainerLogStore.use("wrap");
  const [follow, setFollow] = useContainerLogStore.use("follow");
  const [showLineNumbers, setShowLineNumbers] =
    useContainerLogStore.use("showLineNumbers");

  return (
    <div className="flex flex-wrap items-center gap-3">
      <div className="relative grow max-w-[200px]">
        <Input
          icon={<Search className="size-4" />}
          aria-label="Search logs"
          placeholder="Search logs…"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") setSearchTerm("");
          }}
          className="pl-8 max-w-[200px]"
        />
      </div>

      <div className="flex items-center gap-2">
        <Label htmlFor="tail">Tail</Label>
        <Select value={tail} onValueChange={(v) => setTail(v)}>
          <SelectTrigger id="tail" className="w-[140px]">
            <ScanText className="size-4 mr-2" />
            <SelectValue placeholder="Tail lines" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="100">100 lines</SelectItem>
            <SelectItem value="500">500 lines</SelectItem>
            <SelectItem value="1000">1,000 lines</SelectItem>
            <SelectItem value="5000">5,000 lines</SelectItem>
            <SelectItem value="all">All</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <div className="flex items-center gap-2">
          <Switch
            id="auto-refresh"
            checked={autoRefresh}
            onCheckedChange={setAutoRefresh}
          />
          <Label htmlFor="auto-refresh">Auto-refresh</Label>
        </div>
        <div className="flex items-center gap-2">
          <Switch id="wrap" checked={wrap} onCheckedChange={setWrap} />
          <Label htmlFor="wrap">Wrap lines</Label>
        </div>
        <div className="flex items-center gap-2">
          <Switch
            id="follow"
            checked={follow}
            onCheckedChange={(v) => {
              setFollow(v);
              if (v) handleJumpToLatest();
            }}
          />
          <Label htmlFor="follow">Follow tail</Label>
        </div>
        <div className="flex items-center gap-2">
          <Switch
            id="ln"
            checked={showLineNumbers}
            onCheckedChange={setShowLineNumbers}
          />
          <Label htmlFor="ln">Line #</Label>
        </div>
      </div>
    </div>
  );
}

function ContainerLogControlBadges({
  filteredCount,
  logs,
  levelStats,
}: {
  filteredCount: number;
  logs?: string[];
  levelStats: ReturnType<typeof summarizeLevels>;
}) {
  return (
    <div className="flex items-center gap-2">
      <BarChart3 className="h-3.5 w-3.5 opacity-70" />
      <span className="text-muted-foreground">
        Showing <strong className="text-foreground">{filteredCount}</strong> of{" "}
        {logs?.length ?? 0} lines
      </span>
      <Separator orientation="vertical" className="h-4" />
      <LevelBadge label="ERR" value={levelStats.ERROR} variant="inactive" />
      <LevelBadge label="WARN" value={levelStats.WARN} variant="warning" />
      <LevelBadge label="INFO" value={levelStats.INFO} variant="info" />
      <LevelBadge label="DBG" value={levelStats.DEBUG} variant="outline" />
    </div>
  );
}

function MiniTooltip({ active, payload }: any) {
  if (!active || !payload?.length) return null;
  const p = payload[0];
  return (
    <div className="rounded-md bg-popover text-popover-foreground shadow p-1.5 text-xs">
      {p.payload.key}: {p.value}
    </div>
  );
}

function LevelBadge({
  label,
  value,
  variant,
}: {
  label: "ERR" | "WARN" | "INFO" | "DBG";
  value: number;
  variant?: BadgeProps["variant"];
}) {
  const dotColor = useMemo(() => {
    switch (label) {
      case "ERR":
        return "bg-rose-500";
      case "WARN":
        return "bg-amber-500";
      case "INFO":
        return "bg-blue-500";
      case "DBG":
        return "bg-muted-foreground";
    }
  }, [label]);

  return (
    <Badge withDot={false} variant={variant ?? "outline"} className="gap-1">
      <span className={`inline-block h-1.5 w-1.5 rounded-full ${dotColor}`} />
      {label}
      <span className="ml-1 tabular-nums">{value}</span>
    </Badge>
  );
}
