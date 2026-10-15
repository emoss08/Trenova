import { PageLayout } from "@/components/navigation/sidebar-layout";
import { tablePaginationSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Tabs, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryStates } from "nuqs";
import { lazy } from "react";
import { FactoringSummaryStrip } from "./_components/factoring-summary";
import { factoringSearchParamsParser, type FactoringTab } from "./use-factoring-state";

const Table = lazy(() => import("./_components/factoring-invoice-table"));
const SubmissionTable = lazy(() => import("./_components/factoring-submission-table"));

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
      <FactoringSummaryStrip active={state.view} onSelect={selectTab} />
      <Tabs value={state.view} onValueChange={(value) => selectTab(value as FactoringTab)}>
        <div className="border-border border-b">
          <TabsList variant="underline">
            <TabsTab value="ready">{t("Ready")}</TabsTab>
            <TabsTab value="not-ready">{t("Not ready")}</TabsTab>
            <TabsTab value="submitted">{t("Submitted")}</TabsTab>
            <TabsTab value="funded">{t("Funded")}</TabsTab>
            <TabsTab value="schedules">{t("Schedules")}</TabsTab>
          </TabsList>
        </div>
      </Tabs>
      <DataTableLazyComponent>
        {state.view === "schedules" ? (
          <SubmissionTable />
        ) : (
          <Table key={state.view} tab={state.view} />
        )}
      </DataTableLazyComponent>
    </PageLayout>
  );
}
