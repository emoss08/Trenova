import { Skeleton } from "@/components/ui/skeleton";
import { API_BASE_URL } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { ResponsiveBar } from "@nivo/bar";
import { useQuery } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import { memo, useMemo } from "react";

interface PTOChartWorker {
  id: string;
  firstName: string;
  lastName: string;
  ptoType: string;
}

interface PTOChartDataPoint {
  date: string;
  vacation: number;
  sick: number;
  holiday: number;
  bereavement: number;
  maternity: number;
  paternity: number;
  personal: number;
  workers: Record<string, PTOChartWorker[]>;
}

interface PTOChartProps {
  startDate: number;
  endDate: number;
  type?: string;
  workerId?: string;
  className?: string;
}

async function fetchPTOChartData(
  startDate: number,
  endDate: number,
  type?: string,
  workerId?: string,
): Promise<PTOChartDataPoint[]> {
  const url = new URL(`${API_BASE_URL}/workers/pto/chart/`, window.location.origin);
  url.searchParams.set("startDateFrom", startDate.toString());
  url.searchParams.set("startDateTo", endDate.toString());
  url.searchParams.set("status", "Approved");
  if (type) url.searchParams.set("type", type);
  if (workerId) url.searchParams.set("workerId", workerId);

  const response = await fetch(url.href, { credentials: "include" });
  if (!response.ok) {
    throw new Error("Failed to fetch PTO chart data");
  }
  return response.json();
}

const CustomTooltip = memo(
  ({ workers, id, value }: { workers: PTOChartWorker[]; id: string; value: number }) => {
    return (
      <div className="min-w-[150px] rounded-lg border border-border bg-popover p-3 text-popover-foreground shadow-xl">
        <div className="mb-2 flex items-center gap-2">
          <div className="text-sm font-semibold">{id}</div>
          <div className="text-sm opacity-70">({value})</div>
        </div>
        {workers.length > 0 && (
          <div className="border-t border-border pt-2">
            <div className="mb-1 text-xs font-medium opacity-60">Workers:</div>
            <div className="space-y-0.5 text-xs">
              {workers.slice(0, 5).map((worker: PTOChartWorker, idx: number) => (
                <div key={`${worker.id}-${idx}`}>
                  • {worker.firstName} {worker.lastName}
                </div>
              ))}
              {workers.length > 5 && (
                <div className="text-muted-foreground">+{workers.length - 5} more</div>
              )}
            </div>
          </div>
        )}
      </div>
    );
  },
);

CustomTooltip.displayName = "CustomTooltip";

function ChartLoadingState({ className }: { className?: string }) {
  return (
    <div className={cn("flex h-full w-full flex-col items-center justify-center", className)}>
      <Skeleton className="size-full" />
    </div>
  );
}

function ChartEmptyState() {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <p className="text-sm text-muted-foreground">No PTO data available for the selected period</p>
    </div>
  );
}

function ChartErrorState({ message }: { message?: string }) {
  return (
    <div className="flex h-full w-full items-center justify-center">
      <div className="text-center">
        <p className="text-sm text-destructive">Failed to load chart data</p>
        <p className="text-xs text-muted-foreground">{message || "An error occurred"}</p>
      </div>
    </div>
  );
}

export function PTOChart({ startDate, endDate, type, workerId, className }: PTOChartProps) {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["pto-chart", startDate, endDate, type, workerId],
    queryFn: () => fetchPTOChartData(startDate, endDate, type, workerId),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: startDate > 0 && endDate > 0,
  });

  const { barData, workersMap } = useMemo(() => {
    if (!data || data.length === 0)
      return {
        barData: [] as {
          date: string;
          Vacation: number;
          Sick: number;
          Holiday: number;
          Bereavement: number;
          Maternity: number;
          Paternity: number;
          Personal: number;
        }[],
        workersMap: new Map<string, Record<string, PTOChartWorker[]>>(),
      };

    const wMap = new Map<string, Record<string, PTOChartWorker[]>>();
    const bars = data.map((d) => {
      const dateKey = format(parseISO(d.date), "MMM dd");
      wMap.set(dateKey, d.workers);
      return {
        date: dateKey,
        Vacation: d.vacation || 0,
        Sick: d.sick || 0,
        Holiday: d.holiday || 0,
        Bereavement: d.bereavement || 0,
        Maternity: d.maternity || 0,
        Paternity: d.paternity || 0,
        Personal: d.personal || 0,
      };
    });

    return { barData: bars, workersMap: wMap };
  }, [data]);

  if (isLoading) {
    return <ChartLoadingState className={className} />;
  }

  if (isError) {
    return <ChartErrorState message={error?.message} />;
  }

  if (!barData || barData.length === 0) {
    return <ChartEmptyState />;
  }

  return (
    <div className={cn("relative size-full", className)}>
      <ResponsiveBar
        data={barData}
        keys={["Vacation", "Sick", "Holiday", "Bereavement", "Maternity", "Paternity", "Personal"]}
        indexBy="date"
        margin={{ top: 20, right: 130, bottom: 50, left: 50 }}
        padding={0.3}
        valueScale={{ type: "linear" }}
        indexScale={{ type: "band", round: true }}
        colors={{ scheme: "nivo" }}
        borderColor={{ from: "color", modifiers: [["darker", 0.6]] }}
        axisTop={null}
        axisRight={null}
        axisBottom={{
          tickSize: 5,
          tickPadding: 5,
          tickRotation: -45,
          legendPosition: "middle",
          legendOffset: 40,
        }}
        axisLeft={{
          tickSize: 5,
          tickPadding: 5,
          tickRotation: 0,
          legend: "Count",
          legendPosition: "middle",
          legendOffset: -40,
        }}
        labelSkipWidth={12}
        labelSkipHeight={12}
        legends={[
          {
            dataFrom: "keys",
            anchor: "bottom-right",
            direction: "column",
            translateX: 120,
            itemsSpacing: 3,
            itemWidth: 100,
            itemHeight: 16,
            itemDirection: "left-to-right",
            itemOpacity: 1,
            symbolSize: 14,
            symbolShape: "square",
          },
        ]}
        theme={{
          axis: {
            ticks: {
              text: {
                fontSize: 11,
                fill: "hsl(var(--foreground))",
              },
            },
            legend: {
              text: {
                fontSize: 12,
                fill: "hsl(var(--foreground))",
              },
            },
            domain: {
              line: {
                stroke: "hsl(var(--border))",
              },
            },
          },
          grid: {
            line: {
              stroke: "hsl(var(--border))",
              strokeWidth: 1,
              strokeDasharray: "3 3",
            },
          },
          legends: {
            text: {
              fontSize: 11,
              fill: "hsl(var(--foreground))",
            },
          },
          tooltip: {
            container: {
              background: "hsl(var(--popover))",
              color: "hsl(var(--popover-foreground))",
              fontSize: 12,
              borderRadius: 8,
              boxShadow: "0 4px 6px -1px rgb(0 0 0 / 0.1)",
            },
          },
        }}
        enableGridY={true}
        role="application"
        ariaLabel="PTO chart"
        tooltip={({ id, value, indexValue }) => {
          const dateWorkers = workersMap.get(indexValue as string);
          const workers = dateWorkers?.[id as string] || [];
          return <CustomTooltip workers={workers} id={id as string} value={value} />;
        }}
      />
    </div>
  );
}
