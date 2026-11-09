import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { usePermission } from "@/hooks/use-permission";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { ImportRemittanceDialog } from "@/routes/factoring/_components/import-remittance-dialog";
import type { FactoringCompanyRecord } from "@/lib/graphql/factoring";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryStates } from "nuqs";
import { lazy, useState } from "react";
import { useParams } from "react-router";
import {
  FactoringAgreementTabSkeleton,
  FactoringCompanyDetailsSkeleton,
  FactoringCompanyOverviewSkeleton,
  FactoringCompanyPageSkeleton,
  FactoringCompanyPanelsSkeleton,
} from "../_components/factoring-company-skeleton";
import { FactoringCompanyHeader } from "../_components/factoring-company-header";
import {
  factoringCompanyPageParsers,
  type FactoringCompanyTab,
} from "../use-factoring-company-state";

const Overview = lazy(() => import("../_components/factoring-company-overview"));
const Details = lazy(() => import("../_components/factoring-company-details"));
const AgreementTab = lazy(() => import("../_components/factoring-agreement-tab"));
const CustomersTab = lazy(() => import("../_components/factoring-customers-tab"));
const DeliveryTab = lazy(() => import("../_components/factoring-delivery-tab"));
const LedgerTab = lazy(() => import("../_components/factoring-ledger-tab"));
const AuditTab = lazy(() => import("@/components/audit-tab"));

export const prefetch: RoutePrefetch = ({ params }) =>
  params.id ? [factoringQueries.company(params.id), factoringQueries.overview(params.id)] : [];

/**
 * One factor as a page: where its invoices are and what needs someone, the details its invoices
 * print, its agreement, the customers assigned to it with their notices, how schedules reach it,
 * its ledger and its history.
 */
export function FactoringCompanyPage() {
  const t = useT();
  const { id = "" } = useParams<{ id: string }>();
  const company = useQuery({ ...factoringQueries.company(id), enabled: id !== "" });
  useBreadcrumbLabel(company.data?.name);

  if (company.isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Factoring company"),
          description: t("Loading the factoring company..."),
        }}
      >
        <FactoringCompanyPageSkeleton />
      </PageLayout>
    );
  }

  if (company.error || !company.data) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Factoring company"),
          description: t("This factoring company could not be loaded."),
        }}
      >
        <ErrorState
          layout="sheet"
          error={company.error}
          onRetry={company.error ? () => void company.refetch() : undefined}
          title={company.error ? undefined : t("No factoring company has this link")}
          description={
            company.error
              ? undefined
              : t("It may have been deleted, or it belongs to another organization.")
          }
        />
      </PageLayout>
    );
  }

  return <FactoringCompanyView company={company.data} />;
}

function FactoringCompanyView({ company }: { company: FactoringCompanyRecord }) {
  const t = useT();
  const [state, setState] = useQueryStates(factoringCompanyPageParsers);
  const overview = useQuery(factoringQueries.overview(company.id));
  const [importing, setImporting] = useState(false);
  const { allowed: canImport } = usePermission(Resource.FactoringRemittance, Operation.Create);
  const { allowed: canSubmit } = usePermission(Resource.FactoringSubmission, Operation.Submit);
  const { allowed: canReadAgreements } = usePermission(Resource.FactoringAgreement, Operation.Read);
  const { allowed: canReadCustomers } = usePermission(Resource.Customer, Operation.Read);
  const { allowed: canReadReserve } = usePermission(Resource.FactoringReserve, Operation.Read);
  const { allowed: canReadRemittances } = usePermission(
    Resource.FactoringRemittance,
    Operation.Read,
  );
  const { allowed: canReadAudit } = usePermission(Resource.AuditLog, Operation.Read);

  const visible: Record<FactoringCompanyTab, boolean> = {
    overview: true,
    details: true,
    agreement: canReadAgreements,
    customers: canReadCustomers,
    delivery: true,
    ledger: canReadReserve || canReadRemittances,
    activity: canReadAudit,
  };
  const tab = visible[state.tab] ? state.tab : "overview";
  const selectTab = (next: FactoringCompanyTab) =>
    void setState({ tab: next, agreement: null, noa: null });

  const data = overview.data;
  const agreements = useQuery({
    ...factoringQueries.agreements(company.id),
    enabled: canReadAgreements,
  });
  const inForce = data?.agreement ?? null;
  const agreementNumber =
    inForce && agreements.data
      ? [...agreements.data]
          .sort((a, b) => a.effectiveFrom - b.effectiveFrom || a.createdAt - b.createdAt)
          .findIndex((agreement) => agreement.id === inForce.id) + 1 || null
      : null;

  return (
    <PageLayout
      pageHeaderProps={{
        title: company.name,
        description: t(
          "Where this factor's invoices are, the terms it buys them on, the customers assigned to it, how schedules reach it, and what its invoices print.",
        ),
      }}
      bleed
    >
      <FactoringCompanyHeader
        company={company}
        overview={data}
        overviewUpdatedAt={overview.dataUpdatedAt}
        overviewFetching={overview.isFetching}
        onRefresh={() => void overview.refetch()}
        agreementNumber={agreementNumber}
        tab={tab}
        visible={visible}
        onTab={selectTab}
        canImport={canImport}
        canSubmit={canSubmit}
        onImport={() => setImporting(true)}
      />
      <div className="flex min-w-0 flex-1 flex-col gap-4 px-6 pt-5 pb-7">
        {tab === "overview" ? (
          <LazyComponent fallback={<FactoringCompanyOverviewSkeleton />}>
            <Overview factoringCompanyId={company.id} onTab={selectTab} />
          </LazyComponent>
        ) : tab === "details" ? (
          <LazyComponent fallback={<FactoringCompanyDetailsSkeleton />}>
            <Details
              company={company}
              recourseFromAgreement={Boolean(data?.agreement)}
              onTab={selectTab}
            />
          </LazyComponent>
        ) : tab === "agreement" ? (
          <LazyComponent fallback={<FactoringAgreementTabSkeleton />}>
            <AgreementTab factoringCompanyId={company.id} highlightedId={state.agreement} />
          </LazyComponent>
        ) : tab === "customers" ? (
          <DataTableLazyComponent>
            <CustomersTab factoringCompanyId={company.id} highlightedNoaId={state.noa} />
          </DataTableLazyComponent>
        ) : tab === "delivery" ? (
          <LazyComponent
            fallback={<FactoringCompanyPanelsSkeleton label={t("Loading the delivery settings")} />}
          >
            <DeliveryTab factoringCompanyId={company.id} />
          </LazyComponent>
        ) : tab === "ledger" ? (
          <LazyComponent
            fallback={<FactoringCompanyPanelsSkeleton label={t("Loading the ledger")} />}
          >
            <LedgerTab factoringCompanyId={company.id} />
          </LazyComponent>
        ) : (
          <LazyComponent
            fallback={<FactoringCompanyPanelsSkeleton label={t("Loading the history")} />}
          >
            <AuditTab resourceId={company.id} />
          </LazyComponent>
        )}
      </div>
      <ImportRemittanceDialog
        open={importing}
        onOpenChange={setImporting}
        factoringCompanyId={company.id}
      />
    </PageLayout>
  );
}
