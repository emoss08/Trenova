import { DataTable } from "@/components/data-table/data-table";
import {
  AcknowledgeFactoringNoaDialog,
  ReleaseFactoringNoaDialog,
  SendFactoringNoaDialog,
} from "@/components/factoring/factoring-noa-dialogs";
import { usePermission } from "@/hooks/use-permission";
import type { FactoringNoa } from "@/lib/graphql/factoring";
import {
  createFactoringCompanyCustomersTableConfig,
  FACTORING_COMPANY_CUSTOMER_KEY,
  type FactoringCompanyCustomer,
} from "@/lib/graphql/factoring-company-page";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import type { FactoringCustomerNoticeStage } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, PlusIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { dateToUnixTimestamp } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { getFactoringCustomerColumns } from "./factoring-customers-columns";
import { DraftNoticeDialog } from "./factoring-draft-notice-dialog";

type StageFilter = "all" | FactoringCustomerNoticeStage;

/**
 * The customers whose billing profiles assign their invoices to this factor, as a table: each with
 * where its notice of assignment stands and the one step that moves it on, searchable, filterable
 * and sortable on the customer's own fields and narrowed to a notice stage from the toolbar.
 * Customers are assigned in their billing profiles; each name opens the customer.
 */
export default function FactoringCustomersTab({
  factoringCompanyId,
  highlightedNoaId,
}: {
  factoringCompanyId: string;
  highlightedNoaId: string | null;
}) {
  const t = useT();
  const [stage, setStage] = useState<StageFilter>("all");
  const [nowUnix] = useState(() => dateToUnixTimestamp(new Date()));
  const [drafting, setDrafting] = useState<{ customerId: string | null } | null>(null);
  const [sending, setSending] = useState<FactoringNoa | null>(null);
  const [acknowledging, setAcknowledging] = useState<FactoringNoa | null>(null);
  const [releasing, setReleasing] = useState<FactoringNoa | null>(null);
  const { allowed: canCreate } = usePermission(Resource.FactoringNoa, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.FactoringNoa, Operation.Update);
  const { allowed: canReadMoney } = usePermission(Resource.FactoringReserve, Operation.Read);
  const overview = useQuery(factoringQueries.overview(factoringCompanyId));
  const counts = overview.data?.notices;

  const graphql = useMemo(
    () =>
      createFactoringCompanyCustomersTableConfig(
        factoringCompanyId,
        stage === "all" ? null : stage,
      ),
    [factoringCompanyId, stage],
  );
  const columns = useMemo(
    () =>
      getFactoringCustomerColumns(t, {
        nowUnix,
        canReadMoney,
        actions: {
          canCreate,
          canUpdate,
          onDraft: (customerId) => setDrafting({ customerId }),
          onSend: setSending,
          onAcknowledge: setAcknowledging,
          onRelease: setReleasing,
        },
      }),
    [t, nowUnix, canReadMoney, canCreate, canUpdate],
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      {counts && counts.needsNotice > 0 ? (
        <Alert size="sm" variant="warning">
          <AlertTriangleIcon />
          <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
            <span>
              {t(
                "{0, plural, one {# customer is} other {# customers are}} assigned to this factor without a notice in force. Their invoices print the factor's remit-to, but payments may still reach you.",
                counts.needsNotice,
              )}
            </span>
            {stage !== "NeedsNotice" ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setStage("NeedsNotice")}
              >
                {t("Show them")}
              </Button>
            ) : null}
          </AlertDescription>
        </Alert>
      ) : null}

      <DataTable<FactoringCompanyCustomer>
        name="Factoring Company Customer"
        queryKey={FACTORING_COMPANY_CUSTOMER_KEY}
        graphql={graphql}
        resource={Resource.Customer}
        columns={columns}
        enableCreateAction={false}
        initialColumnVisibility={{ code: false }}
        getRowClassName={(row) =>
          row.original.notice && row.original.notice.id === highlightedNoaId
            ? "bg-brand-subtle"
            : ""
        }
        emptyTitle={
          stage === "all"
            ? t(
                "No customer is assigned to this factor. Assign one in its billing profile, then send it a notice.",
              )
            : t("No customer is at this stage.")
        }
        toolbar={{
          trailing: (
            <SegmentedControl<StageFilter>
              aria-label={t("Notice stage")}
              value={stage}
              onValueChange={setStage}
              items={[
                { value: "all", label: t("All {0}", counts?.assigned ?? 0) },
                {
                  value: "NeedsNotice",
                  label: t("Needs a notice {0}", counts?.needsNotice ?? 0),
                },
                { value: "Awaiting", label: t("Drafted or sent {0}", counts?.awaiting ?? 0) },
                {
                  value: "Acknowledged",
                  label: t("Acknowledged {0}", counts?.acknowledged ?? 0),
                },
              ]}
            />
          ),
          end: canCreate ? (
            <Button type="button" size="sm" onClick={() => setDrafting({ customerId: null })}>
              <PlusIcon className="size-3.5" />
              {t("New notice")}
            </Button>
          ) : undefined,
        }}
      />
      <p className="text-muted-foreground text-xs">
        {t("Customers are assigned to a factor in their billing profile. Each name opens it.")}
      </p>

      <DraftNoticeDialog
        factoringCompanyId={factoringCompanyId}
        customerId={drafting?.customerId ?? null}
        open={drafting !== null}
        onOpenChange={(next) => {
          if (!next) setDrafting(null);
        }}
        onDrafted={(noa) => {
          if (canUpdate) setSending(noa);
        }}
      />
      <SendFactoringNoaDialog
        noa={sending}
        onOpenChange={(next) => {
          if (!next) setSending(null);
        }}
      />
      <AcknowledgeFactoringNoaDialog
        noa={acknowledging}
        onOpenChange={(next) => {
          if (!next) setAcknowledging(null);
        }}
      />
      <ReleaseFactoringNoaDialog
        noa={releasing}
        onOpenChange={(next) => {
          if (!next) setReleasing(null);
        }}
      />
    </div>
  );
}
