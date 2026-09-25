import { useAccountingMappingActions } from "@/hooks/use-accounting-mapping-actions";
import { useAccountingMappingLabels } from "@/hooks/use-accounting-mapping-labels";
import { accountingMappingCreatable, accountingMappingRecordPath } from "@/lib/accounting-sync";
import type { AccountingMapping } from "@/lib/graphql/accounting-sync";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatPercent } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { Link } from "react-router";
import { MappingRecordPicker } from "./mapping-record-picker";
import { MappingStateBadge } from "./mapping-state-badge";

type MappingEditorProps = {
  system: AccountingSystem;
  providerName: string;
  mapping: AccountingMapping;
  canUpdate: boolean;
};

export function MappingEditor({ system, providerName, mapping, canUpdate }: MappingEditorProps) {
  const t = useT();
  const labels = useAccountingMappingLabels();
  const actions = useAccountingMappingActions(system, providerName);
  const [newName, setNewName] = useState(mapping.targetLabel);
  const recordLink = accountingMappingRecordPath(mapping);
  const busy =
    actions.confirm.isPending ||
    actions.reject.isPending ||
    actions.set.isPending ||
    actions.clear.isPending ||
    actions.create.isPending;
  const suggestedCandidates = mapping.candidates.filter(
    (candidate) => candidate.externalId !== mapping.externalId,
  );

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className="text-foreground-muted text-xs">
            {labels.targetType(mapping.targetType)}
          </div>
          <h3 className="truncate text-base font-semibold">
            {recordLink ? (
              <Link to={recordLink} className="text-brand hover:underline">
                {mapping.targetLabel}
              </Link>
            ) : (
              mapping.targetLabel
            )}
          </h3>
        </div>
        <MappingStateBadge state={mapping.state} source={mapping.source} />
      </div>

      <DescriptionList columns={2}>
        <DescriptionItem label={t("{0} record", providerName)}>
          {mapping.externalName || <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Kind")}>
          {labels.recordKind(mapping.providerKind)}
        </DescriptionItem>
        <DescriptionItem label={t("Chosen by")}>
          {mapping.source ? labels.source(mapping.source) : <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Confidence")} numeric>
          {mapping.confidence != null ? (
            formatPercent(mapping.confidence * 100, 0)
          ) : (
            <DescriptionEmpty />
          )}
        </DescriptionItem>
        {mapping.reason ? (
          <DescriptionItem label={t("Why")} className="col-span-2">
            {mapping.reason}
          </DescriptionItem>
        ) : null}
        {mapping.confirmedAt ? (
          <DescriptionItem label={t("Confirmed")} className="col-span-2">
            {mapping.confirmedBy?.name
              ? t("{0} by {1}", formatUnixDateMedium(mapping.confirmedAt), mapping.confirmedBy.name)
              : formatUnixDateMedium(mapping.confirmedAt)}
          </DescriptionItem>
        ) : null}
      </DescriptionList>

      {canUpdate && mapping.state !== "Unmatched" ? (
        <div className="flex flex-wrap gap-2">
          {mapping.state === "Proposed" ? (
            <>
              <Button
                type="button"
                size="sm"
                isLoading={actions.confirm.isPending}
                disabled={busy}
                onClick={() =>
                  actions.confirm.mutate([{ id: mapping.id, externalId: mapping.externalId }])
                }
              >
                {t("Confirm")}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                isLoading={actions.reject.isPending}
                disabled={busy}
                onClick={() => actions.reject.mutate(mapping.id)}
              >
                {t("Turn down")}
              </Button>
            </>
          ) : null}
          <Button
            type="button"
            size="sm"
            variant="ghost"
            isLoading={actions.clear.isPending}
            disabled={busy}
            onClick={() => actions.clear.mutate(mapping.id)}
          >
            {t("Clear")}
          </Button>
        </div>
      ) : null}

      {suggestedCandidates.length > 0 ? (
        <section className="space-y-2">
          <h4 className="flex items-center gap-1.5 text-sm font-medium">
            <AssistMark aria-hidden className="size-3.5" />
            {t("Other records Trenova considered")}
          </h4>
          <ul className="divide-border-subtle divide-y rounded-md border">
            {suggestedCandidates.map((candidate) => (
              <li
                key={candidate.externalId}
                className="flex items-center justify-between gap-3 px-3 py-2"
              >
                <div className="min-w-0">
                  <div className="truncate text-sm">{candidate.name}</div>
                  <div className="text-foreground-muted truncate text-xs">
                    {t("{0} match", formatPercent(candidate.score * 100, 0))}
                    {candidate.reason ? ` · ${candidate.reason}` : ""}
                  </div>
                </div>
                {canUpdate ? (
                  <Button
                    type="button"
                    size="xs"
                    variant="outline"
                    disabled={busy}
                    onClick={() =>
                      actions.set.mutate({
                        mappingId: mapping.id,
                        externalId: candidate.externalId,
                      })
                    }
                  >
                    {t("Use")}
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {canUpdate ? (
        <section className="space-y-2">
          <h4 className="text-sm font-medium">{t("Choose a {0} record", providerName)}</h4>
          <MappingRecordPicker
            system={system}
            kind={mapping.providerKind}
            providerName={providerName}
            disabled={busy}
            isChoosing={actions.set.isPending}
            onChoose={(ref) =>
              actions.set.mutate({ mappingId: mapping.id, externalId: ref.externalId })
            }
          />
        </section>
      ) : null}

      {canUpdate && accountingMappingCreatable(mapping) ? (
        <section className="space-y-2 border-t pt-4">
          <h4 className="text-sm font-medium">{t("Not in {0} yet?", providerName)}</h4>
          <p className="text-foreground-muted text-sm">
            {t(
              "Trenova creates the {0} in {1} and maps {2} to it.",
              labels.recordKind(mapping.providerKind).toLowerCase(),
              providerName,
              mapping.targetLabel,
            )}
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              value={newName}
              onChange={(event) => setNewName(event.target.value)}
              aria-label={t("Name in {0}", providerName)}
              className="max-w-xs"
              disabled={busy}
            />
            <Button
              type="button"
              size="sm"
              variant="outline"
              isLoading={actions.create.isPending}
              disabled={busy || newName.trim() === ""}
              onClick={() => actions.create.mutate({ mappingId: mapping.id, name: newName.trim() })}
            >
              {t("Create in {0}", providerName)}
            </Button>
          </div>
        </section>
      ) : null}
    </div>
  );
}
