import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { randomDrawsQuery, randomPoolsQuery } from "./_components/queries";
import { RandomTestingSkeleton } from "./_components/random-testing-skeleton";

const RandomTestingConsole = lazy(() => import("./_components/random-testing-console"));

// The page reads every pool and every round on mount and narrows on the
// client, so both are warmed before the console renders.
export const prefetch: RoutePrefetch = () => [randomPoolsQuery(), randomDrawsQuery()];

export function RandomTestingPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Random Testing",
        description:
          "The pools drivers are drawn from and the rounds drawn from them. Each round keeps the seed it was drawn with, so a selection can be re-checked years later.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<RandomTestingSkeleton />}>
          <RandomTestingConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
