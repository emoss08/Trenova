import { useT } from "@trenova/shared/i18n/use-t";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { formatOptionalDecimalCurrency } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTEL_SYNC_PLAN_KEY,
  applyCarrierIntelSuggestions,
  fetchCarrierIntelSyncPlan,
  type CarrierIntelFieldUpdate,
  type CarrierIntelInsuranceChange,
} from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelSyncField } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, CheckCircle2Icon, RefreshCwIcon, WandSparklesIcon } from "lucide-react";
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
      <span className="bg-muted text-muted-foreground rounded px-1.5 py-0.5 line-through decoration-1">
        {current}
      </span>
      <ArrowRightIcon className="text-muted-foreground size-3" aria-hidden />
      <span className="bg-muted rounded px-1.5 py-0.5 font-medium">{proposed}</span>
    </span>
  );
}

function InsuranceChangeDetail({ change }: { change: CarrierIntelInsuranceChange }) {
  const t = useT();
  const empty = t("empty");

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

  const apply = useApiMutation<number, void>({
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
    return <Skeleton className="h-32 w-full" />;
  }

  if (planQuery.isError) {
    return (
      <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
        <span>{t("Sync suggestions could not be loaded. {0}", planQuery.error.message)}</span>
        <Button type="button" size="xs" variant="outline" onClick={() => void planQuery.refetch()}>
          <RefreshCwIcon />
          {t("Retry")}
        </Button>
      </div>
    );
  }

  const { data: loadedPlan } = planQuery;

  const autoApplied: { key: string; label: string; detail: ReactNode; reason: string }[] = [
    ...loadedPlan.autoApply.map((update: CarrierIntelFieldUpdate) => ({
      key: `field-${update.field}`,
      label: labels.syncField[update.field],
      detail: <ValueChange current={update.current || t("empty")} proposed={update.proposed} />,
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
      <div className="text-muted-foreground flex items-center gap-2 rounded-lg border border-dashed px-3 py-4 text-sm">
        <CheckCircle2Icon className="size-4 text-green-600" aria-hidden />
        {t("The carrier record matches the latest vetting. Nothing to sync.")}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {selectableCount > 0 ? (
        <section aria-label={t("Suggested updates")} className="flex flex-col gap-2">
          <header className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              {canApply ? (
                <Checkbox
                  aria-label={t("Select all suggestions")}
                  checked={selectedCount > 0 && selectedCount === selectableCount}
                  indeterminate={selectedCount > 0 && selectedCount < selectableCount}
                  onCheckedChange={(checked) => selectAll(checked)}
                />
              ) : null}
              <h4 className="text-sm font-medium">{t("Suggested updates")}</h4>
              <Badge variant="outline" className="tabular-nums">
                {selectableCount}
              </Badge>
            </div>
            {canApply ? (
              <Button
                type="button"
                size="sm"
                disabled={selectedCount === 0}
                isLoading={apply.isPending}
                onClick={() => apply.mutate()}
              >
                <WandSparklesIcon />
                {t("Apply {0, plural, one {# change} other {# changes}}", selectedCount)}
              </Button>
            ) : (
              <span className="text-muted-foreground text-xs">
                {t("Updating carriers requires carrier update permission.")}
              </span>
            )}
          </header>
          <ul className="bg-card divide-y rounded-lg border">
            {loadedPlan.suggestions.map((update) => {
              const checked = selectedFields.has(update.field);
              return (
                <li
                  key={update.field}
                  className={cn("flex items-start gap-3 px-3 py-2", checked && "bg-muted/40")}
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
                    <span className="text-sm font-medium">{labels.syncField[update.field]}</span>
                    <ValueChange
                      current={update.current || t("empty")}
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
                  className={cn("flex items-start gap-3 px-3 py-2", checked && "bg-muted/40")}
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
                    <span className="text-sm font-medium">
                      {labels.policyType[change.policyType]} · {change.policyNumber}
                      <span className="text-muted-foreground font-normal">
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
        <section aria-label={t("New filings")} className="flex flex-col gap-2">
          <h4 className="text-sm font-medium">{t("New filings")}</h4>
          <p className="text-muted-foreground text-xs">
            {t(
              "The provider reports policies this carrier has no record of. Add them on the Compliance & Insurance tab after confirming the certificate.",
            )}
          </p>
          <ul className="bg-card divide-y rounded-lg border">
            {manualInsurance.map((change, index) => (
              <li key={`${change.policyNumber}-${index}`} className="flex flex-col gap-1 px-3 py-2">
                <span className="text-sm font-medium">
                  {labels.policyType[change.policyType]} · {change.policyNumber}
                </span>
                <span className="text-xs">
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
        <section aria-label={t("Applied automatically")} className="flex flex-col gap-2">
          <h4 className="text-sm font-medium">{t("Applied automatically")}</h4>
          <p className="text-muted-foreground text-xs">
            {t("Your sync settings apply these changes on every vetting without asking.")}
          </p>
          <ul className="bg-card divide-y rounded-lg border">
            {autoApplied.map((item) => (
              <li key={item.key} className="flex flex-col gap-1 px-3 py-2">
                <span className="text-sm font-medium">{item.label}</span>
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
