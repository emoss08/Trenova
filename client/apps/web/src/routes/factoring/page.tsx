import { PageLayout } from "@/components/navigation/sidebar-layout";
import { recordPath } from "@/config/record-links";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { tablePaginationSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Tabs, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryStates } from "nuqs";
import { lazy } from "react";
import { Link } from "react-router";
import { FactoringSummaryStrip } from "./_components/factoring-summary";
import { factoringSearchParamsParser, type FactoringTab } from "./use-factoring-state";

const Table = lazy(() => import("./_components/factoring-invoice-table"));
const SubmissionTable = lazy(() => import("./_components/factoring-submission-table"));
const RemittanceTable = lazy(() => import("./_components/factoring-remittance-table"));

export function FactoringPage() {
  const t = useT();
  const [state, setState] = useQueryStates({
    ...factoringSearchParamsParser,
    pageIndex: tablePaginationSearchParamsParser.pageIndex,
  });

  const selectTab = (view: FactoringTab) => {
    void setState({ view, pageIndex: 1 });
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Factoring"),
        description: t(
          "Posted invoices assigned to a factor, split by whether they are ready to sell. Select ready invoices to send them to their factor on a schedule, open one to see the checks it failed, or follow funded invoices until the factor releases their reserve.",
        ),
      }}
    >
      {state.factor ? (
        <FactorNarrowing
          factoringCompanyId={state.factor}
          onClear={() => void setState({ factor: null, pageIndex: 1 })}
        />
      ) : (
        <FactoringSummaryStrip active={state.view} onSelect={selectTab} />
      )}
      <Tabs value={state.view} onValueChange={(value) => selectTab(value as FactoringTab)}>
        <div className="border-border border-b">
          <TabsList variant="underline">
            <TabsTab value="ready">{t("Ready")}</TabsTab>
            <TabsTab value="not-ready">{t("Not ready")}</TabsTab>
            <TabsTab value="submitted">{t("Submitted")}</TabsTab>
            <TabsTab value="funded">{t("Funded")}</TabsTab>
            <TabsTab value="schedules">{t("Schedules")}</TabsTab>
            <TabsTab value="remittances">{t("Remittances")}</TabsTab>
          </TabsList>
        </div>
      </Tabs>
      <DataTableLazyComponent>
        {state.view === "schedules" ? (
          <SubmissionTable factoringCompanyId={state.factor} />
        ) : state.view === "remittances" ? (
          <RemittanceTable factoringCompanyId={state.factor} />
        ) : (
          <Table key={state.view} tab={state.view} factoringCompanyId={state.factor} />
        )}
      </DataTableLazyComponent>
    </PageLayout>
  );
}

/**
 * The workspace narrowed to one factor, as its page opens it: the lists show only that factor's
 * invoices, schedules and remittances, and the organization's figures step aside so they are not
 * read as the factor's.
 */
function FactorNarrowing({
  factoringCompanyId,
  onClear,
}: {
  factoringCompanyId: string;
  onClear: () => void;
}) {
  const t = useT();
  const { data } = useQuery(factoringQueries.company(factoringCompanyId));
  const name = data?.name ?? t("one factor");

  return (
    <Alert size="sm">
      <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
        <span>
          {t("Showing only {0}.", name)}{" "}
          <Link
            to={recordPath("factoring_company", factoringCompanyId)}
            className="ui-focus-ring rounded-sm underline underline-offset-4"
          >
            {t("Open its page")}
          </Link>
        </span>
        <Button type="button" size="sm" variant="outline" onClick={onClear}>
          <XCloseIcon className="size-3.5" />
          {t("Show every factor")}
        </Button>
      </AlertDescription>
    </Alert>
  );
}
