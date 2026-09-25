import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useAccountingSyncActions } from "@/hooks/use-accounting-sync-actions";
import { useAccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import { usePermission } from "@/hooks/use-permission";
import type { AccountingSyncSummary } from "@/lib/graphql/accounting-sync-ledger";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { getEndOfDay } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { bucketCount } from "./ledger-filters";
import {
  backfillObjectTypes,
  backfillSchema,
  pauseSchema,
  type BackfillValues,
  type PauseValues,
} from "./ledger-schemas";

export function LedgerHeaderActions({ summary }: { summary: AccountingSyncSummary }) {
  const t = useT();
  const { allowed: canUpdate } = usePermission(Resource.AccountingSync, Operation.Update);
  const { allowed: canManage } = usePermission(Resource.AccountingIntegration, Operation.Manage);
  const actions = useAccountingSyncActions(summary.integrationType, summary.providerName);
  const [dialog, setDialog] = useState<"pause" | "release" | "backfill" | null>(null);
  const connection = summary.connection;

  if (!connection?.syncEnabledAt) {
    return null;
  }

  const failed = bucketCount(summary.counts, "attention");
  const held = bucketCount(summary.counts, "held");
  const paused = connection.pausedAt != null;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {canUpdate && failed > 0 ? (
        <Button
          type="button"
          size="sm"
          variant="outline"
          isLoading={actions.retry.isPending}
          onClick={() => actions.retry.mutate({})}
        >
          {t("Retry failed ({0})", failed)}
        </Button>
      ) : null}
      {canUpdate && held > 0 ? (
        <Button type="button" size="sm" variant="outline" onClick={() => setDialog("release")}>
          {t("Release held ({0})", held)}
        </Button>
      ) : null}
      {canManage && !summary.activeBackfill ? (
        <Button type="button" size="sm" variant="outline" onClick={() => setDialog("backfill")}>
          {t("Backfill")}
        </Button>
      ) : null}
      {canUpdate ? (
        paused ? (
          <Button
            type="button"
            size="sm"
            isLoading={actions.resume.isPending}
            onClick={() => actions.resume.mutate()}
          >
            {t("Resume sending")}
          </Button>
        ) : (
          <Button type="button" size="sm" variant="outline" onClick={() => setDialog("pause")}>
            {t("Pause sending")}
          </Button>
        )
      ) : null}

      <PauseDialog
        open={dialog === "pause"}
        providerName={summary.providerName}
        isPending={actions.pause.isPending}
        onOpenChange={(open) => setDialog(open ? "pause" : null)}
        onSubmit={(values) => actions.pause.mutateAsync(values.reason).then(() => setDialog(null))}
      />
      <AlertDialog
        open={dialog === "release"}
        onOpenChange={(open) => setDialog(open ? "release" : null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t(
                "{0, plural, one {Release # held document?} other {Release # held documents?}}",
                held,
              )}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "They are sent to {0} right away, in the order they were posted.",
                summary.providerName,
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={actions.release.isPending}>
              {t("Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              isLoading={actions.release.isPending}
              loadingText={t("Releasing...")}
              onClick={() => {
                void actions.release.mutateAsync(undefined).then(
                  () => setDialog(null),
                  () => undefined,
                );
              }}
            >
              {t("Release")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      {dialog === "backfill" ? (
        <BackfillDialog
          summary={summary}
          isPending={actions.backfill.isPending}
          onOpenChange={(open) => setDialog(open ? "backfill" : null)}
          onSubmit={(values) => actions.backfill.mutateAsync(values).then(() => setDialog(null))}
        />
      ) : null}
    </div>
  );
}

function PauseDialog({
  open,
  providerName,
  isPending,
  onOpenChange,
  onSubmit,
}: {
  open: boolean;
  providerName: string;
  isPending: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (values: PauseValues) => Promise<unknown>;
}) {
  const t = useT();
  const form = useForm<PauseValues>({
    resolver: zodResolver(pauseSchema),
    defaultValues: { reason: "" },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Pause sending to {0}?", providerName)}</DialogTitle>
          <DialogDescription>
            {t(
              "Posted documents keep queueing while sending is paused, and go out in order when it resumes.",
            )}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={form.handleSubmit((values) =>
            onSubmit(values).then(
              () => form.reset(),
              () => undefined,
            ),
          )}
        >
          <FormGroup cols={1}>
            <FormControl>
              <TextareaField
                name="reason"
                control={form.control}
                label={t("Reason (optional)")}
                placeholder={t("Month-end close in progress")}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={isPending} loadingText={t("Pausing...")}>
              {t("Pause sending")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

function BackfillDialog({
  summary,
  isPending,
  onOpenChange,
  onSubmit,
}: {
  summary: AccountingSyncSummary;
  isPending: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (values: BackfillValues) => Promise<unknown>;
}) {
  const t = useT();
  const labels = useAccountingSyncLabels();
  const latestAllowed = useMemo(() => getEndOfDay(), []);
  const connection = summary.connection;
  const form = useForm<BackfillValues>({
    resolver: zodResolver(backfillSchema(latestAllowed)),
    defaultValues: {
      rangeStart: connection?.syncStartDate ?? undefined,
      rangeEnd: connection?.syncEnabledAt ?? latestAllowed,
      objectTypes: [],
    },
  });

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Backfill documents")}</DialogTitle>
          <DialogDescription>
            {t(
              "Queues posted documents dated in the range that were never queued, such as those posted before sync was turned on. Documents already queued are left alone.",
            )}
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={form.handleSubmit((values) => onSubmit(values).catch(() => undefined))}>
          <FormGroup cols={2}>
            <FormControl>
              <AutoCompleteDateField name="rangeStart" control={form.control} label={t("From")} />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField name="rangeEnd" control={form.control} label={t("To")} />
            </FormControl>
            <FormControl cols="full">
              <MultiCheckboxField
                name="objectTypes"
                control={form.control}
                label={t("Documents (leave all unticked for every kind)")}
                options={backfillObjectTypes(connection?.syncsDriverSettlements ?? false).map(
                  (value) => ({
                    value,
                    label: labels.objectType[value],
                  }),
                )}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={isPending} loadingText={t("Starting...")}>
              {t("Start backfill")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
