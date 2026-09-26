import { TextareaField } from "@/components/fields/textarea-field";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { ExternalLink } from "@/components/link";
import { recordPath } from "@/config/record-links";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useAccountingSyncActions } from "@/hooks/use-accounting-sync-actions";
import { useAccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import { usePermission } from "@/hooks/use-permission";
import { accountingSyncObjectPath, accountingSyncRecordPhase } from "@/lib/accounting-sync";
import type { AccountingSyncRecord } from "@/lib/graphql/accounting-sync-ledger";
import type { AccountingSyncLedgerRow } from "@/lib/graphql/accounting-sync-ledger-table";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatDurationMs,
  formatUnixDate,
  formatUnixDateTimeShort,
} from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryStates } from "nuqs";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { skipSchema, type SkipValues } from "./ledger-schemas";

const RETRYABLE = new Set(["Blocked", "DeadLettered", "Retrying"]);
const SKIPPABLE = new Set(["Queued", "AwaitingApproval", "Retrying", "Blocked", "DeadLettered"]);

type LedgerRecordPanelProps = DataTablePanelProps<AccountingSyncLedgerRow> & {
  system: AccountingSystem;
  providerName: string;
};

export function LedgerRecordPanel({
  open,
  onOpenChange,
  row,
  system,
  providerName,
}: LedgerRecordPanelProps) {
  const t = useT();
  const [{ panelEntityId }] = useQueryStates(panelSearchParamsParser);
  const recordId = row?.id ?? panelEntityId;
  const fetched = useQuery({
    ...queries.accountingSync.syncRecord(recordId ?? ""),
    enabled: open && !row && !!recordId,
  });
  const record: AccountingSyncRecord | null = row ?? fetched.data ?? null;

  if (!record) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={t("Sync record")}
        size="lg"
      >
        {fetched.isError ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>{t("This sync record could not be loaded.")}</AlertDescription>
          </Alert>
        ) : (
          <ComponentLoader message={t("Loading sync record...")} />
        )}
      </DataTablePanelContainer>
    );
  }

  return (
    <LedgerRecordDetail
      open={open}
      onOpenChange={onOpenChange}
      record={record}
      system={system}
      providerName={providerName}
    />
  );
}

function LedgerRecordDetail({
  open,
  onOpenChange,
  record,
  system,
  providerName,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  record: AccountingSyncRecord;
  system: AccountingSystem;
  providerName: string;
}) {
  const t = useT();
  const labels = useAccountingSyncLabels();
  const { allowed: canUpdate } = usePermission(Resource.AccountingSync, Operation.Update);
  const actions = useAccountingSyncActions(system, providerName);
  const [skipping, setSkipping] = useState(false);
  const attempts = useQuery({ ...queries.accountingSync.syncAttempts(record.id), enabled: open });
  const documentLink = accountingSyncObjectPath(record.objectType, record.objectId);
  const failed = record.status === "Blocked" || record.status === "DeadLettered";

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`${labels.objectType[record.objectType]} ${record.objectNumber}`}
      description={t(
        "{0} · queued {1}",
        labels.sourceEvent[record.sourceEvent],
        formatUnixDateTimeShort(record.queuedAt),
      )}
      size="lg"
      footer={
        canUpdate ? (
          <div className="flex justify-end gap-2">
            {SKIPPABLE.has(record.status) ? (
              <Button type="button" variant="outline" onClick={() => setSkipping(true)}>
                {t("Skip")}
              </Button>
            ) : null}
            {record.status === "AwaitingApproval" ? (
              <Button
                type="button"
                isLoading={actions.release.isPending}
                onClick={() => actions.release.mutate([record.id])}
              >
                {t("Release")}
              </Button>
            ) : null}
            {RETRYABLE.has(record.status) ? (
              <Button
                type="button"
                isLoading={actions.retry.isPending}
                onClick={() => actions.retry.mutate({ ids: [record.id] })}
              >
                {t("Retry now")}
              </Button>
            ) : null}
          </div>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={phaseTone(accountingSyncRecordPhase(record.status))}>
            {labels.status[record.status]}
          </Badge>
          {record.errorCategory ? (
            <Badge variant="neutral" appearance="outline">
              {labels.errorCategory[record.errorCategory]}
            </Badge>
          ) : null}
        </div>

        {failed ? (
          <Alert variant={record.status === "DeadLettered" ? "destructive" : "warning"} size="sm">
            <AlertTitle>{record.resolution || t("{0} did not accept it", providerName)}</AlertTitle>
            {record.errorMessage ? (
              <AlertDescription>
                {record.errorMessage}
                {record.errorCode ? ` (${record.errorCode})` : ""}
              </AlertDescription>
            ) : null}
          </Alert>
        ) : null}

        {skipping ? (
          <SkipForm
            isPending={actions.skip.isPending}
            onCancel={() => setSkipping(false)}
            onSubmit={(values) =>
              actions.skip
                .mutateAsync({ id: record.id, reason: values.reason })
                .then(() => setSkipping(false))
            }
          />
        ) : null}

        <DescriptionList columns={2}>
          <DescriptionItem label={t("Document")}>
            {documentLink ? (
              <Link to={documentLink} className="text-brand font-medium hover:underline">
                {record.objectNumber}
              </Link>
            ) : (
              record.objectNumber
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Action")}>
            {labels.operation[record.operation]}
            {record.revision > 1 ? ` · ${t("revision {0}", record.revision)}` : ""}
          </DescriptionItem>
          <DescriptionItem label={t("Document date")} numeric>
            {record.documentDate ? formatUnixDate(record.documentDate) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Tries")} numeric>
            {record.attemptCount}
          </DescriptionItem>
          <DescriptionItem label={t("Next try")} numeric>
            {record.nextAttemptAt ? (
              formatUnixDateTimeShort(record.nextAttemptAt)
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("In {0}", providerName)}>
            {record.externalUrl ? (
              <ExternalLink href={record.externalUrl}>
                {record.externalDocNumber || record.externalId}
              </ExternalLink>
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Synced")} numeric>
            {record.syncedAt ? formatUnixDateTimeShort(record.syncedAt) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Waits for")}>
            {record.dependsOnRecordId ? (
              <Link
                to={recordPath("accounting_sync_record", record.dependsOnRecordId)}
                className="text-brand font-medium hover:underline"
              >
                {t("The record it depends on")}
              </Link>
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          {record.status === "Skipped" ? (
            <DescriptionItem label={t("Skipped by")}>
              {record.skippedBy?.name ?? t("Unknown user")}
              {record.skippedReason ? `: ${record.skippedReason}` : ""}
            </DescriptionItem>
          ) : null}
        </DescriptionList>

        <section className="space-y-2">
          <h3 className="text-sm font-semibold">{t("Tries")}</h3>
          {attempts.isLoading ? (
            <ComponentLoader message={t("Loading tries...")} />
          ) : attempts.data && attempts.data.length > 0 ? (
            <ol className="divide-y rounded-md border">
              {attempts.data.map((attempt) => (
                <li key={attempt.id} className="space-y-0.5 px-3 py-2 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="font-medium">
                      {t("Try {0}", attempt.attemptNumber)} · {attempt.outcome}
                    </span>
                    <span className="text-foreground-muted text-xs tabular-nums">
                      {formatUnixDateTimeShort(attempt.startedAt)} ·{" "}
                      {formatDurationMs(attempt.durationMs)}
                    </span>
                  </div>
                  {attempt.errorMessage ? (
                    <p className="text-foreground-muted text-xs">
                      {attempt.errorCategory
                        ? `${labels.errorCategory[attempt.errorCategory]}: `
                        : ""}
                      {attempt.errorMessage}
                    </p>
                  ) : null}
                </li>
              ))}
            </ol>
          ) : (
            <p className="text-foreground-muted text-sm">{t("Not tried yet.")}</p>
          )}
        </section>
      </div>
    </DataTablePanelContainer>
  );
}

function SkipForm({
  isPending,
  onCancel,
  onSubmit,
}: {
  isPending: boolean;
  onCancel: () => void;
  onSubmit: (values: SkipValues) => Promise<unknown>;
}) {
  const t = useT();
  const form = useForm<SkipValues>({
    resolver: zodResolver(skipSchema),
    defaultValues: { reason: "" },
  });

  return (
    <Form
      onSubmit={form.handleSubmit((values) => onSubmit(values).catch(() => undefined))}
      className="space-y-3 rounded-md border p-3"
    >
      <FormGroup cols={1}>
        <FormControl>
          <TextareaField
            name="reason"
            control={form.control}
            label={t("Why is this document not sent?")}
            placeholder={t("Already entered in the books by hand")}
          />
        </FormControl>
      </FormGroup>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("Cancel")}
        </Button>
        <Button type="submit" variant="destructive" isLoading={isPending}>
          {t("Skip document")}
        </Button>
      </div>
    </Form>
  );
}
