import { PageLayout } from "@/components/navigation/sidebar-layout";
import { managedPlan } from "@/lib/queries/managed-plan";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ArrowUpRightIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { PlanSections } from "./_components/plan-sections";
import { PlanSkeleton } from "./_components/plan-skeleton";

export const prefetch: RoutePrefetch = () => [managedPlan.current()];

/** Opens the plan's page in the customer's Trenova account, where add-ons are bought. */
function ManagePlanLink({ href }: { href: string }) {
  const t = useT();

  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={buttonVariants({ variant: "default", size: "sm" })}
    >
      {t("Manage plan")}
      <ArrowUpRightIcon className="size-3.5" aria-hidden />
    </a>
  );
}

/**
 * What a Managed Trenova plan includes, how much of it is used this period, the add-ons it does
 * not, and when one joined, left or was set to end. Read-only: the plan is changed from the
 * customer's Trenova account, which the page links to.
 */
export function PlanPage() {
  const t = useT();
  const query = useQuery(managedPlan.current());
  const manageUrl = query.data?.managed ? query.data.manageUrl : null;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Your plan"),
        description: t(
          "What your plan includes, how much of it you have used and what you can add",
        ),
        actions: manageUrl ? <ManagePlanLink href={manageUrl} /> : undefined,
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
