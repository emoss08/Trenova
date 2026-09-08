import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { BenefitsSkeleton } from "./_components/benefits-skeleton";
import {
  benefitPlansQuery,
  declinedEnrollmentsQuery,
  openEnrollmentsQuery,
} from "./_components/queries";

const BenefitsConsole = lazy(() => import("./_components/benefits-console"));

// The plans decide which year the page opens on, so the year's cost rows are
// left to the console; everything the page can know in advance is warmed.
export const prefetch: RoutePrefetch = () => {
  const list: RoutePrefetchQuery[] = [
    benefitPlansQuery(),
    openEnrollmentsQuery(),
    declinedEnrollmentsQuery(),
  ];
  return list;
};

export function BenefitsPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Benefits",
        description:
          "What the carrier offers and who is on it. An employee contribution is taken through an ordinary settlement deduction, so it caps, pauses and reverses exactly like every other one.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<BenefitsSkeleton />}>
          <BenefitsConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
