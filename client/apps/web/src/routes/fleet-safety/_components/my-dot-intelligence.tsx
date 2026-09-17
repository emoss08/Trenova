import { useT } from "@trenova/shared/i18n/use-t";
import {
  CarrierIntelProfileView,
  type CarrierIntelProfileSectionId,
} from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { CarrierKeyFacts } from "@/components/carrier-intelligence/carrier-key-facts";
import { DecisionSummary } from "@/components/carrier-intelligence/decision-summary";
import { IntelEmptySketch } from "@/components/carrier-intelligence/intel-empty-sketch";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { IntelSnapshotHeader } from "@/components/carrier-intelligence/intel-snapshot-header";
import { useCarrierIntelRuleLabels } from "@/components/carrier-intelligence/use-carrier-intel-rule-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import {
  MY_CARRIER_INTELLIGENCE_KEY,
  fetchMyCarrierIntelligence,
  type MyCarrierIntelligence,
} from "@/lib/graphql/carrier-intelligence";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Building2Icon, PlugZapIcon, RefreshCwIcon } from "lucide-react";
import { Link } from "react-router";
import { toast } from "sonner";

export const MY_DOT_PROFILE_SECTIONS = [
  "safety",
  "insurance",
  "authority",
  "company",
] as const satisfies readonly CarrierIntelProfileSectionId[];

const ORGANIZATION_SETTINGS_HREF = "/admin/organization-settings";

function IntegrationsButton() {
  const t = useT();

  return (
    <Button
      size="sm"
      variant="outline"
      nativeButton={false}
      render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
    >
      <PlugZapIcon className="size-3.5" />
      {t("Open integrations")}
    </Button>
  );
}

export function MyDotIntelligence() {
  const t = useT();
  const queryClient = useQueryClient();

  const { allowed: canRead, isLoading: permissionsLoading } = usePermission(
    Resource.CarrierIntelligence,
    Operation.Read,
  );
  const { allowed: canRefresh } = usePermission(Resource.CarrierIntelligence, Operation.Update);

  const mineQuery = useQuery({
    queryKey: [MY_CARRIER_INTELLIGENCE_KEY],
    queryFn: ({ signal }) => fetchMyCarrierIntelligence(false, { signal }),
    enabled: canRead,
  });

  const settingsQuery = useQuery({
    ...queries.carrierIntelSettings.settings(),
    enabled: canRead,
  });

  const ruleLabels = useCarrierIntelRuleLabels(canRead);

  const refresh = useApiMutation<MyCarrierIntelligence, void>({
    resourceName: "DOT profile refresh",
    mutationFn: () => fetchMyCarrierIntelligence(true),
    onSuccess: (result) => {
      queryClient.setQueryData([MY_CARRIER_INTELLIGENCE_KEY], result);
      toast.success(t("DOT profile refreshed"), {
        description: result.snapshot
          ? t("Pulled from {0}.", carrierIntelProviderLabel(result.snapshot.provider))
          : undefined,
      });
    },
  });

  if (permissionsLoading) {
    return <Skeleton className="h-40 w-full" />;
  }

  if (!canRead) {
    return (
      <EmptySheet
        title={t("Your DOT profile is restricted")}
        description={t(
          "Your role cannot view carrier intelligence. Ask an administrator for read access to Carrier Intelligence.",
        )}
        sketch={<IntelEmptySketch />}
      />
    );
  }

  if (mineQuery.isPending || settingsQuery.isPending) {
    return (
      <div className="flex flex-col gap-4" aria-busy data-testid="my-dot-loading">
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

  const loadError = mineQuery.error ?? settingsQuery.error;
  if (loadError) {
    return (
      <IntelInlineError
        error={loadError}
        title={t("Your DOT profile could not be loaded")}
        onRetry={() => {
          void mineQuery.refetch();
          void settingsQuery.refetch();
        }}
      />
    );
  }

  const mine = mineQuery.data;
  if (!mine) {
    return null;
  }

  const provider = settingsQuery.data?.carrierIntelProvider ?? null;
  const snapshot = mine.snapshot;

  if (!mine.dotNumber) {
    return (
      <EmptySheet
        title={t("Your organization has no DOT number")}
        description={t(
          "Add your USDOT number in organization settings to see your own authority, insurance, safety rating and CSA BASICs the way shippers and brokers see them.",
        )}
        sketch={<IntelEmptySketch />}
        action={
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to={ORGANIZATION_SETTINGS_HREF} />}
          >
            <Building2Icon className="size-3.5" />
            {t("Open organization settings")}
          </Button>
        }
      />
    );
  }

  if (!mine.configured || (!provider?.configured && !snapshot)) {
    return (
      <EmptySheet
        title={t("No carrier intelligence provider is connected")}
        description={t(
          "Connect CarrierOK or the free FMCSA QCMobile service to pull USDOT {0}'s public safety and compliance record.",
          mine.dotNumber,
        )}
        sketch={<IntelEmptySketch />}
        action={<IntegrationsButton />}
      />
    );
  }

  const canPull = canRefresh && !!provider?.configured;

  if (!snapshot) {
    return (
      <EmptySheet
        title={t("Your DOT profile has not been pulled yet")}
        description={t(
          "Pull USDOT {0} from {1} to see your authority, insurance, safety rating, CSA BASICs and inspection history.",
          mine.dotNumber,
          carrierIntelProviderLabel(provider?.provider),
        )}
        sketch={<IntelEmptySketch />}
        action={
          canPull ? (
            <Button
              type="button"
              size="sm"
              isLoading={refresh.isPending}
              onClick={() => refresh.mutate()}
            >
              <RefreshCwIcon className="size-3.5" />
              {t("Pull profile")}
            </Button>
          ) : undefined
        }
      />
    );
  }

  const control = settingsQuery.data?.carrierIntelControl ?? null;

  return (
    <div className="flex flex-col gap-4">
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
        label={t("DOT profile summary")}
        title={snapshot.profile.identity?.legalName ?? t("Your DOT profile")}
        riskLevel={snapshot.riskLevel}
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
          snapshot.dotNumber ?? mine.dotNumber,
          carrierIntelProviderLabel(snapshot.provider),
        )}
        note={t(
          "This is how a broker or shipper vetting you sees your record. Findings use your organization's own rules.",
        )}
        actions={
          canPull ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              isLoading={refresh.isPending}
              onClick={() => refresh.mutate()}
            >
              {t("Refresh")}
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
        <div className="flex flex-col gap-6">
          <CarrierKeyFacts profile={snapshot.profile} />
          <CarrierIntelProfileView
            profile={snapshot.profile}
            provider={snapshot.provider}
            sections={MY_DOT_PROFILE_SECTIONS}
          />
        </div>
      ) : null}
    </div>
  );
}
