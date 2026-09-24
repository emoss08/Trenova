import { useT } from "@trenova/shared/i18n/use-t";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { INTEL_EMPTY_VALUE, formatOptionalDecimalCurrency } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTEL_SYNC_PLAN_KEY,
  applyCarrierIntelSuggestions,
  fetchCarrierIntelSyncPlan,
  type CarrierIntelFieldUpdate,
  type CarrierIntelInsuranceChange,
} from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelSyncField } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";

export type IntelligenceSyncPanelProps = {
  carrierId: string;
  canApply: boolean;
  onApplied: () => void;
};

function ValueChange({ current, proposed }: { current: ReactNode; proposed: ReactNode }) {
  return (
    <span className="flex flex-wrap items-center gap-1.5 text-xs">
      <span className="text-muted-foreground">{current}</span>
      <ArrowRightIcon className="text-muted-foreground size-3" aria-hidden />
      <span className="text-foreground">{proposed}</span>
    </span>
  );
}

function InsuranceChangeDetail({ change }: { change: CarrierIntelInsuranceChange }) {
  const empty = INTEL_EMPTY_VALUE;

  if (change.kind === "ShortenExpiration") {
    return (
      <ValueChange
        current={
          change.currentExpirationDate ? formatUnixDateMedium(change.currentExpirationDate) : empty
        }
        proposed={
          change.proposedExpirationDate
            ? formatUnixDateMedium(change.proposedExpirationDate)
            : empty
        }
      />
    );
  }

  return (
    <ValueChange
      current={formatOptionalDecimalCurrency(change.currentCoverage) ?? empty}
      proposed={formatOptionalDecimalCurrency(change.proposedCoverage) ?? empty}
    />
  );
}

export function IntelligenceSyncPanel({
  carrierId,
  canApply,
  onApplied,
}: IntelligenceSyncPanelProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const [selectedFields, setSelectedFields] = useState<Set<CarrierIntelSyncField>>(new Set());
  const [selectedPolicies, setSelectedPolicies] = useState<Set<string>>(new Set());

  const planQuery = useQuery({
    queryKey: [CARRIER_INTEL_SYNC_PLAN_KEY, carrierId],
    queryFn: ({ signal }) => fetchCarrierIntelSyncPlan(carrierId, { signal }),
  });
  const plan = planQuery.data;

  useEffect(() => {
    setSelectedFields(new Set());
    setSelectedPolicies(new Set());
  }, [plan]);

  const applicableInsurance = useMemo(
    () => (plan?.insuranceSuggestions ?? []).filter((change) => change.policyId !== null),
    [plan],
  );
  const manualInsurance = useMemo(
    () => (plan?.insuranceSuggestions ?? []).filter((change) => change.policyId === null),
    [plan],
  );
  const selectableCount = (plan?.suggestions.length ?? 0) + applicableInsurance.length;
  const selectedCount = selectedFields.size + selectedPolicies.size;

  const apply = useApiMutation<number, undefined>({
    resourceName: "Carrier intelligence suggestions",
    mutationFn: () =>
      applyCarrierIntelSuggestions({
        carrierId,
        fields: selectedFields.size > 0 ? [...selectedFields] : null,
        policyIds: selectedPolicies.size > 0 ? [...selectedPolicies] : null,
      }),
    onSuccess: (count) => {
      toast.success(
        t("{0, plural, one {# change applied} other {# changes applied}} to the carrier", count),
      );
      void planQuery.refetch();
      onApplied();
    },
  });

  const toggleField = (field: CarrierIntelSyncField, checked: boolean) => {
    setSelectedFields((previous) => {
      const next = new Set(previous);
      if (checked) {
        next.add(field);
      } else {
        next.delete(field);
      }
      return next;
    });
  };

  const togglePolicy = (policyId: string, checked: boolean) => {
    setSelectedPolicies((previous) => {
      const next = new Set(previous);
      if (checked) {
        next.add(policyId);
      } else {
        next.delete(policyId);
      }
      return next;
    });
  };

  const selectAll = (checked: boolean) => {
    setSelectedFields(checked ? new Set(plan?.suggestions.map((s) => s.field)) : new Set());
    setSelectedPolicies(
      checked
        ? new Set(applicableInsurance.map((change) => change.policyId).filter((id) => id !== null))
        : new Set(),
    );
  };

  if (planQuery.isPending) {
    return (
      <div className="divide-border flex flex-col divide-y" aria-busy>
        {[0, 1, 2].map((index) => (
          <div key={index} className="flex flex-col gap-1.5 py-2.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-3 w-56" />
          </div>
        ))}
      </div>
    );
  }

  if (planQuery.isError) {
    return (
      <IntelInlineError
        error={planQuery.error}
        title={t("Sync suggestions could not be loaded")}
        onRetry={() => void planQuery.refetch()}
      />
    );
  }

  const { data: loadedPlan } = planQuery;

  const autoApplied: { key: string; label: string; detail: ReactNode; reason: string }[] = [
    ...loadedPlan.autoApply.map((update: CarrierIntelFieldUpdate) => ({
      key: `field-${update.field}`,
      label: labels.syncField[update.field],
      detail: (
        <ValueChange current={update.current || INTEL_EMPTY_VALUE} proposed={update.proposed} />
      ),
      reason: update.reason,
    })),
    ...loadedPlan.insuranceAutoApply.map((change, index) => ({
      key: `policy-${change.policyId ?? index}`,
      label: `${labels.policyType[change.policyType]} · ${labels.insuranceChangeKind[change.kind]}`,
      detail: <InsuranceChangeDetail change={change} />,
      reason: change.reason,
    })),
  ];

  const nothingToDo =
    selectableCount === 0 && manualInsurance.length === 0 && autoApplied.length === 0;

  if (nothingToDo) {
    return (
      <p className="text-muted-foreground py-6 text-center text-xs">
        {t("The carrier record matches the latest vetting. Nothing to sync.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {selectableCount > 0 ? (
        <section aria-label={t("Suggested updates")} className="flex flex-col">
          <header className="flex min-h-8 flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              {canApply ? (
                <Checkbox
                  aria-label={t("Select all suggestions")}
                  checked={selectedCount > 0 && selectedCount === selectableCount}
                  indeterminate={selectedCount > 0 && selectedCount < selectableCount}
                  onCheckedChange={(checked) => selectAll(checked)}
                />
              ) : null}
              <h3 className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
                {t("Suggested updates")}
                <span className="tabular-nums">{selectableCount}</span>
              </h3>
            </div>
            {canApply ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={selectedCount === 0}
                isLoading={apply.isPending}
                onClick={() => apply.mutate(undefined)}
              >
                {t("Apply {0, plural, one {# change} other {# changes}}", selectedCount)}
              </Button>
            ) : (
              <span className="text-muted-foreground text-xs">
                {t("Updating carriers requires carrier update permission.")}
              </span>
            )}
          </header>
          <ul className="divide-border divide-y">
            {loadedPlan.suggestions.map((update) => {
              const checked = selectedFields.has(update.field);
              return (
                <li
                  key={update.field}
                  className={cn(
                    "-mx-2 flex items-start gap-3 rounded-md px-2 py-2.5",
                    checked && "bg-muted/40",
                  )}
                >
                  {canApply ? (
                    <Checkbox
                      className="mt-0.5"
                      aria-label={t("Apply {0}", labels.syncField[update.field])}
                      checked={checked}
                      onCheckedChange={(next) => toggleField(update.field, next)}
                    />
                  ) : null}
                  <div className="flex min-w-0 flex-col gap-1">
                    <span className="text-sm">{labels.syncField[update.field]}</span>
                    <ValueChange
                      current={update.current || INTEL_EMPTY_VALUE}
                      proposed={update.proposed}
                    />
                    <span className="text-muted-foreground text-xs">{update.reason}</span>
                  </div>
                </li>
              );
            })}
            {applicableInsurance.map((change) => {
              const policyId = change.policyId ?? "";
              const checked = selectedPolicies.has(policyId);
              return (
                <li
                  key={policyId}
                  className={cn(
                    "-mx-2 flex items-start gap-3 rounded-md px-2 py-2.5",
                    checked && "bg-muted/40",
                  )}
                >
                  {canApply ? (
                    <Checkbox
                      className="mt-0.5"
                      aria-label={t("Apply {0}", change.policyNumber)}
                      checked={checked}
                      onCheckedChange={(next) => togglePolicy(policyId, next)}
                    />
                  ) : null}
                  <div className="flex min-w-0 flex-col gap-1">
                    <span className="text-sm">
                      {labels.policyType[change.policyType]} · {change.policyNumber}
                      <span className="text-muted-foreground">
                        {" "}
                        · {labels.insuranceChangeKind[change.kind]}
                      </span>
                    </span>
                    <InsuranceChangeDetail change={change} />
                    <span className="text-muted-foreground text-xs">{change.reason}</span>
                  </div>
                </li>
              );
            })}
          </ul>
        </section>
      ) : null}
      {manualInsurance.length > 0 ? (
        <section aria-label={t("New filings")} className="flex flex-col gap-1">
          <h3 className="text-muted-foreground text-xs font-medium">{t("New filings")}</h3>
          <p className="text-muted-foreground text-xs">
            {t(
              "The provider reports policies this carrier has no record of. Add them on the Compliance & Insurance tab after confirming the certificate.",
            )}
          </p>
          <ul className="divide-border divide-y">
            {manualInsurance.map((change, index) => (
              <li key={`${change.policyNumber}-${index}`} className="flex flex-col gap-0.5 py-2.5">
                <span className="text-sm">
                  {labels.policyType[change.policyType]} · {change.policyNumber}
                </span>
                <span className="text-muted-foreground text-xs">
                  {change.providerName}
                  {change.proposedCoverage
                    ? ` · ${formatOptionalDecimalCurrency(change.proposedCoverage) ?? ""}`
                    : null}
                  {change.effectiveDate
                    ? ` · ${t("effective {0}", formatUnixDateMedium(change.effectiveDate))}`
                    : null}
                </span>
                <span className="text-muted-foreground text-xs">{change.reason}</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {autoApplied.length > 0 ? (
        <section aria-label={t("Applied automatically")} className="flex flex-col gap-1">
          <h3 className="text-muted-foreground text-xs font-medium">
            {t("Applied automatically")}
          </h3>
          <p className="text-muted-foreground text-xs">
            {t("Your sync settings apply these changes on every vetting without asking.")}
          </p>
          <ul className="divide-border divide-y">
            {autoApplied.map((item) => (
              <li key={item.key} className="flex flex-col gap-1 py-2.5">
                <span className="text-sm">{item.label}</span>
                {item.detail}
                <span className="text-muted-foreground text-xs">{item.reason}</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
