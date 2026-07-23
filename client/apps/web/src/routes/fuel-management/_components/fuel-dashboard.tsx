import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@trenova/shared/components/ui/chart";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { FuelDashboardEntry } from "@/lib/graphql/fuel-surcharge";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { Fuel, TrendingDown, TrendingUp } from "lucide-react";
import { useMemo, useState } from "react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";

const priceChartConfig = {
  price: {
    label: "Diesel $/gal",
    theme: { light: "#2a78d6", dark: "#3987e5" },
  },
} satisfies ChartConfig;

const RANGE_OPTIONS = [
  { label: "13w", value: 13 },
  { label: "26w", value: 26 },
  { label: "52w", value: 52 },
] as const;

function formatWeekOf(priceDate: string) {
  const date = new Date(`${priceDate}T00:00:00Z`);
  return date.toLocaleDateString(undefined, {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function shortDate(priceDate: string) {
  const date = new Date(`${priceDate}T00:00:00Z`);
  return date.toLocaleDateString(undefined, {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
  });
}

export default function FuelDashboard() {
  const { data: entries, isLoading } = useQuery(queries.fuelSurcharge.dashboard());
  const [selectedIndexId, setSelectedIndexId] = useState<string | null>(null);
  const [range, setRange] = useState<number>(26);

  const activeEntries = entries ?? [];
  const defaultEntry =
    activeEntries.find((entry) => entry.index.eiaSeriesId === "EMD_EPD2D_PTE_NUS_DPG") ??
    activeEntries[0];
  const selectedEntry =
    activeEntries.find((entry) => entry.index.id === selectedIndexId) ?? defaultEntry;

  const latestWeek = defaultEntry?.latest?.priceDate;

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-96" />
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 8 }).map((_, index) => (
            <Skeleton key={index} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-72" />
      </div>
    );
  }

  if (activeEntries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center">
        <div className="flex size-12 items-center justify-center rounded-full bg-muted">
          <Fuel className="size-5 text-muted-foreground" />
        </div>
        <p className="mt-3 text-sm font-medium">No fuel indices yet</p>
        <p className="mt-1 max-w-md text-xs text-muted-foreground">
          Enable the EIA Fuel Prices integration to auto-provision the DOE diesel indices and start
          ingesting weekly prices, or create a custom index from the Fuel Indices tab.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {latestWeek && (
        <p className="text-sm text-muted-foreground">
          Latest DOE price week:{" "}
          <span className="font-medium text-foreground">Mon {formatWeekOf(latestWeek)}</span>
          {" · "}surcharge rates roll forward on each program&apos;s effective day
        </p>
      )}

      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
        {activeEntries.map((entry) => (
          <IndexPriceCard
            key={entry.index.id}
            entry={entry}
            selected={selectedEntry?.index.id === entry.index.id}
            onSelect={() => setSelectedIndexId(entry.index.id)}
          />
        ))}
      </div>

      {selectedEntry && (
        <PriceTrendChart
          indexId={selectedEntry.index.id}
          indexName={selectedEntry.index.name}
          range={range}
          onRangeChange={setRange}
        />
      )}
    </div>
  );
}

function IndexPriceCard({
  entry,
  selected,
  onSelect,
}: {
  entry: FuelDashboardEntry;
  selected: boolean;
  onSelect: () => void;
}) {
  const delta = entry.delta !== null && entry.delta !== undefined ? Number(entry.delta) : null;
  const isUp = delta !== null && delta > 0;
  const isDown = delta !== null && delta < 0;

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "rounded-lg border bg-card p-3 text-left transition-colors hover:bg-muted/50",
        selected && "border-primary/60 ring-1 ring-primary/30",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-xs font-medium text-muted-foreground">
          {entry.index.name}
        </span>
        {entry.index.source === "Custom" ? (
          <span className="rounded bg-muted px-1.5 py-0.5 text-2xs text-muted-foreground">
            {entry.index.region || "Custom"}
          </span>
        ) : (
          entry.index.region && (
            <span className="shrink-0 rounded bg-primary/10 px-1.5 py-0.5 text-2xs text-primary">
              {entry.index.region}
            </span>
          )
        )}
      </div>
      <div className="mt-1.5 flex items-baseline gap-2">
        <span className="text-xl font-semibold tabular-nums">
          {entry.latest ? `$${Number(entry.latest.price).toFixed(3)}` : "—"}
        </span>
        {delta !== null && delta !== 0 && (
          <span
            className={cn(
              "flex items-center gap-0.5 text-xs font-medium tabular-nums",
              isUp && "text-red-600 dark:text-red-400",
              isDown && "text-emerald-600 dark:text-emerald-400",
            )}
          >
            {isUp ? <TrendingUp className="size-3" /> : <TrendingDown className="size-3" />}
            {delta > 0 ? "+" : ""}
            {delta.toFixed(3)}
          </span>
        )}
      </div>
      <p className="mt-1 text-2xs text-muted-foreground">
        {entry.latest ? `Week of ${shortDate(entry.latest.priceDate)}` : "No price data yet"}
      </p>
    </button>
  );
}

function PriceTrendChart({
  indexId,
  indexName,
  range,
  onRangeChange,
}: {
  indexId: string;
  indexName: string;
  range: number;
  onRangeChange: (value: number) => void;
}) {
  const { data: history, isLoading } = useQuery(queries.fuelSurcharge.priceHistory(indexId, range));

  const chartData = useMemo(
    () =>
      (history ?? [])
        .slice()
        .reverse()
        .map((point) => ({
          label: shortDate(point.priceDate),
          price: Number(point.price),
        })),
    [history],
  );

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">{indexName} — weekly diesel price</CardTitle>
        <div className="flex gap-1">
          {RANGE_OPTIONS.map((option) => (
            <Button
              key={option.value}
              type="button"
              size="sm"
              variant={range === option.value ? "secondary" : "ghost"}
              onClick={() => onRangeChange(option.value)}
              className="h-7 px-2 text-xs"
            >
              {option.label}
            </Button>
          ))}
        </div>
      </CardHeader>
      <CardContent className="p-4">
        {isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : chartData.length === 0 ? (
          <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">
            No price history for this index yet
          </div>
        ) : (
          <ChartContainer config={priceChartConfig} className="h-64 w-full">
            <LineChart data={chartData} margin={{ left: 4, right: 12, top: 8 }}>
              <CartesianGrid vertical={false} strokeDasharray="3 3" />
              <XAxis
                dataKey="label"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={32}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                width={52}
                domain={["auto", "auto"]}
                tickFormatter={(value: number) => `$${value.toFixed(2)}`}
              />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Line
                dataKey="price"
                type="monotone"
                stroke="var(--color-price)"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4 }}
              />
            </LineChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}
