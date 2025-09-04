/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import type { PTOChartDataPoint } from "@/services/worker";
import { useQuery } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import { memo, useCallback, useMemo } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Rectangle,
  XAxis,
  YAxis,
} from "recharts";

const PTO_COLORS = {
  vacation: "#9333ea", // purple-600
  sick: "#dc2626", // red-600
  holiday: "#2563eb", // blue-600
  bereavement: "#16a34a", // green-600
  maternity: "#db2777", // pink-600
  paternity: "#0d9488", // teal-600
  personal: "#6b7280", // gray-600
} as const;

const PTO_LABELS = {
  vacation: "Vacation",
  sick: "Sick",
  holiday: "Holiday",
  bereavement: "Bereavement",
  maternity: "Maternity",
  paternity: "Paternity",
  personal: "Personal",
} as const;

interface PTOChartProps {
  startDate: number;
  endDate: number;
  type?: string;
}

interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{
    dataKey: string;
    value: number;
    color: string;
    payload: PTOChartDataPoint;
  }>;
  label?: string | number;
}

const CustomTooltip = memo(function CustomTooltip({
  active,
  payload,
  label,
}: CustomTooltipProps) {
  const formattedDate = useMemo(
    () => (label ? format(parseISO(String(label)), "MMM dd, yyyy") : ""),
    [label],
  );

  const tooltipData = useMemo(() => {
    if (!active || !payload || payload.length === 0) return null;

    const data = payload[0].payload;
    const entries = payload
      .filter((entry) => entry.value > 0)
      .map((entry) => {
        const ptoType = entry.dataKey as keyof typeof PTO_LABELS;
        const ptoTypeKey = PTO_LABELS[ptoType];
        const workers = data.workers?.[ptoTypeKey] || [];

        return {
          ptoType,
          ptoTypeKey,
          value: entry.value,
          color: entry.color,
          workers,
        };
      });

    return entries;
  }, [active, payload]);

  if (!tooltipData) return null;

  return (
    <div className="bg-background border border-border rounded-lg p-3 shadow-lg">
      <p className="font-medium text-sm mb-2">{formattedDate}</p>
      <div className="space-y-1">
        {tooltipData.map(({ ptoType, ptoTypeKey, value, color, workers }) => (
          <div key={ptoType} className="text-xs">
            <div className="flex items-center gap-2 mb-1">
              <div
                className="w-3 h-3 rounded-sm"
                style={{ backgroundColor: color }}
              />
              <span className="font-medium">
                {ptoTypeKey}: {value}
              </span>
            </div>
            {workers.length > 0 && (
              <div className="ml-5 text-muted-foreground">
                {workers.map(
                  (
                    worker: { id: string; firstName: string; lastName: string },
                    index: number,
                  ) => (
                    <div key={worker.id}>
                      {worker.firstName} {worker.lastName}
                      {index < workers.length - 1 && ", "}
                    </div>
                  ),
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
});

const chartConfig = {
  vacation: {
    label: PTO_LABELS.vacation,
    color: PTO_COLORS.vacation,
  },
  sick: {
    label: PTO_LABELS.sick,
    color: PTO_COLORS.sick,
  },
  holiday: {
    label: PTO_LABELS.holiday,
    color: PTO_COLORS.holiday,
  },
  bereavement: {
    label: PTO_LABELS.bereavement,
    color: PTO_COLORS.bereavement,
  },
  maternity: {
    label: PTO_LABELS.maternity,
    color: PTO_COLORS.maternity,
  },
  paternity: {
    label: PTO_LABELS.paternity,
    color: PTO_COLORS.paternity,
  },
  personal: {
    label: PTO_LABELS.personal,
    color: PTO_COLORS.personal,
  },
} satisfies ChartConfig;

const STACK_ORDER = [
  "vacation",
  "sick",
  "holiday",
  "bereavement",
  "maternity",
  "paternity",
  "personal",
];

const getBarRadius = (
  dataPoint: PTOChartDataPoint,
  dataKey: string,
): [number, number, number, number] => {
  const activeBars = STACK_ORDER.filter(
    (key) => (dataPoint[key as keyof PTOChartDataPoint] as number) > 0,
  );

  const currentIndex = activeBars.indexOf(dataKey);

  // If this bar type isn't active for this data point, no radius
  if (currentIndex === -1) {
    return [0, 0, 0, 0];
  }

  // Single bar - fully rounded
  if (activeBars.length === 1) {
    return [4, 4, 4, 4];
  }

  // First bar in stack - rounded bottom
  if (currentIndex === 0) {
    return [0, 0, 4, 4];
  }

  // Last bar in stack - rounded top
  if (currentIndex === activeBars.length - 1) {
    return [4, 4, 0, 0];
  }

  // Middle bar - no rounding
  return [0, 0, 0, 0];
};

interface CustomBarProps {
  fill?: string;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  payload?: PTOChartDataPoint;
  dataKey?: string;
}

const CustomBar = (props: CustomBarProps) => {
  const { fill, x, y, width, height, payload, dataKey } = props;

  if (!payload || !dataKey || !x || !y || !width || !height) {
    return null;
  }

  const radius = getBarRadius(payload, dataKey);

  return (
    <Rectangle
      x={x}
      y={y}
      width={width}
      height={height}
      fill={fill}
      radius={radius}
    />
  );
};

const ChartInner = memo(function ChartInner({
  data,
  onTickFormatter,
}: {
  data: PTOChartDataPoint[];
  onTickFormatter: (value: string) => string;
}) {
  return (
    <ChartContainer config={chartConfig} className="w-full h-[300px]">
      <BarChart data={data} accessibilityLayer>
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey="date"
          tickFormatter={onTickFormatter}
          tickLine={false}
          tickMargin={10}
          axisLine={false}
          className="text-xs fill-muted-foreground"
        />
        <YAxis
          tickLine={false}
          tickMargin={10}
          axisLine={false}
          className="text-xs fill-muted-foreground"
          allowDecimals={false}
        />
        <ChartTooltip content={<CustomTooltip />} />
        <ChartLegend content={<ChartLegendContent />} />

        <Bar
          dataKey="vacation"
          stackId="pto"
          name="Vacation"
          fill={PTO_COLORS.vacation}
          shape={(props: any) => <CustomBar {...props} dataKey="vacation" />}
        />
        <Bar
          dataKey="sick"
          stackId="pto"
          name="Sick"
          fill={PTO_COLORS.sick}
          shape={(props: any) => <CustomBar {...props} dataKey="sick" />}
        />
        <Bar
          dataKey="holiday"
          stackId="pto"
          name="Holiday"
          fill={PTO_COLORS.holiday}
          shape={(props: any) => <CustomBar {...props} dataKey="holiday" />}
        />
        <Bar
          dataKey="personal"
          stackId="pto"
          name="Personal"
          fill={PTO_COLORS.personal}
          shape={(props: any) => <CustomBar {...props} dataKey="personal" />}
        />
        <Bar
          dataKey="bereavement"
          stackId="pto"
          name="Bereavement"
          fill={PTO_COLORS.bereavement}
          shape={(props: any) => <CustomBar {...props} dataKey="bereavement" />}
        />
        <Bar
          dataKey="maternity"
          stackId="pto"
          name="Maternity"
          fill={PTO_COLORS.maternity}
          shape={(props: any) => <CustomBar {...props} dataKey="maternity" />}
        />
        <Bar
          dataKey="paternity"
          stackId="pto"
          name="Paternity"
          fill={PTO_COLORS.paternity}
          shape={(props: any) => <CustomBar {...props} dataKey="paternity" />}
        />
      </BarChart>
    </ChartContainer>
  );
});

export default function PTOChart({ startDate, endDate, type }: PTOChartProps) {
  const query = useQuery({
    ...queries.worker.getPTOChartData({
      startDate: startDate!,
      endDate: endDate!,
      type: type || undefined,
    }),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: Boolean(startDate && endDate),
  });

  const chartData = useMemo(() => query.data || [], [query.data]);

  const handleTickFormatter = useCallback(
    (value: string) => format(parseISO(value), "MMM dd"),
    [],
  );

  if (query.isLoading) {
    return <Skeleton className="h-[400px] w-full" />;
  }

  if (query.isError) {
    return (
      <div className="h-[400px] w-full flex items-center justify-center">
        <div className="text-center">
          <p className="text-sm text-destructive mb-2">
            Failed to load chart data
          </p>
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
        <div className="text-center">
          <p className="text-sm font-medium mb-1">No PTO data found</p>
          <p className="text-xs text-muted-foreground">
            Try adjusting the date range or filters
          </p>
        </div>
      </div>
    );
  }

  return <ChartInner data={chartData} onTickFormatter={handleTickFormatter} />;
}
