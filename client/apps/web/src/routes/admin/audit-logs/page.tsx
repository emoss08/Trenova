import { PageLayout } from "@/components/navigation/sidebar-layout";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useResetTableSearchParams } from "@/hooks/data-table/use-reset-table-search-params";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { usePermission } from "@/hooks/use-permission";
import type { AuditActorStat, AuditReview } from "@/lib/graphql/audit-overview";
import { audit } from "@/lib/queries/audit";
import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  ArrowNarrowLeftIcon,
  BotIcon,
  ChevronDownIcon,
  Download01Icon,
  Key01Icon,
  Lock01Icon,
  ShieldAlertIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryState, useQueryStates } from "nuqs";
import { lazy, type ReactNode } from "react";
import { Link, type LoaderFunctionArgs } from "react-router";
import { AuditAlertPoliciesSkeleton } from "./_components/audit-alert-policies-skeleton";
import { AuditOverviewSkeleton } from "./_components/audit-overview-skeleton";
import {
  AUDIT_PARAMS,
  auditEntryParser,
  auditQuickViewParser,
  auditViewParser,
  auditWindowOf,
  auditWindowParser,
  OVERVIEW_FOLDED_STORAGE_KEY,
  type AuditQuickView,
  type AuditView,
} from "./_components/audit-page-state";
import { AuditSealAlert, AuditSealChip } from "./_components/audit-seal";
import { OVERVIEW_REFRESH_MS, useAuditOverview } from "./_components/use-audit-overview";

const AuditOverviewBand = lazy(() => import("./_components/audit-overview"));
const AuditTrail = lazy(() => import("./_components/audit-trail"));
const AuditExportsTable = lazy(() => import("./_components/audit-exports-table"));
const AuditLegalHoldsTable = lazy(() => import("./_components/audit-legal-holds-table"));
const AuditAlertPolicies = lazy(() => import("./_components/audit-alert-policies"));

const SIGN_IN_ACTIVITY_PATH =
  "/admin/organization-settings?tab=security&securityTab=activity&activityView=auth";
const AI_AUDIT_PATH = "/admin/agent-control?tab=audit&audit=trail";

const RANGE_FIELD = "timestamp";
const ACTOR_FIELD = "principalId";
const RECORD_FIELD = "resourceId";

/** Warms the overview the page reads on mount, for the window in the address. */
export function prefetch({ request }: LoaderFunctionArgs) {
  const param = new URL(request.url).searchParams.get(AUDIT_PARAMS.window);
  const days = auditWindowOf(param === null ? Number.NaN : Number(param));
  return [{ ...audit.overview(days), staleTime: OVERVIEW_REFRESH_MS }];
}

function rangeOf(filters: readonly FieldFilter[]): { from: number; to: number } | null {
  const filter = filters.find((f) => f.field === RANGE_FIELD && f.operator === "daterange");
  const value = filter?.value as { from?: number; to?: number } | undefined;
  return value?.from !== undefined && value?.to !== undefined
    ? { from: value.from, to: value.to }
    : null;
}

function actorOf(filters: readonly FieldFilter[]): string | null {
  const filter = filters.find((f) => f.field === ACTOR_FIELD && f.operator === "eq");
  return typeof filter?.value === "string" ? filter.value : null;
}

/** The folded overview's one line: the window's changes and what waits for a review. */
function OverviewLine({ days, onUnfold }: { days: number; onUnfold: () => void }) {
  const t = useT();
  const { data } = useAuditOverview(days);
  return (
    <div className="text-muted-foreground animate-rise flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
      {data ? (
        <>
          <span>
            {days === 1
              ? t("{0, plural, one {# change} other {# changes}} today", data.totals.entries)
              : t(
                  "{0, plural, one {# change} other {# changes}} in the last {1} days",
                  data.totals.entries,
                  days,
                )}
          </span>
          {data.reviewsIncluded ? (
            <span>
              {t(
                "{0, plural, one {# needs a review} other {# need a review}}",
                data.openReviewCount,
              )}
            </span>
          ) : null}
        </>
      ) : null}
      <Button type="button" variant="ghost" size="xs" className="ml-auto" onClick={onUnfold}>
        <ChevronDownIcon aria-hidden className="size-3" />
        {t("Show overview")}
      </Button>
    </div>
  );
}

/** A tool's heading where the trail's would be: its name and the way back to the trail. */
function ToolHeading({ title, onBack }: { title: string; onBack: () => void }) {
  const t = useT();
  return (
    <div className="flex items-center gap-2">
      <Button type="button" variant="ghost" size="xs" onClick={onBack}>
        <ArrowNarrowLeftIcon aria-hidden />
        {t("Trail")}
      </Button>
      <span aria-hidden className="text-foreground-subtle">
        /
      </span>
      <h2 className="text-base font-semibold">{title}</h2>
    </div>
  );
}

function ToolView({
  title,
  onBack,
  children,
}: {
  title: string;
  onBack: () => void;
  children: ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-3">
      <ToolHeading title={title} onBack={onBack} />
      {children}
    </div>
  );
}

export function AuditLogsPage() {
  const t = useT();
  const { allowed: canExport } = usePermission(Resource.AuditLog, Operation.Export);
  const { allowed: canSeeHolds } = usePermission(Resource.AuditLegalHold, Operation.Read);
  const { allowed: canSeeSecurity } = usePermission(Resource.SecurityAuditLog, Operation.Read);
  const { allowed: canSeeSignIns } = usePermission(Resource.AuthEvent, Operation.Read);
  const { allowed: canSeeAIAudit } = usePermission(Resource.AIAuditTrail, Operation.Read);

  const [requestedView, setView] = useQueryState(AUDIT_PARAMS.view, auditViewParser);
  const [windowParam, setWindow] = useQueryState(AUDIT_PARAMS.window, auditWindowParser);
  const [quick, setQuick] = useQueryState(AUDIT_PARAMS.quick, auditQuickViewParser);
  const [entryId, setEntryId] = useQueryState(AUDIT_PARAMS.entry, auditEntryParser);
  const [table, setTable] = useQueryStates(searchParamsParser);
  const [folded, setFolded] = useLocalStorage(OVERVIEW_FOLDED_STORAGE_KEY, false);
  const resetTable = useResetTableSearchParams();

  const days = auditWindowOf(windowParam);
  const allowed: Record<AuditView, boolean> = {
    trail: true,
    exports: canExport,
    holds: canSeeHolds,
    alerts: canSeeSecurity,
  };
  const view = allowed[requestedView] ? requestedView : "trail";
  const fieldFilters = table.fieldFilters ?? [];

  const toggleView = (next: AuditView) => {
    resetTable();
    void setView(view === next ? null : next);
  };

  const replaceFilter = (field: string, filter: FieldFilter | null) => {
    const kept = fieldFilters.filter((f) => f.field !== field);
    void setTable({ fieldFilters: filter ? [...kept, filter] : kept, pageIndex: 1 });
  };

  const onRangeChange = (range: { from: number; to: number }) =>
    replaceFilter(RANGE_FIELD, { field: RANGE_FIELD, operator: "daterange", value: range });
  const onSelectActor = (actor: AuditActorStat | null) =>
    replaceFilter(
      ACTOR_FIELD,
      actor ? { field: ACTOR_FIELD, operator: "eq", value: actor.principalId } : null,
    );
  const onShowRecordHistory = (resourceId: string) => {
    void setQuick(null);
    replaceFilter(RECORD_FIELD, { field: RECORD_FIELD, operator: "eq", value: resourceId });
  };
  const onQuickView = (next: AuditQuickView) => {
    void setQuick(next === "all" ? null : next);
    void setTable({ pageIndex: 1 });
  };
  const onOpenReview = (review: AuditReview) => {
    void setView(null);
    void setEntryId(review.entryId);
  };

  const headerButton = (target: AuditView, label: string, Icon: IconComponent) => (
    <Button
      type="button"
      variant={view === target ? "secondary" : "outline"}
      size="sm"
      aria-pressed={view === target}
      onClick={() => toggleView(target)}
    >
      <Icon aria-hidden />
      {label}
    </Button>
  );

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Audit log"),
        description: t(
          "Every change, access and administrative action in your organization, sealed against tampering",
        ),
        context: canSeeSecurity ? <AuditSealChip /> : null,
        actions: (
          <div className="flex flex-wrap items-center gap-2">
            {canSeeSignIns ? (
              <Button
                variant="ghost"
                size="sm"
                nativeButton={false}
                render={<Link to={SIGN_IN_ACTIVITY_PATH} />}
              >
                <Key01Icon aria-hidden />
                {t("Sign-in activity")}
              </Button>
            ) : null}
            {canSeeAIAudit ? (
              <Button
                variant="ghost"
                size="sm"
                nativeButton={false}
                render={<Link to={AI_AUDIT_PATH} />}
              >
                <BotIcon aria-hidden />
                {t("AI audit trail")}
              </Button>
            ) : null}
            {canSeeHolds || canSeeSecurity || canExport ? (
              <span aria-hidden className="bg-border mx-1 h-4.5 w-px" />
            ) : null}
            {canSeeHolds ? headerButton("holds", t("Legal holds"), Lock01Icon) : null}
            {canSeeSecurity ? headerButton("alerts", t("Alert rules"), ShieldAlertIcon) : null}
            {canExport ? headerButton("exports", t("Exports"), Download01Icon) : null}
          </div>
        ),
      }}
    >
      {canSeeSecurity ? <AuditSealAlert /> : null}
      {view === "trail" ? (
        <>
          {folded ? (
            <OverviewLine days={days} onUnfold={() => setFolded(false)} />
          ) : (
            <LazyComponent fallback={<AuditOverviewSkeleton />}>
              <AuditOverviewBand
                days={days}
                onDaysChange={(next) => void setWindow(next === 7 ? null : next)}
                range={rangeOf(fieldFilters)}
                onRangeChange={onRangeChange}
                selectedActorId={actorOf(fieldFilters)}
                onSelectActor={onSelectActor}
                onQuickView={onQuickView}
                onOpenReview={onOpenReview}
              />
            </LazyComponent>
          )}
          <DataTableLazyComponent key="trail">
            <AuditTrail
              days={days}
              quick={quick}
              onQuickChange={onQuickView}
              entryId={entryId}
              onEntryChange={(id) => void setEntryId(id)}
              canSeeReviews={canSeeSecurity}
              onShowRecordHistory={onShowRecordHistory}
              headerActions={
                folded ? null : (
                  <Button type="button" variant="ghost" size="xs" onClick={() => setFolded(true)}>
                    <ChevronDownIcon aria-hidden className="size-3 rotate-180" />
                    {t("Hide overview")}
                  </Button>
                )
              }
            />
          </DataTableLazyComponent>
        </>
      ) : null}
      {view === "exports" ? (
        <ToolView title={t("Exports")} onBack={() => toggleView("exports")}>
          <DataTableLazyComponent key="exports">
            <AuditExportsTable />
          </DataTableLazyComponent>
        </ToolView>
      ) : null}
      {view === "holds" ? (
        <ToolView title={t("Legal holds")} onBack={() => toggleView("holds")}>
          <DataTableLazyComponent key="holds">
            <AuditLegalHoldsTable />
          </DataTableLazyComponent>
        </ToolView>
      ) : null}
      {view === "alerts" ? (
        <ToolView title={t("Alert rules")} onBack={() => toggleView("alerts")}>
          <LazyComponent key="alerts" fallback={<AuditAlertPoliciesSkeleton />}>
            <AuditAlertPolicies />
          </LazyComponent>
        </ToolView>
      ) : null}
    </PageLayout>
  );
}
