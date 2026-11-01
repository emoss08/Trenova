import { MappingEditor } from "@/components/accounting-mapping/mapping-editor";
import { MappingRow } from "@/components/accounting-mapping/mapping-row";
import { ReferenceRefreshStatus } from "@/components/accounting-mapping/reference-refresh-status";
import { useAccountingMappingActions } from "@/hooks/use-accounting-mapping-actions";
import { useAccountingMappingList } from "@/hooks/use-accounting-mapping-list";
import {
  ACCOUNTING_MAPPINGS_PATH,
  checkedMappingConfirmations,
  mappingCheckKey,
} from "@/lib/accounting-sync";
import type { AccountingMappingSummary } from "@/lib/graphql/accounting-sync";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, useMemo, useState } from "react";
import { Link } from "react-router";
import type { AccountingVendor } from "./accounting-vendors";
import { IntegrationSetupFooter } from "../shared/integration-setup-wizard";

const REVIEW_PAGE_SIZE = 50;

export function AccountingMapStep({
  vendor,
  summary,
  canUpdate,
}: {
  vendor: AccountingVendor;
  summary: AccountingMappingSummary;
  canUpdate: boolean;
}) {
  const t = useT();
  const actions = useAccountingMappingActions(vendor.system, vendor.name);
  const required = useAccountingMappingList({
    system: vendor.system,
    filter: { requiredOnly: true },
    pageSize: REVIEW_PAGE_SIZE,
  });
  const proposals = useAccountingMappingList({
    system: vendor.system,
    filter: { states: ["Proposed"] },
    pageSize: REVIEW_PAGE_SIZE,
  });
  const [overrides, setOverrides] = useState<Record<string, boolean>>({});
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const progressLabelId = useId();

  const rows = useMemo(() => {
    const seen = new Set(required.mappings.map((mapping) => mapping.id));
    return [...required.mappings, ...proposals.mappings.filter((mapping) => !seen.has(mapping.id))];
  }, [required.mappings, proposals.mappings]);
  const checked = checkedMappingConfirmations(rows, overrides);
  const checkedIds = new Set(checked.map((item) => item.id));
  const selected = rows.find((mapping) => mapping.id === selectedId) ?? null;
  const connection = summary.connection;

  if (!connection) {
    return null;
  }

  return (
    <>
      <div className="space-y-1.5">
        <h3 className="text-base font-semibold">{t("Match your records")}</h3>
        <p className="text-foreground-muted text-sm">
          {t(
            "Tell Trenova which {0} account, item, customer and vendor each Trenova record is sent as. Trenova proposes a match where it is sure enough; nothing is used until a person confirms it.",
            vendor.name,
          )}
        </p>
      </div>

      <ReferenceRefreshStatus
        connection={connection}
        providerName={vendor.name}
        canUpdate={canUpdate}
        isRequesting={actions.refreshReference.isPending}
        onRefresh={() => actions.refreshReference.mutate()}
      />

      <div className="space-y-1.5">
        <p id={progressLabelId} className="text-sm font-medium">
          {t(
            "{0} of {1} required mappings confirmed",
            summary.requiredConfirmed,
            summary.requiredTotal,
          )}
        </p>
        <Progress
          size="sm"
          value={summary.requiredConfirmed}
          max={Math.max(summary.requiredTotal, 1)}
          variant={
            summary.requiredTotal > 0 && summary.requiredConfirmed >= summary.requiredTotal
              ? "success"
              : "default"
          }
          aria-labelledby={progressLabelId}
        />
      </div>

      {required.isLoading || proposals.isLoading ? (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : required.isError || proposals.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("The mappings could not be loaded.")}</AlertDescription>
        </Alert>
      ) : (
        <div className="divide-border-subtle divide-y rounded-lg border">
          {rows.map((mapping) => (
            <MappingRow
              key={mapping.id}
              mapping={mapping}
              selected={mapping.id === selectedId}
              checked={checkedIds.has(mapping.id)}
              canCheck={canUpdate}
              onSelect={() => setSelectedId(mapping.id === selectedId ? null : mapping.id)}
              onCheckedChange={(value) =>
                setOverrides((current) => ({ ...current, [mappingCheckKey(mapping)]: value }))
              }
            />
          ))}
        </div>
      )}

      {proposals.hasNextPage ? (
        <p className="text-foreground-muted text-xs">
          {t("More proposals are waiting on the mappings page.")}
        </p>
      ) : null}

      {selected ? (
        <div className="rounded-lg border p-4">
          <MappingEditor
            key={selected.id}
            system={vendor.system}
            providerName={vendor.name}
            mapping={selected}
            canUpdate={canUpdate}
          />
        </div>
      ) : null}

      <IntegrationSetupFooter className="justify-between">
        <Button
          variant="link"
          size="sm"
          nativeButton={false}
          render={<Link to={ACCOUNTING_MAPPINGS_PATH} />}
        >
          {t("Open all mappings")}
        </Button>
        {canUpdate ? (
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              isLoading={actions.confirm.isPending}
              disabled={checked.length === 0}
              onClick={() => actions.confirm.mutate(checked, { onSuccess: () => setOverrides({}) })}
            >
              {t("Confirm {0} checked", checked.length)}
            </Button>
            <Button
              type="button"
              isLoading={actions.completeSetup.isPending}
              disabled={!summary.canCompleteSetup}
              onClick={() => actions.completeSetup.mutate()}
            >
              {t("Finish setup")}
            </Button>
          </div>
        ) : null}
      </IntegrationSetupFooter>
    </>
  );
}
