import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { useResetTableSearchParams } from "@/hooks/data-table/use-reset-table-search-params";
import { DataTableLazyComponent, LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  Download01Icon,
  Key01Icon,
  Lock01Icon,
  ShieldAlertIcon,
  BotIcon,
  ActivityIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy } from "react";
import { Link } from "react-router";
import { AuditChainStatusSkeleton } from "./_components/audit-chain-status-skeleton";
import { AuditAlertPoliciesSkeleton } from "./_components/audit-alert-policies-skeleton";

const AuditLogTable = lazy(() => import("./_components/audit-log-table"));
const AuditExportsTable = lazy(() => import("./_components/audit-exports-table"));
const AuditLegalHoldsTable = lazy(() => import("./_components/audit-legal-holds-table"));
const AuditAlertPolicies = lazy(() => import("./_components/audit-alert-policies"));
const AuditChainStatus = lazy(() => import("./_components/audit-chain-status"));

const auditViewValues = ["entries", "exports", "holds", "alerts"] as const;
type AuditViewValue = (typeof auditViewValues)[number];

const auditViewParser = parseAsStringLiteral(auditViewValues)
  .withDefault("entries")
  .withOptions({ history: "push" });

const AUDIT_VIEW_LABELS = defineLabels({
  entries: "Entries",
  exports: "Exports",
  holds: "Legal holds",
  alerts: "Alerts",
} as const);

const AUDIT_VIEW_ICONS: Record<AuditViewValue, IconComponent> = {
  entries: ActivityIcon,
  exports: Download01Icon,
  holds: Lock01Icon,
  alerts: ShieldAlertIcon,
};

const SIGN_IN_ACTIVITY_PATH =
  "/admin/organization-settings?tab=security&securityTab=activity&activityView=auth";
const AI_AUDIT_PATH = "/admin/agent-control?tab=audit&audit=trail";

export function AuditLogsPage() {
  const t = useT();
  const resetTable = useResetTableSearchParams();
  const { allowed: canExport } = usePermission(Resource.AuditLog, Operation.Export);
  const { allowed: canSeeHolds } = usePermission(Resource.AuditLegalHold, Operation.Read);
  const { allowed: canSeeSecurity } = usePermission(Resource.SecurityAuditLog, Operation.Read);
  const { allowed: canSeeSignIns } = usePermission(Resource.AuthEvent, Operation.Read);
  const { allowed: canSeeAIAudit } = usePermission(Resource.AIAuditTrail, Operation.Read);
  const [requestedView, setView] = useQueryState("view", auditViewParser);

  const allowed: Record<AuditViewValue, boolean> = {
    entries: true,
    exports: canExport,
    holds: canSeeHolds,
    alerts: canSeeSecurity,
  };
  const view = allowed[requestedView] ? requestedView : "entries";
  const viewItems: SegmentedControlItem<AuditViewValue>[] = auditViewValues
    .filter((value) => allowed[value])
    .map((value) => ({
      value,
      label: AUDIT_VIEW_LABELS[value],
      icon: AUDIT_VIEW_ICONS[value],
    }));

  const changeView = (value: AuditViewValue) => {
    if (value === view) {
      return;
    }
    resetTable();
    void setView(value);
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Audit log"),
        description: t(
          "Every change, access and administrative action in your organization, sealed against tampering",
        ),
        actions: (
          <div className="flex flex-wrap items-center gap-2">
            {canSeeSignIns ? (
              <Button
                variant="outline"
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
                variant="outline"
                size="sm"
                nativeButton={false}
                render={<Link to={AI_AUDIT_PATH} />}
              >
                <BotIcon aria-hidden />
                {t("AI audit trail")}
              </Button>
            ) : null}
          </div>
        ),
      }}
    >
      {canSeeSecurity ? (
        <LazyComponent fallback={<AuditChainStatusSkeleton />}>
          <AuditChainStatus />
        </LazyComponent>
      ) : null}
      {viewItems.length > 1 ? (
        <div className="max-w-full overflow-x-auto">
          <SegmentedControl
            aria-label={t("Audit log view")}
            items={viewItems}
            value={view}
            onValueChange={changeView}
          />
        </div>
      ) : null}
      {view === "entries" ? (
        <DataTableLazyComponent key="entries">
          <AuditLogTable />
        </DataTableLazyComponent>
      ) : null}
      {view === "exports" ? (
        <DataTableLazyComponent key="exports">
          <AuditExportsTable />
        </DataTableLazyComponent>
      ) : null}
      {view === "holds" ? (
        <DataTableLazyComponent key="holds">
          <AuditLegalHoldsTable />
        </DataTableLazyComponent>
      ) : null}
      {view === "alerts" ? (
        <LazyComponent key="alerts" fallback={<AuditAlertPoliciesSkeleton />}>
          <AuditAlertPolicies />
        </LazyComponent>
      ) : null}
    </PageLayout>
  );
}
