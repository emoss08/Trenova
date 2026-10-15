import type { FactoringRemittanceLine } from "@/lib/graphql/factoring-remittance";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import type {
  FactoringRemittanceLineKind,
  FactoringSubmissionItemStatus,
} from "@trenova/graphql/generated/graphql";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { useState } from "react";

/** The schedule item statuses a line of this kind can be booked against. */
export function acceptedItemStatuses(
  kind: FactoringRemittanceLineKind,
): FactoringSubmissionItemStatus[] {
  switch (kind) {
    case "Funding":
    case "Rejection":
      return ["Pending"];
    case "ReserveRelease":
    case "Chargeback":
      return ["Funded"];
    case "Other":
      return [];
  }
}

/**
 * Matches a line to any open item on the factor's schedules, for when none of
 * the suggestions is the invoice the factor paid.
 */
export function MatchItemDialog({
  factoringCompanyId,
  currencyCode,
  line,
  pending,
  onMatch,
  onOpenChange,
}: {
  factoringCompanyId: string;
  currencyCode: string;
  line: FactoringRemittanceLine | null;
  pending: boolean;
  onMatch: (itemId: string) => void;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const open = line !== null;
  const [scheduleId, setScheduleId] = useState<string>("");
  const [itemId, setItemId] = useState<string>("");
  const schedules = useQuery({
    ...factoringQueries.openSchedules(factoringCompanyId),
    enabled: open,
  });
  const schedule = useQuery({
    ...factoringQueries.submission(scheduleId),
    enabled: scheduleId !== "",
  });
  const accepted = line ? acceptedItemStatuses(line.kind) : [];
  const items = (schedule.data?.items ?? []).filter((item) => accepted.includes(item.status));

  const close = (next: boolean) => {
    if (!next) {
      setScheduleId("");
      setItemId("");
    }
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Match to an invoice")}</DialogTitle>
          <DialogDescription>
            {line
              ? t(
                  "Row {0} ({1}) pays which invoice on the factor's schedules?",
                  line.rowNumber,
                  line.invoiceNumber ?? line.factorReference ?? "",
                )
              : null}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          {schedules.isLoading ? (
            <Skeleton className="h-9 w-full" />
          ) : (
            <label className="flex flex-col gap-1 text-sm">
              <span className="font-medium">{t("Schedule")}</span>
              <Select
                value={scheduleId}
                onValueChange={(value) => {
                  setScheduleId(String(value));
                  setItemId("");
                }}
              >
                <SelectTrigger aria-label={t("Schedule")}>
                  <SelectValue placeholder={t("Select a schedule")} />
                </SelectTrigger>
                <SelectContent>
                  {(schedules.data ?? []).map((option) => (
                    <SelectItem key={option.id} value={option.id}>
                      {t(
                        "{0} · {1, plural, one {# invoice} other {# invoices}}",
                        option.number,
                        option.invoiceCount,
                      )}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
          )}
          {scheduleId !== "" ? (
            schedule.isLoading ? (
              <Skeleton className="h-9 w-full" />
            ) : items.length === 0 ? (
              <p className="text-muted-foreground text-sm">
                {t("Nothing on this schedule can take this line.")}
              </p>
            ) : (
              <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{t("Invoice")}</span>
                <Select value={itemId} onValueChange={(value) => setItemId(String(value))}>
                  <SelectTrigger aria-label={t("Invoice")}>
                    <SelectValue placeholder={t("Select an invoice")} />
                  </SelectTrigger>
                  <SelectContent>
                    {items.map((item) => (
                      <SelectItem key={item.id} value={item.id}>
                        {t(
                          "{0} · {1}",
                          item.invoiceNumber,
                          formatCurrency(item.faceMinor / 100, currencyCode),
                        )}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>
            )
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => close(false)}>
            {t("Back")}
          </Button>
          <Button
            type="button"
            disabled={itemId === ""}
            isLoading={pending}
            onClick={() => onMatch(itemId)}
          >
            {t("Match")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
