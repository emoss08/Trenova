import { useQueryParams } from "@trenova/shared/hooks/use-query-params";
import { useT } from "@trenova/shared/i18n/use-t";
import { customerStatementPath } from "@/lib/ar-links";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { ar, customerLedgerPages } from "@/lib/queries/ar";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CoinsHandIcon, Download01Icon, File06Icon } from "@trenova/shared/components/icons";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useNavigate } from "@tanstack/react-router";
import { MemoEntryButton } from "@/components/billing/memo-entry-button";
import { CustomerSnapshotHeader } from "./_components/customer-snapshot-header";
import { LedgerExportDialog } from "./_components/ledger-export-dialog";
import { LedgerTable } from "./_components/ledger-table";

type FilterValues = {
  customerId: string;
};

const LEDGER_COLUMNS = [
  { label: "Date" },
  { label: "Document" },
  { label: "Event" },
  { label: "Source" },
  { label: "Debit", numeric: true },
  { label: "Credit", numeric: true },
  { label: "Balance", numeric: true },
] as const;

export function CustomerLedgerPage() {
  const t = useT();

  const navigate = useNavigate();
  const { allowed: canRecordPayment } = usePermission(Resource.CustomerPayment, Operation.Create);
  const { allowed: canExport } = usePermission(Resource.AccountsReceivable, Operation.Export);
  const [exportOpen, setExportOpen] = useState(false);
  const [searchParams, setSearchParams] = useQueryParams();
  const initialCustomerId = searchParams.get("customerId") ?? "";

  const filterForm = useForm<FilterValues>({
    defaultValues: { customerId: initialCustomerId },
  });
  const customerId = useWatch({ control: filterForm.control, name: "customerId" });

  useEffect(() => {
    const current = searchParams.get("customerId") ?? "";
    if ((customerId || "") !== current) {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (customerId) {
            next.set("customerId", customerId);
          } else {
            next.delete("customerId");
          }
          return next;
        },
        { replace: true },
      );
    }
  }, [customerId, searchParams, setSearchParams]);

  const ledger = useInfiniteQuery({
    ...customerLedgerPages(customerId),
    enabled: Boolean(customerId),
  });
  const ledgerLoading = ledger.isLoading;
  const entries = ledger.data?.pages.flatMap((page) => page.entries);

  const { data: profile, isLoading: profileLoading } = useQuery({
    ...ar.customerProfile(customerId),
    enabled: Boolean(customerId),
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Customer ledger"),
        description: t("Statement-style transaction history with the customer's AR profile."),
        actions: customerId ? (
          <div className="flex items-center gap-2">
            {canExport ? (
              <Button variant="outline" size="sm" onClick={() => setExportOpen(true)}>
                <Download01Icon className="size-4" />
                {t("Export")}
              </Button>
            ) : null}
            <Button
              variant="outline"
              size="sm"
              onClick={() => void navigate({ to: customerStatementPath(customerId) })}
            >
              <File06Icon className="size-4" />
              {t("Statement")}
            </Button>
            <MemoEntryButton customerId={customerId} billType="CreditMemo" />
            <MemoEntryButton customerId={customerId} billType="DebitMemo" />
            {canRecordPayment ? (
              <Button
                size="sm"
                onClick={() =>
                  void navigate({
                    to: `/accounting/ar/payments?panelType=create&customerId=${customerId}`,
                  })
                }
              >
                <CoinsHandIcon className="size-4" />
                {t("Record payment")}
              </Button>
            ) : null}
          </div>
        ) : undefined,
      }}
    >
      <>
        <div className="w-75">
          <label className="text-2xs text-muted-foreground mb-1 block font-medium">
            {t("Customer")}
          </label>
          <CustomerAutocompleteField
            control={filterForm.control}
            name="customerId"
            placeholder={t("Select a customer...")}
            clearable
          />
        </div>

        {!customerId ? (
          <EmptyTable
            title={t("Pick a customer")}
            description={t(
              "Choose a customer above and their receivables profile, running ledger and payment history are laid out here.",
            )}
            columns={LEDGER_COLUMNS}
          />
        ) : (
          <>
            <CustomerSnapshotHeader profile={profile} isLoading={profileLoading} />

            {ledgerLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} className="h-10 w-full" />
                ))}
              </div>
            ) : !entries || entries.length === 0 ? (
              <EmptyTable
                title={t("No activity yet")}
                description={t(
                  "Nothing has posted to this customer's receivables. Their first invoice or payment starts the ledger.",
                )}
                columns={LEDGER_COLUMNS}
              />
            ) : (
              <LedgerTable
                entries={entries}
                hasNextPage={ledger.hasNextPage}
                isFetchingNextPage={ledger.isFetchingNextPage}
                onLoadMore={() => void ledger.fetchNextPage()}
              />
            )}
            <LedgerExportDialog
              open={exportOpen}
              onOpenChange={setExportOpen}
              customerId={customerId}
              customerName={profile?.snapshot.customerName}
            />
          </>
        )}
      </>
    </PageLayout>
  );
}
