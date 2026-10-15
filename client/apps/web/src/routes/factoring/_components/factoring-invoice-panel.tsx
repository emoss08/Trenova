import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FactoringVerdictBadge } from "@/components/factoring/factoring-badges";
import { FactoringSubmissionLink } from "@/components/factoring/factoring-submission-link";
import { FactoringChecksList } from "@/components/factoring/factoring-checks-list";
import { SectionPanel } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import type { FactoringInvoiceState, FactoringInvoiceStateRow } from "@/lib/graphql/factoring";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useQueryStates } from "nuqs";
import { Link } from "react-router";

/**
 * One invoice's readiness: what it is worth, who it is assigned to, and each
 * check with the action that clears it. Opened from a row, or from a link
 * naming an invoice that is not on the current page.
 */
export function FactoringInvoicePanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<FactoringInvoiceStateRow>) {
  const t = useT();
  const [{ panelEntityId }] = useQueryStates(panelSearchParamsParser);
  const invoiceId = row?.invoiceId ?? panelEntityId;
  const fetched = useQuery({
    ...factoringQueries.invoice(invoiceId ?? ""),
    enabled: open && !!invoiceId,
  });
  const state: FactoringInvoiceState | null = fetched.data?.factoringState ?? row ?? null;

  if (!state) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={t("Factoring readiness")}
        size="lg"
      >
        {fetched.isError || (fetched.isSuccess && !fetched.data?.factoringState) ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>
              {t("This invoice's factoring readiness could not be loaded.")}
            </AlertDescription>
          </Alert>
        ) : (
          <ComponentLoader message={t("Loading factoring readiness...")} />
        )}
      </DataTablePanelContainer>
    );
  }

  const number = state.invoice?.number ?? fetched.data?.number ?? "";

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={t("Invoice {0}", number)}
      description={t("Checked {0}", formatUnixDateTimeShort(state.evaluatedAt))}
      size="lg"
      footer={
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Close")}
          </Button>
          <Button type="button" render={<Link to={recordPath("invoice", state.invoiceId)} />}>
            {t("Open invoice")}
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <DescriptionList columns={2}>
          <DescriptionItem label={t("Verdict")}>
            <FactoringVerdictBadge verdict={state.verdict} />
          </DescriptionItem>
          <DescriptionItem label={t("Open balance")} numeric>
            {formatMinor(state.openBalanceMinor, state.invoice?.currencyCode ?? "USD")}
          </DescriptionItem>
          <DescriptionItem label={t("Customer")}>
            {state.customer ? (
              <Link
                to={recordPath("customer", state.customer.id)}
                className="text-brand hover:underline"
              >
                {state.customer.name}
              </Link>
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Factor")}>
            {state.factoringCompany ? (
              <Link
                to={recordPath("factoring_company", state.factoringCompany.id)}
                className="text-brand hover:underline"
              >
                {state.factoringCompany.name}
              </Link>
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Ready since")}>
            {state.readySince ? formatUnixDateTimeShort(state.readySince) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Failed checks")} numeric>
            {state.failedChecks}
          </DescriptionItem>
          {state.submission ? (
            <DescriptionItem label={t("Schedule")}>
              <FactoringSubmissionLink submission={state.submission} />
            </DescriptionItem>
          ) : null}
        </DescriptionList>
        <SectionPanel
          title={t("Checks")}
          count={state.failedChecks}
          help={t(
            "Every check an invoice must pass before it can be sold to the factor. A failed check names what clears it.",
          )}
        >
          <FactoringChecksList state={state} invoiceNumber={number} />
        </SectionPanel>
      </div>
    </DataTablePanelContainer>
  );
}
