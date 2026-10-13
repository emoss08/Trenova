import { useT } from "@trenova/shared/i18n/use-t";
import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { SectionPanel } from "@/components/section-panel";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  CREDIT_STATUS_LABEL,
  PROMISE_STATUS_LABEL,
  PROMISE_STATUS_TONE,
  promisePaidShare,
} from "@/lib/collection-labels";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { CollectionAccountView, CollectionActivity } from "@/types/collection";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { Link } from "react-router";
import { customerStatementPath } from "@/lib/ar-links";
import { toast } from "sonner";
import { Mail01Icon } from "@trenova/shared/components/icons";
import { ActivityTimeline } from "./activity-timeline";
import { SendReminderDialog } from "./send-reminder-dialog";
import { LogActivityForm } from "./log-activity-form";

/**
 * One customer as collections works it: what is overdue, the promise standing
 * against it, who is chasing it and when, and everything said so far.
 */
export function CollectionAccountSheet({
  customerId,
  onClose,
}: {
  customerId: string | null;
  onClose: () => void;
}) {
  const t = useT();
  const { data: view, isLoading } = useQuery({
    ...queries.collection.account(customerId ?? ""),
    enabled: customerId !== null,
  });

  return (
    <Sheet open={customerId !== null} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="right" className="w-[min(92vw,640px)] gap-0 p-0 sm:max-w-none">
        <SheetHeader className="border-b px-4 py-3">
          <SheetTitle>{view?.row.customerName ?? t("Collections account")}</SheetTitle>
          <SheetDescription>
            {view
              ? t(
                  "{0} overdue of {1} open across {2, plural, one {# invoice} other {# invoices}}",
                  formatMinor(view.row.overdueMinor, view.row.currencyCode),
                  formatMinor(view.row.totalOpenMinor, view.row.currencyCode),
                  view.row.openInvoiceCount,
                )
              : t("Loading...")}
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="min-h-0 flex-1">
          {isLoading || !view || !customerId ? (
            <div className="flex flex-col gap-3 p-4">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-40 w-full" />
            </div>
          ) : (
            <AccountBody customerId={customerId} view={view} />
          )}
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}

function AccountBody({ customerId, view }: { customerId: string; view: CollectionAccountView }) {
  const t = useT();
  const { row } = view;
  const [reminding, setReminding] = useState(false);

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="flex justify-end">
        <Button size="sm" variant="outline" onClick={() => setReminding(true)}>
          <Mail01Icon className="size-3.5" />
          {t("Send reminder")}
        </Button>
      </div>
      <SendReminderDialog
        customerId={customerId}
        customerName={row.customerName}
        open={reminding}
        onOpenChange={setReminding}
      />
      <DescriptionList columns={2}>
        <DescriptionItem label={t("Oldest past due")}>
          {row.oldestDaysPastDue > 0 ? t("{0} days", row.oldestDaysPastDue) : t("Not past due")}
        </DescriptionItem>
        <DescriptionItem label={t("Credit")}>
          {CREDIT_STATUS_LABEL[row.creditStatus] ?? row.creditStatus}
        </DescriptionItem>
        <DescriptionItem label={t("Disputed invoices")}>{row.disputedInvoiceCount}</DescriptionItem>
        <DescriptionItem label={t("Receivables")}>
          <span className="flex gap-3">
            <Link
              className="ui-focus-ring text-brand rounded-control underline-offset-2 hover:underline"
              to={`/accounting/ar/customer-ledger?customerId=${customerId}`}
            >
              {t("Ledger")}
            </Link>
            <Link
              className="ui-focus-ring text-brand rounded-control underline-offset-2 hover:underline"
              to={`/accounting/ar/open-items?customerId=${customerId}`}
            >
              {t("Open items")}
            </Link>
            <Link
              className="ui-focus-ring text-brand rounded-control underline-offset-2 hover:underline"
              to={customerStatementPath(customerId)}
            >
              {t("Statement")}
            </Link>
          </span>
        </DescriptionItem>
      </DescriptionList>

      {row.promise ? (
        <SectionPanel
          title={t("Promise to pay")}
          action={
            <Badge variant={PROMISE_STATUS_TONE[row.promise.status]}>
              {PROMISE_STATUS_LABEL[row.promise.status]}
            </Badge>
          }
        >
          <div className="flex flex-col gap-1 px-3 py-2.5 text-sm">
            <span className="tabular-nums">
              {t(
                "{0} by {1}, promised to {2}",
                formatMinor(row.promise.amountMinor, row.promise.currencyCode),
                formatUnixDateMedium(row.promise.promisedOn),
                row.promise.madeByName,
              )}
            </span>
            <span className="text-muted-foreground text-xs tabular-nums">
              {t(
                "{0} paid since ({1}%)",
                formatMinor(row.promise.paidSinceMinor, row.promise.currencyCode),
                promisePaidShare(row.promise),
              )}
            </span>
          </div>
        </SectionPanel>
      ) : null}

      <AssignmentPanel customerId={customerId} view={view} />

      <SectionPanel title={t("Log a contact")}>
        <div className="px-3 py-3">
          <LogActivityForm customerId={customerId} />
        </div>
      </SectionPanel>

      <HistoryPanel
        key={`${customerId}:${view.activities[0]?.id ?? "none"}`}
        customerId={customerId}
        view={view}
      />
    </div>
  );
}

type AssignmentValues = {
  collectorId: string | null;
  followUpOn: number | null;
};

function AssignmentPanel({
  customerId,
  view,
}: {
  customerId: string;
  view: CollectionAccountView;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<AssignmentValues>({
    defaultValues: { collectorId: view.row.collectorId, followUpOn: view.row.nextFollowUpOn },
  });
  const { reset } = form;
  useEffect(() => {
    reset({ collectorId: view.row.collectorId, followUpOn: view.row.nextFollowUpOn });
  }, [reset, view.row.collectorId, view.row.nextFollowUpOn]);
  const followUpOn = useWatch({ control: form.control, name: "followUpOn" });

  const settle = (next: CollectionAccountView) => {
    queryClient.setQueryData(["collection", "account", customerId], next);
    void queryClient.invalidateQueries({ queryKey: ["collection", "worklist"] });
  };

  const assign = useApiMutation({
    mutationFn: (collectorId: string | null) =>
      apiService.collectionService.assignCollector(customerId, collectorId),
    resourceName: "CollectionAccount",
    form,
    onSuccess: (next) => {
      settle(next);
      toast.success(
        next.row.collectorName ? t("Assigned to {0}", next.row.collectorName) : t("Unassigned"),
      );
    },
  });
  const schedule = useApiMutation({
    mutationFn: (date: number | null) =>
      apiService.collectionService.scheduleFollowUp(customerId, date, ""),
    resourceName: "CollectionAccount",
    form,
    onSuccess: (next) => {
      settle(next);
      toast.success(
        next.row.nextFollowUpOn
          ? t("Follow-up set for {0}", formatUnixDateMedium(next.row.nextFollowUpOn))
          : t("Follow-up cleared"),
      );
    },
  });

  const followUpChanged = (followUpOn ?? null) !== view.row.nextFollowUpOn;

  return (
    <SectionPanel title={t("Collector and follow-up")}>
      <div className="grid grid-cols-2 gap-3 px-3 py-3">
        <UserAutocompleteField
          control={form.control}
          name="collectorId"
          label={t("Collector")}
          placeholder={t("Unassigned")}
          clearable
          onOptionChange={(option) => {
            const next = option?.id ?? null;
            if (next !== view.row.collectorId) assign.mutate(next);
          }}
        />
        <div className="flex flex-col gap-1.5">
          <AutoCompleteDateField
            control={form.control}
            name="followUpOn"
            label={t("Next follow-up")}
            placeholder={t("Not scheduled")}
            clearable
          />
          {followUpChanged ? (
            <Button
              size="sm"
              variant="outline"
              className="self-start"
              isLoading={schedule.isPending}
              onClick={() => schedule.mutate(followUpOn ?? null)}
            >
              {followUpOn ? t("Save follow-up") : t("Clear follow-up")}
            </Button>
          ) : null}
        </div>
      </div>
    </SectionPanel>
  );
}

function HistoryPanel({ customerId, view }: { customerId: string; view: CollectionAccountView }) {
  const t = useT();
  const [older, setOlder] = useState<CollectionActivity[]>([]);
  const [hasMore, setHasMore] = useState(view.hasMore);
  const [loading, setLoading] = useState(false);

  const activities = [...view.activities, ...older];
  const loadOlder = async () => {
    const last = activities.at(-1);
    if (!last) return;
    setLoading(true);
    try {
      const page = await apiService.collectionService.getAccount(customerId, {
        createdAt: last.createdAt,
        id: last.id,
      });
      setOlder((current) => [...current, ...page.activities]);
      setHasMore(page.hasMore);
    } catch {
      toast.error(t("Older history could not be loaded"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <SectionPanel title={t("History")} count={activities.length}>
      <ActivityTimeline activities={activities} />
      {hasMore ? (
        <div className="border-t px-3 py-2">
          <Button variant="ghost" size="sm" isLoading={loading} onClick={() => void loadOlder()}>
            {t("Load older")}
          </Button>
        </div>
      ) : null}
    </SectionPanel>
  );
}
