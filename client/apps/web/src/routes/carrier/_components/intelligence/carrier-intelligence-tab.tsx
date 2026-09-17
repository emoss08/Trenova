import { useT } from "@trenova/shared/i18n/use-t";
import { CarrierIntelProfileView } from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { CarrierKeyFacts } from "@/components/carrier-intelligence/carrier-key-facts";
import { SectionNote } from "@/components/carrier-intelligence/intel-facts";
import { IntelEmptySketch } from "@/components/carrier-intelligence/intel-empty-sketch";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { useCarrierIntelRuleLabels } from "@/components/carrier-intelligence/use-carrier-intel-rule-labels";
import { usePermission } from "@/hooks/use-permission";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTELLIGENCE_KEY,
  CARRIER_INTEL_EVENTS_KEY,
  CARRIER_INTEL_HISTORY_KEY,
  CARRIER_INTEL_OVERRIDES_KEY,
  CARRIER_INTEL_SYNC_PLAN_KEY,
  fetchCarrierIntelligence,
} from "@/lib/graphql/carrier-intelligence";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlugZapIcon, ScanSearchIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useCallback, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { IntelligenceEvents } from "./intelligence-events";
import { IntelligenceFindings } from "./intelligence-findings";
import { IntelligenceHeader } from "./intelligence-header";
import { IntelligenceHistory } from "./intelligence-history";
import { IntelligenceSyncPanel } from "./intelligence-sync-panel";
import { MarkReviewedDialog } from "@/components/carrier-intelligence/mark-reviewed-dialog";
import { useRefreshCarrierForm } from "./use-refresh-carrier-form";
import { VetCarrierDialog } from "./vet-carrier-dialog";

export type CarrierIntelligenceTabProps = {
  carrierId: string;
};

const INTEL_VIEWS = ["findings", "profile", "sync", "timeline", "history"] as const;
type IntelView = (typeof INTEL_VIEWS)[number];

const intelViewParser = parseAsStringLiteral(INTEL_VIEWS).withDefault("findings");

export function CarrierIntelligenceTab({ carrierId }: CarrierIntelligenceTabProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const refreshCarrierForm = useRefreshCarrierForm(carrierId);
  const [view, setView] = useQueryState("intelView", intelViewParser);
  const [vetOpen, setVetOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);

  const { allowed: canRead, isLoading: permissionsLoading } = usePermission(
    Resource.CarrierIntelligence,
    Operation.Read,
  );
  const { allowed: canUpdate } = usePermission(Resource.CarrierIntelligence, Operation.Update);
  const { allowed: canApprove } = usePermission(Resource.CarrierIntelligence, Operation.Approve);
  const { allowed: canExport } = usePermission(Resource.CarrierIntelligence, Operation.Export);
  const { allowed: canUpdateCarrier } = usePermission(Resource.Carrier, Operation.Update);

  const intelQuery = useQuery({
    queryKey: [CARRIER_INTELLIGENCE_KEY, carrierId],
    queryFn: ({ signal }) => fetchCarrierIntelligence(carrierId, { signal }),
    enabled: canRead,
  });

  const settingsQuery = useQuery({
    ...queries.carrierIntelSettings.settings(),
    enabled: canRead,
  });

  const ruleLabels = useCarrierIntelRuleLabels(canRead);

  const invalidateIntel = useCallback(() => {
    for (const key of [
      CARRIER_INTELLIGENCE_KEY,
      CARRIER_INTEL_HISTORY_KEY,
      CARRIER_INTEL_EVENTS_KEY,
      CARRIER_INTEL_SYNC_PLAN_KEY,
      CARRIER_INTEL_OVERRIDES_KEY,
    ]) {
      void queryClient.invalidateQueries({ queryKey: [key, carrierId] });
    }
    void queryClient.invalidateQueries({ queryKey: ["carrier-list"] });
  }, [carrierId, queryClient]);

  const handleCarrierChanged = useCallback(() => {
    invalidateIntel();
    void refreshCarrierForm();
  }, [invalidateIntel, refreshCarrierForm]);

  if (permissionsLoading) {
    return <Skeleton className="h-40 w-full" />;
  }

  if (!canRead) {
    return (
      <EmptySheet
        title={t("Carrier intelligence is restricted")}
        description={t(
          "Your role cannot view carrier intelligence. Ask an administrator for read access to Carrier Intelligence.",
        )}
        sketch={<IntelEmptySketch />}
      />
    );
  }

  if (intelQuery.isPending) {
    return (
      <div className="flex flex-col gap-4" aria-busy>
        <div className="flex flex-col gap-2 border-b pb-4">
          <Skeleton className="h-3.5 w-80" />
          <Skeleton className="h-3 w-56" />
        </div>
        <Skeleton className="h-8 w-96" />
        <div className="flex flex-col gap-3">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    );
  }

  if (intelQuery.isError) {
    return (
      <IntelInlineError
        error={intelQuery.error}
        title={t("Carrier intelligence could not be loaded")}
        onRetry={() => void intelQuery.refetch()}
      />
    );
  }

  const { carrier, carrierIntelProvider: provider } = intelQuery.data;

  if (!carrier) {
    return (
      <p className="text-muted-foreground text-sm">
        {t("This carrier could not be found. It may have been removed.")}
      </p>
    );
  }

  const snapshot = carrier.intelligence;
  const providerName = snapshot?.provider ?? provider.provider;

  if (!provider.configured && !snapshot) {
    return (
      <EmptySheet
        title={t("No carrier intelligence provider is connected")}
        description={t(
          "Connect CarrierOK or the free FMCSA QCMobile service to vet carriers, watch them for authority, insurance and safety changes, and block risky tenders.",
        )}
        sketch={<IntelEmptySketch />}
        action={
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to="/admin/integrations?category=CarrierCompliance" />}
          >
            <PlugZapIcon className="size-3.5" />
            {t("Open integrations")}
          </Button>
        }
      />
    );
  }

  const vetDialog = (
    <VetCarrierDialog
      carrierId={carrierId}
      provider={provider}
      lastDepth={snapshot?.depth ?? null}
      open={vetOpen}
      onOpenChange={setVetOpen}
      onVetted={handleCarrierChanged}
    />
  );

  if (!snapshot) {
    return (
      <>
        <EmptySheet
          title={t("This carrier has not been vetted")}
          description={
            carrier.dotNumber
              ? t(
                  "Vet USDOT {0} against {1} to see its authority, insurance, safety record and any findings that would block a tender.",
                  carrier.dotNumber,
                  carrierIntelProviderLabel(providerName),
                )
              : t(
                  "Add the carrier's USDOT number on the Identity tab and save, then vet it to see its authority, insurance and safety record.",
                )
          }
          sketch={<IntelEmptySketch />}
          action={
            canUpdate && carrier.dotNumber ? (
              <Button type="button" size="sm" onClick={() => setVetOpen(true)}>
                <ScanSearchIcon className="size-3.5" />
                {t("Vet carrier")}
              </Button>
            ) : undefined
          }
        />
        {vetDialog}
      </>
    );
  }

  const blockerCount = snapshot.blockingCodes.length;
  const findingCount = snapshot.findings.filter((finding) => finding.action !== "Off").length;
  const quietCount = (count: number): ReactNode =>
    count > 0 ? (
      <span className="text-muted-foreground text-xs font-normal tabular-nums">{count}</span>
    ) : null;

  return (
    <div className="flex flex-col gap-4">
      {!provider.configured ? (
        <p className="text-muted-foreground flex flex-wrap items-center gap-x-1.5 text-xs">
          <PlugZapIcon className="size-3.5" aria-hidden />
          {t(
            "No provider is connected, so this is the last snapshot on file and cannot be refreshed.",
          )}
          <Link
            to="/admin/integrations?category=CarrierCompliance"
            className="text-foreground underline-offset-2 hover:underline"
          >
            {t("Open integrations")}
          </Link>
        </p>
      ) : null}
      <IntelligenceHeader
        carrierId={carrierId}
        snapshot={snapshot}
        enrollment={carrier.monitoringEnrollment}
        openEventCount={carrier.openIntelEventCount}
        provider={provider.provider}
        control={settingsQuery.data?.carrierIntelControl ?? null}
        canUpdate={canUpdate && provider.configured}
        onVet={() => setVetOpen(true)}
        onMarkReviewed={() => setReviewOpen(true)}
        onChanged={invalidateIntel}
      />
      <Tabs value={view} onValueChange={(value) => void setView(value as IntelView)}>
        <TabsList variant="underline" className="w-full justify-start overflow-x-auto border-b">
          <TabsTab value="findings" className="grow-0 px-2 text-sm">
            {t("Findings")}
            {quietCount(findingCount)}
          </TabsTab>
          <TabsTab value="profile" className="grow-0 px-2 text-sm">
            {t("Profile")}
          </TabsTab>
          <TabsTab value="sync" className="grow-0 px-2 text-sm">
            {t("Sync")}
          </TabsTab>
          <TabsTab value="timeline" className="grow-0 px-2 text-sm">
            {t("Timeline")}
            {quietCount(carrier.openIntelEventCount)}
          </TabsTab>
          <TabsTab value="history" className="grow-0 px-2 text-sm">
            {t("History")}
          </TabsTab>
        </TabsList>
        <TabsContent value="findings" className="pt-3">
          <IntelligenceFindings
            carrierId={carrierId}
            findings={snapshot.findings}
            notFound={snapshot.notFound}
            provider={snapshot.provider}
            ruleLabels={ruleLabels}
            canApprove={canApprove}
            onChanged={invalidateIntel}
          />
        </TabsContent>
        <TabsContent value="profile" className="pt-3">
          {snapshot.notFound ? (
            <SectionNote>
              {t(
                "{0} has no record for USDOT {1}, so there is no profile to show.",
                carrierIntelProviderLabel(snapshot.provider),
                snapshot.dotNumber,
              )}
            </SectionNote>
          ) : (
            <div className="flex flex-col gap-6">
              <CarrierKeyFacts profile={snapshot.profile} />
              <CarrierIntelProfileView
                profile={snapshot.profile}
                provider={snapshot.provider}
                showCoverageSummary
              />
            </div>
          )}
        </TabsContent>
        <TabsContent value="sync" className="pt-3">
          <IntelligenceSyncPanel
            carrierId={carrierId}
            canApply={canUpdate && canUpdateCarrier}
            onApplied={handleCarrierChanged}
          />
        </TabsContent>
        <TabsContent value="timeline" className="pt-3">
          <IntelligenceEvents
            carrierId={carrierId}
            canUpdate={canUpdate}
            onChanged={invalidateIntel}
          />
        </TabsContent>
        <TabsContent value="history" className="pt-3">
          <IntelligenceHistory
            carrierId={carrierId}
            currentSnapshotId={snapshot.id}
            ruleLabels={ruleLabels}
            canExport={canExport}
          />
        </TabsContent>
      </Tabs>
      {vetDialog}
      <MarkReviewedDialog
        carrierId={carrierId}
        blockingCount={blockerCount}
        open={reviewOpen}
        onOpenChange={setReviewOpen}
        onReviewed={invalidateIntel}
      />
    </div>
  );
}
