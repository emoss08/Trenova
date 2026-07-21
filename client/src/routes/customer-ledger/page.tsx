import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { Operation, Resource } from "@/types/permission";
import { useQuery } from "@tanstack/react-query";
import {
  BookOpenIcon,
  DownloadIcon,
  FileTextIcon,
  HandCoinsIcon,
  ReceiptTextIcon,
  UserSearchIcon,
} from "lucide-react";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { CustomerSnapshotHeader } from "./_components/customer-snapshot-header";
import { LedgerTable } from "./_components/ledger-table";

type FilterValues = {
  customerId: string;
};

export function CustomerLedgerPage() {
  const navigate = useNavigate();
  const { allowed: canRecordPayment } = usePermission(
    Resource.CustomerPayment,
    Operation.Create,
  );
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
        title: "Customer Ledger",
        description: "Statement-style transaction history with the customer's AR profile.",
        actions: customerId ? (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleExport} disabled={!entries?.length}>
              <DownloadIcon className="size-4" />
              Export
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                void navigate(`/accounting/ar/customer-statement/${customerId}`)
              }
            >
              <FileTextIcon className="size-4" />
              Statement
            </Button>
            {canRecordPayment ? (
              <Button
                size="sm"
                onClick={() =>
                  void navigate(
                    `/accounting/ar/payments?panelType=create&customerId=${customerId}`,
                  )
                }
              >
                <HandCoinsIcon className="size-4" />
                Record Payment
              </Button>
            ) : null}
          </div>
        ) : undefined,
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <div className="w-[300px]">
          <label className="mb-1 block text-2xs font-medium text-muted-foreground">
            Customer
          </label>
          <CustomerAutocompleteField
            control={filterForm.control}
            name="customerId"
            placeholder="Select a customer..."
            clearable
          />
        </div>

        {!customerId ? (
          <div className="flex justify-center pt-12">
            <EmptyState
              title="Select a customer"
              description="Choose a customer to see their AR profile, running ledger, and payment history."
              icons={[UserSearchIcon, BookOpenIcon, ReceiptTextIcon]}
            />
          </div>
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
              <div className="flex justify-center pt-8">
                <EmptyState
                  title="No ledger activity"
                  description="This customer has no posted AR transactions yet."
                  icons={[BookOpenIcon, FileTextIcon, ReceiptTextIcon]}
                />
              </div>
            ) : (
              <LedgerTable entries={entries} />
            )}
          </>
        )}
      </div>
    </PageLayout>
  );
}
