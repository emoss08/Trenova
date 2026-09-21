import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  assessLateCharges,
  type LateChargeAssessmentResult,
} from "@/lib/graphql/accounts-receivable";
import { queries } from "@/lib/queries";
import { invalidateInvoiceQueries } from "@/lib/queries/invoice";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import type { LateChargeAssessmentMode } from "@/types/billing-control";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { getEndOfDay } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AlertTriangleIcon, PlayIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";
import { LateChargePreviewTable } from "./_components/late-charge-preview-table";

type FilterValues = {
  customerId: string;
  asOfDate: number | null;
};

export const prefetch: RoutePrefetch = () => {
  const list: RoutePrefetchQuery[] = [queries.ar.lateChargePreview(), queries.billingControl.get()];
  return list;
};

function modeNotice(mode: LateChargeAssessmentMode | undefined, t: ReturnType<typeof useT>) {
  switch (mode) {
    case "Automatic":
      return {
        tone: "info" as const,
        text: t(
          "The nightly run raises and posts these debit memos automatically; running now assesses them ahead of it.",
        ),
      };
    case "Preview":
      return {
        tone: "warning" as const,
        text: t(
          "The nightly run only previews: it computes what it would raise and writes nothing. Run it here to raise the memos, or switch the mode to Automatic in billing control.",
        ),
      };
    default:
      return {
        tone: "warning" as const,
        text: t(
          "Late charge assessment is disabled for this organization. You can still preview here; enable it in billing control to raise memos.",
        ),
      };
  }
}

/**
 * What the late-charge run would raise as of a date, and a way to run it now
 * for chosen customers. Previewing never writes; assessing raises one debit
 * memo per customer and shows the memo numbers in place of the preview.
 */
export function LateChargesPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canAssess } = usePermission(Resource.Invoice, Operation.Create);
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [confirming, setConfirming] = useState(false);
  const [lastRun, setLastRun] = useState<LateChargeAssessmentResult | null>(null);

  const filterForm = useForm<FilterValues>({ defaultValues: { customerId: "", asOfDate: null } });
  const customerId = useWatch({ control: filterForm.control, name: "customerId" });
  const asOfValue = useWatch({ control: filterForm.control, name: "asOfDate" });
  const asOfUnix = useMemo(
    () => (asOfValue ? getEndOfDay(new Date(asOfValue * 1000)) : null),
    [asOfValue],
  );
  // No filters means the same key the route prefetch warmed.
  const input = useMemo(
    () =>
      customerId || asOfUnix
        ? { customerIds: customerId ? [customerId] : null, asOfDate: asOfUnix }
        : undefined,
    [customerId, asOfUnix],
  );

  const previewQuery = useQuery(queries.ar.lateChargePreview(input));
  const controlQuery = useQuery(queries.billingControl.get());
  const mode = controlQuery.data?.lateChargeAssessmentMode;
  const notice = modeNotice(mode, t);
  const result = lastRun ?? previewQuery.data;

  const selectedIds = useMemo(
    () =>
      (result?.customers ?? [])
        .filter((row) => !row.skipped && selected[row.customerId])
        .map((row) => row.customerId),
    [result, selected],
  );
  const selectedTotalMinor = useMemo(
    () =>
      (result?.customers ?? [])
        .filter((row) => selectedIds.includes(row.customerId))
        .reduce((sum, row) => sum + row.totalChargeMinor, 0),
    [result, selectedIds],
  );

  const assess = useApiMutation({
    resourceName: "late charges",
    mutationFn: () => assessLateCharges({ customerIds: selectedIds, asOfDate: asOfUnix }),
    onSuccess: (run) => {
      setLastRun(run);
      setSelected({});
      setConfirming(false);
      invalidateInvoiceQueries(queryClient);
      void queryClient.invalidateQueries({ queryKey: queries.ar.lateChargePreview._def });
      toast.success(
        t(
          "{0, plural, one {# debit memo} other {# debit memos}} raised for {1}",
          run.memosCreated,
          formatCurrency(run.totalChargeMinor / 100),
        ),
      );
    },
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Late charges"),
        description: t(
          "Overdue invoices charged once per thirty-day period at each customer's rate, raised as one debit memo per customer.",
        ),
        actions: canAssess ? (
          <Button
            size="sm"
            disabled={selectedIds.length === 0 || mode === "Disabled" || assess.isPending}
            onClick={() => setConfirming(true)}
          >
            <PlayIcon className="size-4" />
            {t("Assess now")}
          </Button>
        ) : undefined,
      }}
    >
      <div className="flex flex-wrap items-end gap-3">
        <div className="w-65">
          <CustomerAutocompleteField
            control={filterForm.control}
            name="customerId"
            label={t("Customer")}
            placeholder={t("All customers")}
            clearable
          />
        </div>
        <div className="w-45">
          <AutoCompleteDateField
            control={filterForm.control}
            name="asOfDate"
            label={t("As of")}
            placeholder={t("Today")}
            clearable
          />
        </div>
      </div>

      {controlQuery.isLoading ? null : (
        <Alert variant={notice.tone} size="sm">
          <AlertTriangleIcon />
          <AlertDescription>
            <span>
              {notice.text}{" "}
              <Link to="/billing/configuration-files/billing-control" className="underline">
                {t("Billing control")}
              </Link>
            </span>
          </AlertDescription>
        </Alert>
      )}

      {previewQuery.isLoading && !result ? (
        <Skeleton className="h-64 w-full rounded-md" />
      ) : previewQuery.isError && !result ? (
        <Alert variant="destructive" size="sm">
          <AlertDescription>
            {t("Failed to load the late charge preview. Try refreshing the page.")}
          </AlertDescription>
        </Alert>
      ) : result ? (
        <LateChargePreviewTable
          result={result}
          selected={selected}
          onSelectedChange={setSelected}
          hasActiveFilters={Boolean(customerId || asOfValue)}
          onClearFilters={() => filterForm.reset()}
        />
      ) : null}

      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Assess late charges now?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Raises {0, plural, one {# debit memo} other {# debit memos}} totalling {1}. Each invoice period is charged once; a rerun cannot double-charge.",
                selectedIds.length,
                formatCurrency(selectedTotalMinor / 100),
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction disabled={assess.isPending} onClick={() => assess.mutate()}>
              {t("Assess late charges")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </PageLayout>
  );
}
