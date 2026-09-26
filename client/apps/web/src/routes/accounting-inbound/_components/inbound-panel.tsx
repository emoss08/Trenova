import { TextareaField } from "@/components/fields/textarea-field";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { ExternalLink } from "@/components/link";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useAccountingInboundActions } from "@/hooks/use-accounting-inbound-actions";
import { useAccountingInboundLabels } from "@/hooks/use-accounting-inbound-labels";
import { usePermission } from "@/hooks/use-permission";
import {
  accountingAppliedObjectPath,
  accountingInboundCanApply,
  accountingInboundIsOpen,
  accountingInboundPhase,
  accountingSyncObjectPath,
  formatAccountingMinor,
} from "@/lib/accounting-sync";
import type {
  AccountingInboundApplyPreview,
  AccountingInboundChange,
} from "@/lib/graphql/accounting-inbound";
import type { AccountingInboundRow } from "@/lib/graphql/accounting-inbound-table";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { formatUnixDate, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryStates } from "nuqs";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { ignoreInboundSchema, type IgnoreInboundValues } from "./inbound-schemas";

type InboundChangePanelProps = DataTablePanelProps<AccountingInboundRow> & {
  providerName: string;
};

export function InboundChangePanel({
  open,
  onOpenChange,
  row,
  providerName,
}: InboundChangePanelProps) {
  const t = useT();
  const [{ panelEntityId }] = useQueryStates(panelSearchParamsParser);
  const changeId = row?.id ?? panelEntityId;
  const fetched = useQuery({
    ...queries.accountingSync.inboundChange(changeId ?? ""),
    enabled: open && !row && !!changeId,
  });
  const change: AccountingInboundChange | null = row ?? fetched.data ?? null;

  if (!change) {
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={t("Payment from the books")}
        size="lg"
      >
        {fetched.isError ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>{t("This payment could not be loaded.")}</AlertDescription>
          </Alert>
        ) : (
          <ComponentLoader message={t("Loading payment...")} />
        )}
      </DataTablePanelContainer>
    );
  }

  return (
    <InboundChangeDetail
      open={open}
      onOpenChange={onOpenChange}
      change={change}
      providerName={providerName}
    />
  );
}

function InboundChangeDetail({
  open,
  onOpenChange,
  change,
  providerName,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  change: AccountingInboundChange;
  providerName: string;
}) {
  const t = useT();
  const labels = useAccountingInboundLabels();
  const { allowed: canUpdate } = usePermission(Resource.AccountingSync, Operation.Update);
  const actions = useAccountingInboundActions();
  const [ignoring, setIgnoring] = useState(false);
  const applicable = accountingInboundCanApply(change.status, change.reason);
  const preview = useQuery({
    ...queries.accountingSync.inboundPreview(change.id),
    enabled: open && applicable,
  });
  const currency = change.currencyCode;
  const canIgnore = canUpdate && accountingInboundIsOpen(change.status);
  const canApply = canUpdate && applicable && preview.data?.canApply === true;
  const number = change.externalNumber || change.referenceNumber || change.externalId;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={`${labels.kind[change.kind]} ${number}`}
      description={t(
        "{0} · {1} · recorded in {2}",
        change.partyName || t("Unknown party"),
        formatAccountingMinor(change.amountMinor, currency),
        providerName,
      )}
      size="lg"
      footer={
        canIgnore || canApply ? (
          <div className="flex justify-end gap-2">
            {canIgnore ? (
              <Button type="button" variant="outline" onClick={() => setIgnoring(true)}>
                {t("Ignore")}
              </Button>
            ) : null}
            {canApply ? (
              <Button
                type="button"
                isLoading={actions.apply.isPending}
                loadingText={t("Applying...")}
                onClick={() => actions.apply.mutate(change.id)}
              >
                {t("Apply in Trenova")}
              </Button>
            ) : null}
          </div>
        ) : undefined
      }
    >
      <div className="space-y-5">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={phaseTone(accountingInboundPhase(change.status))}>
            {labels.status[change.status]}
          </Badge>
          {change.reason ? (
            <Badge variant="neutral" appearance="outline">
              {labels.reason[change.reason]}
            </Badge>
          ) : null}
        </div>

        {change.resolution && change.status !== "Applied" ? (
          <Alert size="sm" variant={applicable ? "info" : "warning"}>
            <AlertTitle>{change.reason ? labels.reason[change.reason] : t("Note")}</AlertTitle>
            <AlertDescription>{change.resolution}</AlertDescription>
          </Alert>
        ) : null}

        {ignoring ? (
          <IgnoreForm
            isPending={actions.ignore.isPending}
            onCancel={() => setIgnoring(false)}
            onSubmit={(values) =>
              actions.ignore
                .mutateAsync({ id: change.id, note: values.note })
                .then(() => setIgnoring(false))
            }
          />
        ) : null}

        <DescriptionList columns={2}>
          <DescriptionItem label={t("Paid by")}>
            {change.partyName || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Paid on")} numeric>
            {change.txnDate ? formatUnixDate(change.txnDate) : <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Amount")} numeric>
            {formatAccountingMinor(change.amountMinor, currency)}
          </DescriptionItem>
          <DescriptionItem label={t("Left unapplied")} numeric>
            {change.unappliedMinor > 0 ? (
              formatAccountingMinor(change.unappliedMinor, currency)
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Reference")}>
            {change.referenceNumber || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Method")}>
            {change.methodName || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("In {0}", providerName)}>
            {change.externalUrl ? (
              <ExternalLink href={change.externalUrl}>{number}</ExternalLink>
            ) : (
              number
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Changed in {0}", providerName)}>
            {change.providerModifiedAt ? (
              change.providerModifiedBy ? (
                t(
                  "{0} by {1}",
                  formatUnixDateTimeShort(change.providerModifiedAt),
                  change.providerModifiedBy,
                )
              ) : (
                formatUnixDateTimeShort(change.providerModifiedAt)
              )
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
          <DescriptionItem label={t("Read by Trenova")} numeric>
            {formatUnixDateTimeShort(change.detectedAt)}
          </DescriptionItem>
          {change.decidedAt ? (
            <DescriptionItem
              label={change.status === "Ignored" ? t("Ignored by") : t("Applied by")}
            >
              {t(
                "{0}, {1}",
                change.decidedBy?.name ?? t("Trenova"),
                formatUnixDateTimeShort(change.decidedAt),
              )}
              {change.note ? `: ${change.note}` : ""}
            </DescriptionItem>
          ) : null}
        </DescriptionList>

        <section className="space-y-2">
          <h3 className="text-sm font-semibold">{t("What it pays")}</h3>
          {change.lines.length > 0 ? (
            <ul className="divide-y rounded-md border">
              {change.lines.map((line, idx) => {
                const path =
                  line.objectType && line.objectId
                    ? accountingSyncObjectPath(line.objectType, line.objectId)
                    : null;
                return (
                  <li
                    key={`${line.documentKind}:${line.documentExternalId}:${idx}`}
                    className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm"
                  >
                    <div className="min-w-0 space-y-0.5">
                      <p className="truncate">
                        {labels.documentKind[line.documentKind]}{" "}
                        {path ? (
                          <Link to={path} className="text-brand font-medium hover:underline">
                            {line.objectNumber}
                          </Link>
                        ) : (
                          line.objectNumber || line.documentExternalId
                        )}
                      </p>
                      <p className="text-foreground-muted text-xs">
                        {line.objectId
                          ? t(
                              "{0} open in Trenova",
                              formatAccountingMinor(line.openMinor, currency),
                            )
                          : t("Not a document Trenova sent")}
                      </p>
                    </div>
                    <span className="tabular-nums">
                      {formatAccountingMinor(line.amountMinor, currency)}
                    </span>
                  </li>
                );
              })}
            </ul>
          ) : (
            <p className="text-foreground-muted text-sm">{t("It does not name any document.")}</p>
          )}
        </section>

        {applicable ? (
          <InboundApplyPreview
            isLoading={preview.isLoading}
            isError={preview.isError}
            preview={preview.data}
            currency={currency}
          />
        ) : null}

        {change.appliedObjects.length > 0 ? (
          <section className="space-y-2">
            <h3 className="text-sm font-semibold">{t("Posted in Trenova")}</h3>
            <ul className="divide-y rounded-md border">
              {change.appliedObjects.map((object) => {
                const path = accountingAppliedObjectPath(object.type, object.id);
                return (
                  <li key={`${object.type}:${object.id}`} className="px-3 py-2 text-sm">
                    {path ? (
                      <Link to={path} className="text-brand font-medium hover:underline">
                        {labels.appliedObject[object.type]}
                      </Link>
                    ) : (
                      labels.appliedObject[object.type]
                    )}
                  </li>
                );
              })}
            </ul>
          </section>
        ) : null}
      </div>
    </DataTablePanelContainer>
  );
}

function InboundApplyPreview({
  isLoading,
  isError,
  preview,
  currency,
}: {
  isLoading: boolean;
  isError: boolean;
  preview: AccountingInboundApplyPreview | undefined;
  currency: string;
}) {
  const t = useT();

  return (
    <section className="space-y-2">
      <h3 className="text-sm font-semibold">{t("Applying it posts")}</h3>
      {isLoading ? <ComponentLoader message={t("Working out what it posts...")} /> : null}
      {isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("What it would post could not be worked out.")}</AlertDescription>
        </Alert>
      ) : null}
      {preview && !preview.canApply ? (
        <Alert size="sm" variant="warning">
          <AlertTitle>{t("It cannot be applied yet")}</AlertTitle>
          <AlertDescription>{preview.blocker}</AlertDescription>
        </Alert>
      ) : null}
      {preview?.canApply ? (
        <>
          <ul className="divide-y rounded-md border">
            {preview.lines.map((line) => (
              <li
                key={`${line.objectType}:${line.objectId}`}
                className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm"
              >
                <span className="min-w-0 truncate">
                  {t(
                    "{0}, open {1}",
                    line.objectNumber,
                    formatAccountingMinor(line.openMinor, currency),
                  )}
                </span>
                <span className="tabular-nums">
                  {formatAccountingMinor(line.amountMinor, currency)}
                </span>
              </li>
            ))}
          </ul>
          <p className="text-foreground-muted text-xs">
            {preview.unappliedMinor > 0
              ? t(
                  "Posted on {0}. {1} stays on the account as unapplied cash.",
                  formatUnixDate(preview.paidAt),
                  formatAccountingMinor(preview.unappliedMinor, currency),
                )
              : t("Posted on {0}.", formatUnixDate(preview.paidAt))}
          </p>
        </>
      ) : null}
    </section>
  );
}

function IgnoreForm({
  isPending,
  onCancel,
  onSubmit,
}: {
  isPending: boolean;
  onCancel: () => void;
  onSubmit: (values: IgnoreInboundValues) => Promise<unknown>;
}) {
  const t = useT();
  const form = useForm<IgnoreInboundValues>({
    resolver: zodResolver(ignoreInboundSchema),
    defaultValues: { note: "" },
  });

  return (
    <Form
      onSubmit={form.handleSubmit((values) => onSubmit(values).catch(() => undefined))}
      className="space-y-3 rounded-md border p-3"
    >
      <FormGroup cols={1}>
        <FormControl>
          <TextareaField
            name="note"
            control={form.control}
            label={t("Why does this payment stay out of Trenova?")}
            placeholder={t("Already recorded in Trenova by hand")}
          />
        </FormControl>
      </FormGroup>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("Cancel")}
        </Button>
        <Button type="submit" variant="destructive" isLoading={isPending}>
          {t("Ignore payment")}
        </Button>
      </div>
    </Form>
  );
}
