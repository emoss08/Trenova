import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { recordPath } from "@/config/record-links";
import {
  submitInvoicesToFactor,
  type FactoringInvoiceStateRow,
  type FactoringSubmissionPlan,
} from "@/lib/graphql/factoring";
import {
  FACTORING_SUBMISSION_MAX_INVOICES,
  FACTORING_TRANSPORT_METHOD_LABELS,
  formatBasisPoints,
  groupInvoicesByFactor,
} from "@/lib/factoring";
import { factoring as factoringQueries, invalidateFactoringQueries } from "@/lib/queries/factoring";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";

/**
 * Sends the selected ready invoices to their factor on one schedule of
 * accounts. A schedule goes to one factor, so a selection spanning several
 * picks which factor's invoices to send; the plan shown is what the server
 * would send, with the funding it expects under the agreement in force, and
 * nothing leaves until the person confirms it.
 */
export function SubmitToFactorDialog({
  open,
  rows,
  onOpenChange,
}: {
  open: boolean;
  rows: FactoringInvoiceStateRow[];
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const groups = useMemo(() => groupInvoicesByFactor(rows), [rows]);
  const [chosenFactorId, setChosenFactorId] = useState<string | null>(null);
  const group =
    groups.find((candidate) => candidate.factoringCompanyId === chosenFactorId) ?? groups[0];
  const invoiceIds = group?.invoiceIds ?? [];
  const tooMany = invoiceIds.length > FACTORING_SUBMISSION_MAX_INVOICES;

  const plan = useQuery({
    ...factoringQueries.plan(group?.factoringCompanyId ?? "", invoiceIds),
    enabled: open && !!group && !tooMany,
    retry: false,
    staleTime: 0,
  });

  const submit = useMutation({
    mutationFn: () =>
      submitInvoicesToFactor({
        factoringCompanyId: group!.factoringCompanyId,
        invoiceIds,
      }),
    onSuccess: (submission) => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
      void navigate(recordPath("factoring_submission", submission.id));
    },
    onError: (error) => {
      void plan.refetch();
      const described = describeError(error);
      toast.error(t("The schedule was not sent"), {
        description: described.detail ?? described.message,
      });
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl">
        <DialogHeader>
          <DialogTitle>{t("Submit to factor")}</DialogTitle>
          <DialogDescription>
            {t(
              "Review the schedule of accounts and the funding expected under the agreement in force. The schedule, each invoice's packet and a CSV are sent to the factor once you confirm.",
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="flex min-w-0 flex-col gap-4">
          {groups.length === 0 ? (
            <Alert size="sm" variant="warning">
              <AlertDescription>
                {t("None of the selected invoices is assigned to a factor.")}
              </AlertDescription>
            </Alert>
          ) : null}
          {groups.length > 1 && group ? (
            <FactorChoice
              groups={groups}
              value={group.factoringCompanyId}
              onChange={setChosenFactorId}
              selectedCount={rows.length}
            />
          ) : null}
          {tooMany ? (
            <Alert size="sm" variant="warning">
              <AlertDescription>
                {t(
                  "A schedule carries at most {0} invoices. Select fewer and send the rest on another schedule.",
                  FACTORING_SUBMISSION_MAX_INVOICES,
                )}
              </AlertDescription>
            </Alert>
          ) : null}
          {group && !tooMany ? <PlanBody query={plan} /> : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            disabled={!plan.data || submit.isPending || tooMany}
            isLoading={submit.isPending}
            loadingText={t("Sending...")}
            onClick={() => submit.mutate()}
          >
            {plan.data
              ? t("Send {0, plural, one {# invoice} other {# invoices}}", plan.data.invoiceCount)
              : t("Send")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function FactorChoice({
  groups,
  value,
  onChange,
  selectedCount,
}: {
  groups: ReturnType<typeof groupInvoicesByFactor>;
  value: string;
  onChange: (factoringCompanyId: string) => void;
  selectedCount: number;
}) {
  const t = useT();
  const items = groups.map((group) => ({
    value: group.factoringCompanyId,
    label: t(
      "{0} ({1, plural, one {# invoice} other {# invoices}})",
      group.name,
      group.invoiceIds.length,
    ),
  }));

  return (
    <Alert size="sm" variant="info">
      <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
        <span>
          {t(
            "The {0} selected invoices are with {1} factors. A schedule goes to one factor; send the others on their own schedule.",
            selectedCount,
            groups.length,
          )}
        </span>
        <Select
          value={value}
          items={items}
          onValueChange={(next) => {
            if (typeof next === "string") onChange(next);
          }}
        >
          <SelectTrigger className="w-64" aria-label={t("Factor")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </AlertDescription>
    </Alert>
  );
}

function PlanBody({
  query,
}: {
  query: { data?: FactoringSubmissionPlan; isLoading: boolean; error: unknown };
}) {
  const t = useT();

  if (query.isLoading) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  if (query.error || !query.data) {
    const described = query.error ? describeError(query.error) : null;
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription>
          {described?.detail ?? t("The schedule could not be planned. Try again in a moment.")}
        </AlertDescription>
      </Alert>
    );
  }

  const plan = query.data;
  const money = (minor: number) => formatCurrency(minor / 100, plan.currencyCode);

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <KpiStrip aria-label={t("Expected funding")}>
        <KpiStripItem
          label={t("Face value")}
          value={money(plan.faceMinor)}
          sub={t("{0, plural, one {# invoice} other {# invoices}}", plan.invoiceCount)}
        />
        <KpiStripItem label={t("Advance")} value={money(plan.expectedAdvanceMinor)} />
        <KpiStripItem label={t("Fee")} value={money(plan.expectedFeeMinor)} />
        <KpiStripItem
          label={t("Reserve")}
          value={money(plan.expectedReserveMinor)}
          sub={t("Released when customers pay")}
        />
      </KpiStrip>
      <p className="text-muted-foreground text-sm">
        {plan.transportMethod === "Email"
          ? plan.attachPackets
            ? t(
                "Emailed to {0} at {1}, with each invoice's packet attached.",
                plan.factoringCompanyName,
                plan.transportTarget,
              )
            : t(
                "Emailed to {0} at {1}, without the invoice packets.",
                plan.factoringCompanyName,
                plan.transportTarget,
              )
          : t(
              "Uploaded by {0} to {1}.",
              FACTORING_TRANSPORT_METHOD_LABELS[plan.transportMethod],
              plan.transportTarget,
            )}
      </p>
      <div className="max-h-80 overflow-auto rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Invoice #")}</TableHead>
              <TableHead>{t("Customer")}</TableHead>
              <TableHead>{t("Due")}</TableHead>
              <TableHead className="text-right">{t("Days")}</TableHead>
              <TableHead className="text-right">{t("Face value")}</TableHead>
              <TableHead className="text-right">{t("Advance")}</TableHead>
              <TableHead className="text-right">{t("Fee")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {plan.items.map((item) => (
              <TableRow key={item.invoiceId}>
                <TableCell className="font-mono">{item.invoiceNumber}</TableCell>
                <TableCell>{item.customerName}</TableCell>
                <TableCell>{item.dueDate ? formatUnixDate(item.dueDate) : ""}</TableCell>
                <TableCell className="text-right tabular-nums">{item.funding.days}</TableCell>
                <TableCell className="text-right tabular-nums">
                  {money(item.funding.faceMinor)}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {money(item.funding.advanceMinor)}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  <span title={formatBasisPoints(item.funding.feeRateBp)}>
                    {money(item.funding.feeMinor)}
                  </span>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
