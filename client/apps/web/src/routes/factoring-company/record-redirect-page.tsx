import { PageLayout } from "@/components/navigation/sidebar-layout";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { Navigate, useParams } from "react-router";
import { FactoringCompanyPageSkeleton } from "./_components/factoring-company-skeleton";
import { factoringCompanyPath, type FactoringCompanyTab } from "./use-factoring-company-state";

/**
 * Where a link to an agreement or a notice lands: both live on their factor's page, and the link
 * names only the record, so this finds the factor and replaces itself with that page, on the tab
 * that holds the record and with the record picked out.
 */
function OwnerRedirect({
  owner,
  tab,
  open,
}: {
  owner: { data?: string; isLoading: boolean; error: unknown; refetch: () => unknown };
  tab: FactoringCompanyTab;
  open: { agreement?: string; noa?: string };
}) {
  const t = useT();

  if (owner.data) {
    return <Navigate replace to={factoringCompanyPath(owner.data, { tab, ...open })} />;
  }

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Factoring company"),
        description: t("Finding the factor this record belongs to."),
      }}
    >
      {owner.isLoading ? (
        <FactoringCompanyPageSkeleton />
      ) : (
        <ErrorState
          layout="sheet"
          error={owner.error}
          onRetry={() => void owner.refetch()}
          title={t("This record could not be opened")}
          description={t(
            "It may have been deleted, or it belongs to another organization. Its factor's page lists what is left.",
          )}
        />
      )}
    </PageLayout>
  );
}

export function FactoringAgreementRedirectPage() {
  const { id = "" } = useParams<{ id: string }>();
  const owner = useQuery({ ...factoringQueries.agreementOwner(id), enabled: id !== "" });
  return <OwnerRedirect owner={owner} tab="agreement" open={{ agreement: id }} />;
}

export function FactoringNoticeRedirectPage() {
  const { id = "" } = useParams<{ id: string }>();
  const owner = useQuery({ ...factoringQueries.noaOwner(id), enabled: id !== "" });
  return <OwnerRedirect owner={owner} tab="customers" open={{ noa: id }} />;
}
