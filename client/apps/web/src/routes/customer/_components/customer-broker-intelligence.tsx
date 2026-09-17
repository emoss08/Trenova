import { useT } from "@trenova/shared/i18n/use-t";
import { CarrierIntelProfileView } from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { FindingList } from "@/components/carrier-intelligence/finding-list";
import { FreshnessIndicator } from "@/components/carrier-intelligence/freshness-indicator";
import { IntelEmptySketch } from "@/components/carrier-intelligence/intel-empty-sketch";
import { ReviewStateBadge } from "@/components/carrier-intelligence/review-state-badge";
import { RiskLevelBadge } from "@/components/carrier-intelligence/risk-level-badge";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import {
  CUSTOMER_BROKER_INTELLIGENCE_KEY,
  fetchCustomerBrokerIntelligence,
  vetCustomerBroker,
  type CustomerBrokerIntelligence as CustomerBrokerIntelligenceData,
  type CustomerBrokerVetResult,
} from "@/lib/graphql/carrier-intelligence";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { Customer } from "@trenova/shared/types/customer";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlugZapIcon, RadarIcon, RefreshCwIcon, ScanSearchIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";

const BROKER_PROFILE_CARDS = ["identity", "authority", "insurance", "network"] as const;

type SavedBrokerSettings = {
  dotNumber: string | null;
  brokerVettingEnabled: boolean;
};

function useUnsavedBrokerSettings(server: SavedBrokerSettings): boolean {
  const { control } = useFormContext<Customer>();
  const dotNumber = useWatch({ control, name: "dotNumber" });
  const brokerVettingEnabled = useWatch({ control, name: "brokerVettingEnabled" });
  return (
    (dotNumber || null) !== (server.dotNumber || null) ||
    brokerVettingEnabled !== server.brokerVettingEnabled
  );
}

function UnsavedNotice({ server }: { server: SavedBrokerSettings }) {
  const t = useT();
  const unsaved = useUnsavedBrokerSettings(server);

  if (!unsaved) {
    return null;
  }

  return (
    <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-2 text-xs">
      {t(
        "The DOT number or broker vetting setting has unsaved changes. Save the customer before vetting so the new values are used.",
      )}
    </p>
  );
}

export function CustomerBrokerIntelligence({ customerId }: { customerId: string }) {
  const t = useT();
  const queryClient = useQueryClient();

  const { allowed: canRead, isLoading: permissionsLoading } = usePermission(
    Resource.CarrierIntelligence,
    Operation.Read,
  );
  const { allowed: canUpdate } = usePermission(Resource.CarrierIntelligence, Operation.Update);

  const intelQuery = useQuery({
    queryKey: [CUSTOMER_BROKER_INTELLIGENCE_KEY, customerId],
    queryFn: ({ signal }) => fetchCustomerBrokerIntelligence(customerId, { signal }),
    enabled: canRead,
  });

  const settingsQuery = useQuery({
    ...queries.carrierIntelSettings.settings(),
    enabled: canRead,
  });

  const ruleLabels = useMemo<Record<string, string>>(
    () =>
      Object.fromEntries(
        (settingsQuery.data?.carrierIntelRuleCatalog ?? []).map((rule) => [rule.code, rule.label]),
      ),
    [settingsQuery.data],
  );

  const handleVetted = useCallback(
    (result: CustomerBrokerVetResult) => {
      queryClient.setQueryData(
        [CUSTOMER_BROKER_INTELLIGENCE_KEY, customerId],
        (current: CustomerBrokerIntelligenceData | null | undefined) =>
          current ? { ...current, brokerIntelligence: result.snapshot } : current,
      );
      void queryClient.invalidateQueries({
        queryKey: [CUSTOMER_BROKER_INTELLIGENCE_KEY, customerId],
      });
    },
    [customerId, queryClient],
  );

  const vet = useApiMutation<CustomerBrokerVetResult, boolean>({
    resourceName: "Broker vetting",
    mutationFn: (force) => vetCustomerBroker(customerId, force),
    onSuccess: (result) => {
      const details: string[] = [];
      if (result.fromCache) {
        details.push(t("A fresh snapshot was already on file, so no provider call was made."));
      }
      if (result.usedFallback) {
        details.push(t("The primary provider was unavailable; the fallback provider answered."));
      }
      if (result.changeCount > 0 || result.raisedCount > 0) {
        details.push(
          t(
            "{0, plural, one {# change} other {# changes}} detected, {1, plural, one {# event} other {# events}} raised.",
            result.changeCount,
            result.raisedCount,
          ),
        );
      }
      toast.success(result.fromCache ? t("Vetting is current") : t("Broker vetted"), {
        description: details.length > 0 ? details.join(" ") : undefined,
      });
      handleVetted(result);
    },
  });

  if (permissionsLoading) {
    return <Skeleton className="h-40 w-full" />;
  }

  if (!canRead) {
    return (
      <EmptySheet
        title={t("Broker vetting is restricted")}
        description={t(
          "Your role cannot view carrier intelligence. Ask an administrator for read access to Carrier Intelligence.",
        )}
        sketch={<IntelEmptySketch />}
      />
    );
  }

  if (intelQuery.isPending || settingsQuery.isPending) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  const loadError = intelQuery.error ?? settingsQuery.error;
  if (loadError) {
    return (
      <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
        <span>{t("Broker vetting could not be loaded. {0}", loadError.message)}</span>
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={() => {
            void intelQuery.refetch();
            void settingsQuery.refetch();
          }}
        >
          <RefreshCwIcon />
          {t("Retry")}
        </Button>
      </div>
    );
  }

  const customer = intelQuery.data;

  if (!customer) {
    return (
      <p className="text-muted-foreground rounded-lg border border-dashed p-3 text-sm">
        {t("This customer could not be found. It may have been removed.")}
      </p>
    );
  }

  const provider = settingsQuery.data?.carrierIntelProvider ?? null;
  const snapshot = customer.brokerIntelligence;
  const server: SavedBrokerSettings = {
    dotNumber: customer.dotNumber,
    brokerVettingEnabled: customer.brokerVettingEnabled,
  };

  if (!customer.brokerVettingEnabled) {
    return (
      <div className="flex flex-col gap-3">
        <UnsavedNotice server={server} />
        <EmptySheet
          title={t("Broker vetting is off for this customer")}
          description={t(
            "Turn on Vet as a broker on the General tab and save to check this customer's broker authority, surety bond and insurance.",
          )}
          sketch={<IntelEmptySketch />}
        />
      </div>
    );
  }

  if (!customer.dotNumber) {
    return (
      <div className="flex flex-col gap-3">
        <UnsavedNotice server={server} />
        <EmptySheet
          title={t("Add the customer's DOT number")}
          description={t(
            "Broker vetting looks the customer up by USDOT number. Enter it on the General tab and save, then vet the broker.",
          )}
          sketch={<IntelEmptySketch />}
        />
      </div>
    );
  }

  if (!provider?.configured && !snapshot) {
    return (
      <EmptySheet
        title={t("No carrier intelligence provider is connected")}
        description={t(
          "Connect CarrierOK or the free FMCSA QCMobile service to vet brokers for authority, bond and insurance.",
        )}
        sketch={<IntelEmptySketch />}
        action={
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
          >
            <PlugZapIcon className="size-3.5" />
            {t("Open integrations")}
          </Button>
        }
      />
    );
  }

  const canVet = canUpdate && !!provider?.configured;

  if (!snapshot) {
    return (
      <div className="flex flex-col gap-3">
        <UnsavedNotice server={server} />
        <EmptySheet
          title={t("This broker has not been vetted")}
          description={t(
            "Vet USDOT {0} against {1} to see its broker authority, surety bond, insurance and any findings.",
            customer.dotNumber,
            carrierIntelProviderLabel(provider?.provider),
          )}
          sketch={<IntelEmptySketch />}
          action={
            canVet ? (
              <Button
                type="button"
                size="sm"
                isLoading={vet.isPending}
                onClick={() => vet.mutate(false)}
              >
                <ScanSearchIcon className="size-3.5" />
                {t("Vet broker")}
              </Button>
            ) : undefined
          }
        />
      </div>
    );
  }

  const control = settingsQuery.data?.carrierIntelControl ?? null;

  return (
    <div className="flex flex-col gap-4">
      <UnsavedNotice server={server} />
      {!provider?.configured ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-2 text-xs">
          {t(
            "No provider is connected right now, so this is the last snapshot on file and cannot be refreshed.",
          )}{" "}
          <Link to={CARRIER_INTEL_INTEGRATIONS_PATH} className="underline">
            {t("Open integrations")}
          </Link>
        </p>
      ) : null}
      <section
        aria-label={t("Broker vetting summary")}
        className="bg-card flex flex-wrap items-start justify-between gap-3 rounded-lg border p-3"
      >
        <div className="flex min-w-0 flex-col gap-1.5">
          <div className="flex flex-wrap items-center gap-1.5">
            <RadarIcon className="text-muted-foreground size-4" aria-hidden />
            <h3 className="text-sm font-semibold">{t("Broker vetting")}</h3>
            <span className="text-muted-foreground text-xs">
              {t(
                "USDOT {0} via {1}",
                snapshot.dotNumber ?? customer.dotNumber,
                carrierIntelProviderLabel(snapshot.provider),
              )}
            </span>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <RiskLevelBadge level={snapshot.riskLevel} />
            <ReviewStateBadge state={snapshot.reviewState} reviewedAt={snapshot.reviewedAt} />
            {snapshot.blockingCodes.length > 0 ? (
              <Badge variant="inactive" className="max-h-5 tabular-nums">
                {t(
                  "{0, plural, one {# blocker} other {# blockers}}",
                  snapshot.blockingCodes.length,
                )}
              </Badge>
            ) : null}
          </div>
          <FreshnessIndicator
            fetchedAt={snapshot.fetchedAt}
            confirmedAt={snapshot.confirmedAt}
            effectiveAsOf={snapshot.effectiveAsOf}
            sourceAsOf={snapshot.sourceAsOf}
            staleAfterHours={control?.preTenderMaxAgeHours}
            expiredAfterHours={control?.hardMaxAgeHours}
          />
          {snapshot.reviewNote ? (
            <p className="text-muted-foreground text-xs">
              {t("Review note: {0}", snapshot.reviewNote)}
            </p>
          ) : null}
        </div>
        {canVet ? (
          <Button
            type="button"
            size="sm"
            isLoading={vet.isPending}
            onClick={() => vet.mutate(true)}
          >
            <ScanSearchIcon />
            {t("Vet now")}
          </Button>
        ) : null}
      </section>
      <section aria-label={t("Findings")} className="flex flex-col gap-2">
        <h4 className="text-sm font-medium">{t("Findings")}</h4>
        <FindingList findings={snapshot.findings} ruleLabels={ruleLabels} />
      </section>
      <CarrierIntelProfileView
        profile={snapshot.profile}
        provider={snapshot.provider}
        notFound={snapshot.notFound}
        cards={BROKER_PROFILE_CARDS}
        columns={1}
      />
    </div>
  );
}
