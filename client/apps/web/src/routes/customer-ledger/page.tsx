import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { DownloadIcon, FileTextIcon, HandCoinsIcon } from "lucide-react";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { CustomerSnapshotHeader } from "./_components/customer-snapshot-header";
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
  const [searchParams, setSearchParams] = useSearchParams();
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

  const { data: entries, isLoading: ledgerLoading } = useQuery({
    ...queries.ar.customerLedger(customerId),
    enabled: Boolean(customerId),
  });

  const { data: profile, isLoading: profileLoading } = useQuery({
    ...queries.ar.customerProfile(customerId),
    enabled: Boolean(customerId),
  });

  const handleExport = () => {
    if (!entries?.length) return;
    let runningBalance = 0;
    const rows = [
      ["Date", "Document", "Event", "Source Type", "Amount", "Balance"],
      ...entries.map((entry) => {
        runningBalance += entry.amountMinor;
        return [
          new Date(entry.transactionDate * 1000).toISOString().slice(0, 10),
          entry.documentNumber,
          entry.eventType,
          entry.sourceObjectType,
          (entry.amountMinor / 100).toFixed(2),
          (runningBalance / 100).toFixed(2),
        ];
      }),
    ];
    const csv = rows.map((row) => row.map((cell) => `"${cell}"`).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `customer-ledger-${customerId}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Customer Ledger"),
        description: t("Statement-style transaction history with the customer's AR profile."),
        actions: customerId ? (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleExport} disabled={!entries?.length}>
              <DownloadIcon className="size-4" />
              {t("Export")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void navigate(`/accounting/ar/customer-statement/${customerId}`)}
            >
              <FileTextIcon className="size-4" />
              {t("Statement")}
            </Button>
            {canRecordPayment ? (
              <Button
                size="sm"
                onClick={() =>
                  void navigate(`/accounting/ar/payments?panelType=create&customerId=${customerId}`)
                }
              >
                <HandCoinsIcon className="size-4" />
                {t("Record Payment")}
              </Button>
            ) : null}
          </div>
        ) : undefined,
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
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
              <LedgerTable entries={entries} />
            )}
          </>
        )}
      </div>
    </PageLayout>
  );
}
