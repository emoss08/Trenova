/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { lazy, Suspense } from "react";

// Lazy load all recharts components
export const LazyAreaChart = lazy(() =>
  import("recharts").then((module) => ({ default: module.AreaChart })),
);

export const LazyArea = lazy(() =>
  import("recharts").then((module) => ({ default: module.Area })),
);

export const LazyRadialBarChart = lazy(() =>
  import("recharts").then((module) => ({ default: module.RadialBarChart })),
);

export const LazyRadialBar = lazy(() =>
  import("recharts").then((module) => ({ default: module.RadialBar })),
);

export const LazyPolarAngleAxis = lazy(() =>
  import("recharts").then((module) => ({ default: module.PolarAngleAxis })),
);

export const LazyResponsiveContainer = lazy(() =>
  import("recharts").then((module) => ({
    default: module.ResponsiveContainer,
  })),
);

export const LazyTooltip = lazy(() =>
  import("recharts").then((module) => ({ default: module.Tooltip })),
);

export const LazyBarChart = lazy(() =>
  import("recharts").then((module) => ({ default: module.BarChart })),
);

export const LazyBar = lazy(() =>
  import("recharts").then((module) => ({ default: module.Bar })),
);

export const LazyXAxis = lazy(() =>
  import("recharts").then((module) => ({ default: module.XAxis })),
);

export const LazyYAxis = lazy(() =>
  import("recharts").then((module) => ({ default: module.YAxis })),
);

export const LazyLegend = lazy(() =>
  import("recharts").then((module) => ({ default: module.Legend })),
);

export const LazyCartesianGrid = lazy(() =>
  import("recharts").then((module) => ({ default: module.CartesianGrid })),
);

// Chart wrapper component
export function ChartWrapper({ children }: { children: React.ReactNode }) {
  return (
    <Suspense
      fallback={
        <div className="h-[200px] w-full animate-pulse bg-muted rounded-md" />
      }
    >
      {children}
    </Suspense>
  );
}
