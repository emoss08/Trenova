/* eslint-disable react/display-name */
/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { PTO_COLOR_SCHEME_KEY } from "@/constants/env";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { queries } from "@/lib/queries";
import type { PTOChartDataPoint } from "@/services/worker";
import { useUser } from "@/stores/user-store";
import { ResponsiveBar } from "@nivo/bar";
import { ColorSchemeId, colorSchemes } from "@nivo/colors";
import { useSuspenseQuery } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import { memo, useMemo } from "react";

interface PTOChartProps {
  startDate: number;
  endDate: number;
  type?: string;
}

const CustomTooltip = memo(({ data, id, value }: any) => {
  const workers = data.workers?.[id] || [];

  return (
    <div className="bg-popover text-popover-foreground border border-border rounded-lg p-3 shadow-xl min-w-[150px]">
      <div className="flex items-center gap-2 mb-2">
        <div className="text-sm font-semibold">{id}</div>
        <div className="text-sm opacity-70">({value})</div>
      </div>
      {workers.length > 0 && (
        <div className="border-t border-border pt-2">
          <div className="text-xs font-medium opacity-60 mb-1">Workers:</div>
          <div className="text-xs space-y-0.5">
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
});

export default function PTOChart({ startDate, endDate, type }: PTOChartProps) {
  const user = useUser();
  const [colorScheme, setColorScheme] = useLocalStorage(
    PTO_COLOR_SCHEME_KEY,
    "nivo",
  );

  const query = useSuspenseQuery({
    ...queries.worker.getPTOChartData({
      startDate,
      endDate,
      type,
      timezone: user?.timezone,
    }),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });

  const chartData = useMemo(() => {
    if (!query.data || query.data.length === 0) {
      return [];
    }

    return query.data.map((d: PTOChartDataPoint) => ({
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
  }, [query.data]);

  if (query.isLoading) {
    return <Skeleton className="h-[400px] w-full" />;
  }

  if (query.isError) {
    return (
      <div className="h-[400px] w-full flex items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-destructive">Failed to load chart data</p>
          <p className="text-xs text-muted-foreground">
            {query.error?.message || "An error occurred"}
          </p>
        </div>
      </div>
    );
  }

  if (!chartData || chartData.length === 0) {
    return (
      <div className="h-[400px] w-full flex items-center justify-center">
        <p className="text-sm text-muted-foreground">
          No PTO data available for the selected period
        </p>
      </div>
    );
  }

  return (
    <div className="relative size-full">
      <div className="absolute top-0 right-0 z-10">
        <div className="flex items-center gap-2">
          <div className="text-xs font-medium">Color Scheme:</div>
          <Select
            value={colorScheme}
            onValueChange={(v) => setColorScheme(v as ColorSchemeId)}
          >
            <SelectTrigger className="w-[120px] h-8 text-xs">
              <SelectValue placeholder="Color" />
            </SelectTrigger>
            <SelectContent className="min-w-[170px]">
              {Object.keys(colorSchemes).map((scheme) => (
                <SelectItem key={scheme} value={scheme} className="text-xs">
                  {scheme}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <ResponsiveBar
        data={chartData as any}
        keys={[
          "Vacation",
          "Sick",
          "Holiday",
          "Bereavement",
          "Maternity",
          "Paternity",
          "Personal",
        ]}
        indexBy="date"
        margin={{ top: 50, right: 130, bottom: 50, left: 60 }}
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
          text: {
            fill: "var(--foreground)",
          },
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
          labels: {
            text: {
              fill: "var(--foreground)",
            },
          },
        }}
        enableGridY={true}
        role="application"
        ariaLabel="PTO chart"
        tooltip={CustomTooltip}
      />
    </div>
  );
}
