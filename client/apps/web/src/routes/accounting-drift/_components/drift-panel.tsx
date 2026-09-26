import { TextareaField } from "@/components/fields/textarea-field";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { ExternalLink } from "@/components/link";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useAccountingDriftActions } from "@/hooks/use-accounting-drift-actions";
import { useAccountingDriftLabels } from "@/hooks/use-accounting-drift-labels";
import { useAccountingSyncLabels } from "@/hooks/use-accounting-sync-labels";
import { usePermission } from "@/hooks/use-permission";
import { describeApiError } from "@/lib/api-error-message";
import {
  accountingDriftPhase,
  accountingDriftWithinTolerance,
  accountingSyncObjectPath,
  formatAccountingMinor,
} from "@/lib/accounting-sync";
import type { AccountingDriftFinding } from "@/lib/graphql/accounting-drift";
import type { AccountingDriftRow } from "@/lib/graphql/accounting-drift-table";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import type { AccountingDriftDirection } from "@trenova/graphql/generated/graphql";
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
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryStates } from "nuqs";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { dismissDriftSchema, type DismissDriftValues } from "./drift-schemas";

type DriftPanelProps = DataTablePanelProps<AccountingDriftRow> & {
  providerName: string;
  toleranceMinor: number;
};

type DriftAction = AccountingDriftDirection | "Dismiss";

export function DriftFindingPanel({
  open,
  onOpenChange,
  row,
  providerName,
  toleranceMinor,
}: DriftPanelProps) {
  const t = useT();
  const [{ panelEntityId }] = useQueryStates(panelSearchParamsParser);
  const findingId = row?.id ?? panelEntityId;
  const fetched = useQuery({
    ...queries.accountingSync.driftFinding(findingId ?? ""),
    enabled: open && !row && !!findingId,
  });
  const finding: AccountingDriftFinding | null = row ?? fetched.data ?? null;

  if (!finding) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={t("Difference with the books")}
        size="lg"
      >
        {fetched.isError ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>{t("This difference could not be loaded.")}</AlertDescription>
          </Alert>
        ) : (
          <ComponentLoader message={t("Loading difference...")} />
        )}
      </DataTablePanelContainer>
    );
  }

  return (
    <DriftFindingDetail
      open={open}
      onOpenChange={onOpenChange}
      finding={finding}
      providerName={providerName}
      toleranceMinor={toleranceMinor}
    />
  );
}

function DriftFindingDetail({
  open,
  onOpenChange,
  finding,
  providerName,
  toleranceMinor,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  finding: AccountingDriftFinding;
  providerName: string;
  toleranceMinor: number;
}) {
  const t = useT();
  const labels = useAccountingDriftLabels();
  const syncLabels = useAccountingSyncLabels();
  const { allowed: canUpdate } = usePermission(Resource.AccountingSync, Operation.Update);
  const [action, setAction] = useState<DriftAction | null>(null);
  const currency = finding.currencyCode;
  const isOpen = finding.status === "Open";
  const objectPath = accountingSyncObjectPath(finding.objectType, finding.objectId);
  const withinTolerance = accountingDriftWithinTolerance(finding, toleranceMinor);
  const actionable = canUpdate && isOpen && !finding.pushed;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`${syncLabels.objectType[finding.objectType]} ${finding.objectNumber}`}
      description={t(
        "{0} · {1} · in {2}",
        finding.partyName || t("Unknown party"),
        labels.kind[finding.kind],
        providerName,
      )}
      size="lg"
      footer={
        actionable && action === null ? (
          <div className="flex flex-wrap justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setAction("Dismiss")}>
              {t("Dismiss")}
            </Button>
            {finding.directions.map((direction) => (
              <Button
                key={direction}
                type="button"
                variant={direction === "PushTrenovaValue" ? "default" : "outline"}
                onClick={() => setAction(direction)}
              >
                {labels.direction[direction]}
              </Button>
            ))}
          </div>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={phaseTone(accountingDriftPhase(finding.status))}>
            {labels.status[finding.status]}
          </Badge>
          <Badge variant="neutral" appearance="outline">
            {labels.kind[finding.kind]}
          </Badge>
          {withinTolerance ? (
            <Badge variant="neutral" appearance="outline">
              {t("Within tolerance")}
            </Badge>
          ) : null}
        </div>

        {finding.pushed ? (
          <Alert size="sm" variant="info">
            <AlertTitle>{t("Trenova's value was sent")}</AlertTitle>
            <AlertDescription>
              {t(
                "The difference closes once the next check finds {0} matches Trenova.",
                providerName,
              )}
            </AlertDescription>
          </Alert>
        ) : null}

        <DescriptionList columns={2}>
          <DescriptionItem label={t("In Trenova")} numeric>
            {finding.trenovaMinor == null
              ? finding.trenovaState || <DescriptionEmpty />
              : formatAccountingMinor(finding.trenovaMinor, currency)}
          </DescriptionItem>
          <DescriptionItem label={t("In {0}", providerName)} numeric>
            {finding.providerMinor == null
              ? finding.providerState || <DescriptionEmpty />
              : formatAccountingMinor(finding.providerMinor, currency)}
          </DescriptionItem>
          <DescriptionItem label={t("State in Trenova")}>
            {finding.trenovaState || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("State in {0}", providerName)}>
            {finding.providerState || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Difference")} numeric>
            {finding.differenceMinor == null ? (
              <DescriptionEmpty />
            ) : (
              formatAccountingMinor(finding.differenceMinor, currency)
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Reconciliation tolerance")} numeric>
            {formatAccountingMinor(toleranceMinor, currency)}
          </DescriptionItem>
          <DescriptionItem label={t("Changed in {0}", providerName)}>
            {finding.providerModifiedAt ? (
              finding.providerModifiedBy ? (
                t(
                  "{0} by {1}",
                  formatUnixDateTimeShort(finding.providerModifiedAt),
                  finding.providerModifiedBy,
                )
              ) : (
                formatUnixDateTimeShort(finding.providerModifiedAt)
              )
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Found by Trenova")} numeric>
            {formatUnixDateTimeShort(finding.detectedAt)}
          </DescriptionItem>
          <DescriptionItem label={t("In Trenova")}>
            {objectPath ? (
              <Link to={objectPath} className="text-brand font-medium hover:underline">
                {finding.objectNumber}
              </Link>
            ) : (
              finding.objectNumber || <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("In {0}", providerName)}>
            {finding.externalUrl ? (
              <ExternalLink href={finding.externalUrl}>
                {finding.externalId || t("Open")}
              </ExternalLink>
            ) : (
              finding.externalId || <DescriptionEmpty />
            )}
          </DescriptionItem>
          {finding.resolvedAt ? (
            <DescriptionItem
              label={finding.status === "Dismissed" ? t("Dismissed by") : t("Settled")}
            >
              {finding.resolution ? `${labels.resolution[finding.resolution]}: ` : ""}
              {t(
                "{0}, {1}",
                finding.resolvedBy?.name ?? t("Trenova"),
                formatUnixDateTimeShort(finding.resolvedAt),
              )}
              {finding.resolutionNote ? ` · ${finding.resolutionNote}` : ""}
            </DescriptionItem>
          ) : null}
        </DescriptionList>

        {finding.detail.length > 0 ? (
          <section className="space-y-2">
            <h3 className="text-sm font-semibold">{t("Documents that differ")}</h3>
            <ul className="divide-y rounded-md border">
              {finding.detail.map((line) => {
                const path = accountingSyncObjectPath(line.objectType, line.objectId);
                return (
                  <li
                    key={line.objectId}
                    className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm"
                  >
                    <span className="min-w-0 truncate">
                      {path ? (
                        <Link to={path} className="text-brand font-medium hover:underline">
                          {line.objectNumber}
                        </Link>
                      ) : (
                        line.objectNumber
                      )}
                    </span>
                    <span className="tabular-nums">
                      {t(
                        "{0} in Trenova, {1} in {2}",
                        formatAccountingMinor(line.trenovaMinor, currency),
                        formatAccountingMinor(line.providerMinor, currency),
                        providerName,
                      )}
                    </span>
                  </li>
                );
              })}
            </ul>
          </section>
        ) : null}

        {action === "Dismiss" ? (
          <DismissForm
            findingId={finding.id}
            withinTolerance={withinTolerance}
            onDone={() => setAction(null)}
          />
        ) : null}
        {action && action !== "Dismiss" ? (
          <FixConfirmation
            findingId={finding.id}
            direction={action}
            open={open}
            onDone={() => setAction(null)}
          />
        ) : null}
      </div>
    </DataTablePanelContainer>
  );
}

function FixConfirmation({
  findingId,
  direction,
  open,
  onDone,
}: {
  findingId: string;
  direction: AccountingDriftDirection;
  open: boolean;
  onDone: () => void;
}) {
  const t = useT();
  const labels = useAccountingDriftLabels();
  const actions = useAccountingDriftActions();
  const preview = useQuery({
    ...queries.accountingSync.driftPreview(findingId, direction),
    enabled: open,
  });

  return (
    <section className="space-y-3 rounded-md border p-3">
      <h3 className="text-sm font-semibold">{labels.direction[direction]}</h3>
      {preview.isLoading ? <ComponentLoader message={t("Working out what it does...")} /> : null}
      {preview.isError ? (
        <Alert size="sm" variant="warning">
          <AlertTitle>{t("This fix cannot be made now")}</AlertTitle>
          <AlertDescription>
            {describeApiError(preview.error, t("What it would do could not be worked out."))}
          </AlertDescription>
        </Alert>
      ) : null}
      {preview.data ? <p className="text-sm">{preview.data.summary}</p> : null}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onDone}>
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          disabled={!preview.data}
          isLoading={actions.resolve.isPending}
          loadingText={t("Fixing...")}
          onClick={() =>
            actions.resolve
              .mutateAsync({ id: findingId, direction })
              .then(onDone)
              .catch(() => undefined)
          }
        >
          {labels.direction[direction]}
        </Button>
      </div>
    </section>
  );
}

function DismissForm({
  findingId,
  withinTolerance,
  onDone,
}: {
  findingId: string;
  withinTolerance: boolean;
  onDone: () => void;
}) {
  const t = useT();
  const actions = useAccountingDriftActions();
  const form = useForm<DismissDriftValues>({
    resolver: zodResolver(dismissDriftSchema),
    defaultValues: { note: "" },
  });

  return (
    <Form
      onSubmit={form.handleSubmit((values) =>
        actions.dismiss
          .mutateAsync({ id: findingId, note: values.note })
          .then(onDone)
          .catch(() => undefined),
      )}
      className="space-y-3 rounded-md border p-3"
    >
      <p className="text-sm">
        {withinTolerance
          ? t("Both sides stay as they are. The difference is within the tolerance.")
          : t("Both sides stay as they are. The difference is larger than the tolerance.")}
      </p>
      <FormGroup cols={1}>
        <FormControl>
          <TextareaField
            name="note"
            control={form.control}
            label={t("Why do both sides stay as they are?")}
            placeholder={t("Agreed with the customer; the books keep their figure")}
          />
        </FormControl>
      </FormGroup>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onDone}>
          {t("Cancel")}
        </Button>
        <Button type="submit" variant="destructive" isLoading={actions.dismiss.isPending}>
          {t("Dismiss difference")}
        </Button>
      </div>
    </Form>
  );
}
