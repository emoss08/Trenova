import { LoadingSkeletonState } from "@/components/loading-skeleton";
import { useTheme } from "@/components/theme-provider";
import { useLocalStorage } from "@/hooks/use-local-storage";
import type { PTOChartDataPoint } from "@/types/worker";
import { ResponsiveBar } from "@nivo/bar";
import { type ColorSchemeId } from "@nivo/colors";
import { format, parseISO } from "date-fns";
import { useEffect, useMemo } from "react";
import { ApprovedChartOptions, isApprovedPTOColorScheme } from "./approved-chart-options";

const CustomTooltip = ({ data, id, value }: any) => {
  const workers = data.workers?.[id] || [];

  return (
    <div className="min-w-37.5 rounded-lg border border-border bg-popover p-3 text-popover-foreground shadow-xl">
      <div className="mb-2 flex items-center gap-2">
        <div className="text-sm font-semibold">{id}</div>
        <div className="text-sm opacity-70">({value})</div>
      </div>
      {workers.length > 0 && (
        <div className="border-t border-border pt-2">
          <div className="space-y-0.5 text-xs">
            {workers.map((worker: any) => (
              <div key={worker.id}>
                • {worker.firstName} {worker.lastName}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default function ApprovedPTOChart({
  data,
  isLoading,
  isError,
  errorMessage,
}: {
  data: PTOChartDataPoint[];
  isLoading: boolean;
  isError: boolean;
  errorMessage?: string;
}) {
  const { theme } = useTheme();

  const DEFAULT_APPROVED_PTO_COLOR_SCHEME: ColorSchemeId = useMemo(() => {
    return theme === "dark" ? "greys" : "nivo";
  }, [theme]);

  const [storedColorScheme, setColorScheme] = useLocalStorage(
    "pto-chart-color-scheme",
    DEFAULT_APPROVED_PTO_COLOR_SCHEME,
  );
  const colorScheme = isApprovedPTOColorScheme(storedColorScheme)
    ? storedColorScheme
    : DEFAULT_APPROVED_PTO_COLOR_SCHEME;

  useEffect(() => {
    if (!isApprovedPTOColorScheme(storedColorScheme)) {
      setColorScheme(DEFAULT_APPROVED_PTO_COLOR_SCHEME);
    }
  }, [setColorScheme, storedColorScheme, DEFAULT_APPROVED_PTO_COLOR_SCHEME]);

  const chartData = useMemo(() => {
    if (!data || data.length === 0) {
      return [];
    }

    return data.map((d: PTOChartDataPoint) => ({
      date: format(parseISO(d.date), "MMM dd"),
      Vacation: d.vacation || 0,
      Sick: d.sick || 0,
      Holiday: d.holiday || 0,
      Bereavement: d.bereavement || 0,
      Maternity: d.maternity || 0,
      Paternity: d.paternity || 0,
      Personal: d.personal || 0,
      workers: d.workers,
    }));
  }, [data]);

  if (isLoading) {
    return <LoadingSkeletonState description="Loading chart data..." className="h-75 w-full" />;
  }

  if (isError) {
    return (
      <div className="flex h-100 w-full items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-destructive">Failed to load chart data</p>
          <p className="text-xs text-muted-foreground">{errorMessage || "An error occurred"}</p>
        </div>
      </div>
    );
  }

  if (!chartData || chartData.length === 0) {
    return (
      <div className="flex h-100 w-full items-center justify-center">
        <p className="text-sm text-muted-foreground">
          No PTO data available for the selected period
        </p>
      </div>
    );
  }

  return (
    <ChartOuter>
      <ApprovedChartOptions
        colorScheme={colorScheme}
        setColorScheme={(value) => setColorScheme(value)}
      />
      <p className="mb-2 text-xs text-muted-foreground">
        Counts are daily occupancy. Multi-day PTO appears on each covered day.
      </p>
      <ResponsiveBar
        data={chartData as any}
        keys={["Vacation", "Sick", "Holiday", "Bereavement", "Maternity", "Paternity", "Personal"]}
        indexBy="date"
        margin={{ top: 20, right: 130, bottom: 60, left: 60 }}
        padding={0.3}
        valueScale={{ type: "linear" }}
        indexScale={{ type: "band", round: true }}
        colors={{ scheme: colorScheme as ColorSchemeId }}
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
            symbolSize: 16,
            symbolShape: "square",
          },
        ]}
        theme={{
          axis: {
            ticks: {
              text: {
                fontSize: 11,
                fill: "var(--foreground)",
                fontFamily: "var(--font-table)",
              },
            },
            legend: {
              text: {
                fontSize: 12,
                fill: "var(--foreground)",
              },
            },
            domain: {
              line: {
                stroke: "var(--border)",
              },
            },
          },
          grid: {
            line: {
              stroke: "var(--border)",
              strokeWidth: 1,
              strokeDasharray: "3 3",
            },
          },
          legends: {
            text: {
              fontSize: 11,
              fill: "var(--foreground)",
              fontFamily: "var(--font-table)",
            },
          },
        }}
        enableGridY={true}
        role="application"
        ariaLabel="PTO chart"
        tooltip={CustomTooltip}
      />
    </ChartOuter>
  );
}
function ChartOuter({ children }: { children: React.ReactNode }) {
  return <div className="relative h-[300px] w-full">{children}</div>;
}
