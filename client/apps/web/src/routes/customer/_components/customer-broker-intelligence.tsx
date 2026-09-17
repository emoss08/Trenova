import { useT } from "@trenova/shared/i18n/use-t";
import {
  CarrierIntelProfileView,
  type CarrierIntelProfileSectionId,
} from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { DecisionSummary } from "@/components/carrier-intelligence/decision-summary";
import { IntelEmptySketch } from "@/components/carrier-intelligence/intel-empty-sketch";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { IntelSnapshotHeader } from "@/components/carrier-intelligence/intel-snapshot-header";
import { useCarrierIntelRuleLabels } from "@/components/carrier-intelligence/use-carrier-intel-rule-labels";
import { StatusDot } from "@/components/carrier-intelligence/status-dot";
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
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { Customer } from "@trenova/shared/types/customer";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlugZapIcon, ScanSearchIcon } from "lucide-react";
import { useCallback } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";

const BROKER_PROFILE_SECTIONS = [
  "company",
  "authority",
  "insurance",
  "network",
] as const satisfies readonly CarrierIntelProfileSectionId[];

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
    <p className="text-muted-foreground flex items-center gap-2 text-xs">
      <StatusDot tone="medium" />
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

  const ruleLabels = useCarrierIntelRuleLabels(canRead);

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
      <div className="flex flex-col gap-4" aria-busy>
        <div className="flex flex-col gap-2 border-b pb-4">
          <Skeleton className="h-3.5 w-80" />
          <Skeleton className="h-3 w-56" />
        </div>
        <div className="flex flex-col gap-3">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    );
  }

  const loadError = intelQuery.error ?? settingsQuery.error;
  if (loadError) {
    return (
      <IntelInlineError
        error={loadError}
        title={t("Broker vetting could not be loaded")}
        onRetry={() => {
          void intelQuery.refetch();
          void settingsQuery.refetch();
        }}
      />
    );
  }

  const customer = intelQuery.data;

  if (!customer) {
    return (
      <p className="text-muted-foreground text-sm">
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
        <p className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-xs">
          <PlugZapIcon className="size-3.5" aria-hidden />
          {t(
            "No provider is connected, so this is the last snapshot on file and cannot be refreshed.",
          )}
          <Link
            to={CARRIER_INTEL_INTEGRATIONS_PATH}
            className="text-foreground underline-offset-2 hover:underline"
          >
            {t("Open integrations")}
          </Link>
        </p>
      ) : null}
      <IntelSnapshotHeader
        label={t("Broker vetting summary")}
        title={snapshot.profile.identity?.legalName ?? t("Broker vetting")}
        riskLevel={snapshot.riskLevel}
        reviewState={snapshot.reviewState}
        reviewedAt={snapshot.reviewedAt}
        blockingCount={snapshot.blockingCodes.length}
        freshness={{
          effectiveAsOf: snapshot.effectiveAsOf,
          fetchedAt: snapshot.fetchedAt,
          confirmedAt: snapshot.confirmedAt,
          depth: snapshot.depth,
          depthFetchedAt: snapshot.depthFetchedAt,
          fetchedDepth: snapshot.fetchedDepth,
          sourceAsOf: snapshot.sourceAsOf,
          staleAfterHours: control?.preTenderMaxAgeHours,
          expiredAfterHours: control?.hardMaxAgeHours,
        }}
        meta={t(
          "USDOT {0} via {1}",
          snapshot.dotNumber ?? customer.dotNumber,
          carrierIntelProviderLabel(snapshot.provider),
        )}
        note={snapshot.reviewNote ? t("Review note: {0}", snapshot.reviewNote) : null}
        actions={
          canVet ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              isLoading={vet.isPending}
              onClick={() => vet.mutate(true)}
            >
              {t("Vet now")}
            </Button>
          ) : null
        }
      />
      <DecisionSummary
        findings={snapshot.findings}
        ruleLabels={ruleLabels}
        notFound={snapshot.notFound}
        provider={snapshot.provider}
      />
      {!snapshot.notFound ? (
        <CarrierIntelProfileView
          profile={snapshot.profile}
          provider={snapshot.provider}
          sections={BROKER_PROFILE_SECTIONS}
        />
      ) : null}
    </div>
  );
}
