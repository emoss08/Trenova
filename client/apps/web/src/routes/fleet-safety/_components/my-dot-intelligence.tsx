import { useT } from "@trenova/shared/i18n/use-t";
import { CarrierIntelProfileView } from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { FindingList } from "@/components/carrier-intelligence/finding-list";
import { FreshnessIndicator } from "@/components/carrier-intelligence/freshness-indicator";
import { IntelEmptySketch } from "@/components/carrier-intelligence/intel-empty-sketch";
import { RiskLevelBadge } from "@/components/carrier-intelligence/risk-level-badge";
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
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Building2Icon, PlugZapIcon, RadarIcon, RefreshCwIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

export const MY_DOT_PROFILE_CARDS = [
  "safety",
  "basics",
  "inspections",
  "insurance",
  "authority",
  "contacts",
] as const;

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

  const ruleLabels = useMemo<Record<string, string>>(
    () =>
      Object.fromEntries(
        (settingsQuery.data?.carrierIntelRuleCatalog ?? []).map((rule) => [rule.code, rule.label]),
      ),
    [settingsQuery.data],
  );

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
      <div className="flex flex-col gap-3" data-testid="my-dot-loading">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  const loadError = mineQuery.error ?? settingsQuery.error;
  if (loadError) {
    return (
      <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
        <span>{t("Your DOT profile could not be loaded. {0}", loadError.message)}</span>
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={() => {
            void mineQuery.refetch();
            void settingsQuery.refetch();
          }}
        >
          <RefreshCwIcon />
          {t("Retry")}
        </Button>
      </div>
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
  const findingCount = snapshot.findings.filter((finding) => finding.action !== "Off").length;

  return (
    <div className="flex flex-col gap-4">
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
        aria-label={t("DOT profile summary")}
        className="bg-card flex flex-wrap items-start justify-between gap-3 rounded-lg border p-3"
      >
        <div className="flex min-w-0 flex-col gap-1.5">
          <div className="flex flex-wrap items-center gap-1.5">
            <RadarIcon className="text-muted-foreground size-4" aria-hidden />
            <h3 className="text-sm font-semibold">
              {snapshot.profile.identity?.legalName ?? t("Your DOT profile")}
            </h3>
            <span className="text-muted-foreground text-xs">
              {t(
                "USDOT {0} via {1}",
                snapshot.dotNumber ?? mine.dotNumber,
                carrierIntelProviderLabel(snapshot.provider),
              )}
            </span>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <RiskLevelBadge level={snapshot.riskLevel} />
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
          <p className="text-muted-foreground text-xs">
            {t(
              "This is how a broker or shipper vetting you sees your record. Findings use your organization's own rules.",
            )}
          </p>
        </div>
        {canPull ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            isLoading={refresh.isPending}
            onClick={() => refresh.mutate()}
          >
            <RefreshCwIcon />
            {t("Refresh")}
          </Button>
        ) : null}
      </section>
      <section aria-label={t("Findings")} className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <h4 className="text-sm font-medium">{t("Findings")}</h4>
          {findingCount > 0 ? (
            <Badge variant="secondary" className="max-h-5 tabular-nums">
              {findingCount}
            </Badge>
          ) : null}
        </div>
        <FindingList
          findings={snapshot.findings}
          ruleLabels={ruleLabels}
          emptyMessage={t("No findings. Your record passes every enabled rule.")}
        />
      </section>
      <CarrierIntelProfileView
        profile={snapshot.profile}
        provider={snapshot.provider}
        notFound={snapshot.notFound}
        cards={MY_DOT_PROFILE_CARDS}
      />
    </div>
  );
}
