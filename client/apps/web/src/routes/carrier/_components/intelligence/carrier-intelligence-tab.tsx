import { useT } from "@trenova/shared/i18n/use-t";
import { CarrierIntelProfileView } from "@/components/carrier-intelligence/carrier-intel-profile-view";
import { IntelEmptySketch } from "@/components/carrier-intelligence/intel-empty-sketch";
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
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlugZapIcon, RefreshCwIcon, ScanSearchIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useCallback, useMemo, useState } from "react";
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

  const ruleLabels = useMemo<Record<string, string>>(
    () =>
      Object.fromEntries(
        (settingsQuery.data?.carrierIntelRuleCatalog ?? []).map((rule) => [rule.code, rule.label]),
      ),
    [settingsQuery.data],
  );

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
      <div className="flex flex-col gap-3">
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-8 w-72" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (intelQuery.isError) {
    return (
      <div className="text-destructive flex items-center justify-between gap-2 rounded-lg border border-dashed p-3 text-sm">
        <span>{t("Carrier intelligence could not be loaded. {0}", intelQuery.error.message)}</span>
        <Button type="button" size="xs" variant="outline" onClick={() => void intelQuery.refetch()}>
          <RefreshCwIcon />
          {t("Retry")}
        </Button>
      </div>
    );
  }

  const { carrier, carrierIntelProvider: provider } = intelQuery.data;

  if (!carrier) {
    return (
      <p className="text-muted-foreground rounded-lg border border-dashed p-3 text-sm">
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

  return (
    <div className="flex flex-col gap-4">
      {!provider.configured ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-2 text-xs">
          {t(
            "No provider is connected right now, so this is the last snapshot on file and cannot be refreshed.",
          )}{" "}
          <Link to="/admin/integrations?category=CarrierCompliance" className="underline">
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
        <TabsList variant="underline">
          <TabsTab value="findings">
            {t("Findings")}
            {findingCount > 0 ? (
              <Badge
                variant={blockerCount > 0 ? "inactive" : "secondary"}
                className="max-h-5 tabular-nums"
              >
                {findingCount}
              </Badge>
            ) : null}
          </TabsTab>
          <TabsTab value="profile">{t("Profile")}</TabsTab>
          <TabsTab value="sync">{t("Sync")}</TabsTab>
          <TabsTab value="timeline">
            {t("Timeline")}
            {carrier.openIntelEventCount > 0 ? (
              <Badge variant="warning" className="max-h-5 tabular-nums">
                {carrier.openIntelEventCount}
              </Badge>
            ) : null}
          </TabsTab>
          <TabsTab value="history">{t("History")}</TabsTab>
        </TabsList>
        <TabsContent value="findings" className="pt-3">
          <IntelligenceFindings
            carrierId={carrierId}
            findings={snapshot.findings}
            ruleLabels={ruleLabels}
            canApprove={canApprove}
            onChanged={invalidateIntel}
          />
        </TabsContent>
        <TabsContent value="profile" className="pt-3">
          <CarrierIntelProfileView
            profile={snapshot.profile}
            provider={snapshot.provider}
            notFound={snapshot.notFound}
            columns={1}
          />
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
