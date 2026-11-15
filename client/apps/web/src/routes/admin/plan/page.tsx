import { PageLayout } from "@/components/navigation/sidebar-layout";
import { managedPlan } from "@/lib/queries/managed-plan";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { PlanSections } from "./_components/plan-sections";
import { PlanSkeleton } from "./_components/plan-skeleton";

export const prefetch: RoutePrefetch = () => [managedPlan.current()];

/**
 * What a Managed Trenova plan includes, the add-ons it does not, and when one joined or left.
 * Read-only: the plan is changed from the customer's Trenova account, never here.
 */
export function PlanPage() {
  const t = useT();
  const query = useQuery(managedPlan.current());

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Your plan"),
        description: t("What your plan includes, the add-ons it does not, and recent changes"),
      }}
    >
      {query.isPending ? (
        <PlanSkeleton />
      ) : query.isError ? (
        <ErrorState
          error={query.error}
          onRetry={() => void query.refetch()}
          layout="sheet"
          title={t("Your plan could not be loaded")}
        />
      ) : !query.data.managed ? (
        <Alert variant="info" size="sm">
          <AlertDescription>
            {t("This installation is not on a Managed Trenova plan, so every feature is on.")}
          </AlertDescription>
        </Alert>
      ) : (
        <PlanSections plan={query.data} />
      )}
    </PageLayout>
  );
}
