import { InfoPopover } from "@/components/info-popover";
import { useResetTableSearchParams } from "@/hooks/data-table/use-reset-table-search-params";
import {
  activityViewParser,
  activityViewValues,
  type ActivityViewValue,
} from "@/hooks/use-organization-setting-state";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import {
  ActivityIcon,
  Key01Icon,
  LinkExternal01Icon,
  Lock01Icon,
  ShieldAlertIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryState } from "nuqs";
import { lazy, useCallback, useMemo } from "react";
import { firstAllowed, useSecurityAccess } from "./use-security-access";

const AuthEventsTable = lazy(() => import("./activity/auth-events-table"));
const RiskDecisionsTable = lazy(() => import("./activity/risk-decisions-table"));
const ExternalIdentitiesTable = lazy(() => import("./activity/external-identities-table"));
const MFAAuthenticatorsTable = lazy(() => import("./activity/mfa-authenticators-table"));

const ACTIVITY_VIEW_LABELS = defineLabels({
  auth: "Sign-ins",
  risk: "Risk decisions",
  identities: "External identities",
  mfa: "Two-factor devices",
} as const);

const ACTIVITY_VIEW_ICONS: Record<ActivityViewValue, IconComponent> = {
  auth: Key01Icon,
  risk: ShieldAlertIcon,
  identities: LinkExternal01Icon,
  mfa: Lock01Icon,
};

/**
 * Every sign-in, the risk checks run on them, the identities linked from your providers and the
 * two-factor devices people have enrolled, one table at a time. Each is the shared data table,
 * so it searches, filters, sorts and pages on the server like every other list in the product.
 */
export function ActivityTab() {
  const t = useT();
  const access = useSecurityAccess();
  const resetTable = useResetTableSearchParams();
  const [requestedView, setActivityView] = useQueryState("activityView", activityViewParser);
  const activityView =
    firstAllowed(requestedView, activityViewValues, access.activity) ?? requestedView;
  const viewItems = useMemo<SegmentedControlItem<ActivityViewValue>[]>(
    () =>
      activityViewValues
        .filter((view) => access.activity[view])
        .map((view) => ({
          value: view,
          label: ACTIVITY_VIEW_LABELS[view],
          icon: ACTIVITY_VIEW_ICONS[view],
        })),
    [access.activity],
  );
  const handleActivityViewChange = useCallback(
    (value: ActivityViewValue) => {
      if (value === activityView) {
        return;
      }
      resetTable();
      void setActivityView(value);
    },
    [activityView, resetTable, setActivityView],
  );

  return (
    <section aria-label={t("Activity")} className="flex min-w-0 flex-col gap-3">
      <header className="flex flex-wrap items-center gap-x-3 gap-y-2">
        <div className="flex items-center gap-2">
          <ActivityIcon aria-hidden className="text-foreground-subtle size-3.5" />
          <h3 className="text-sm font-semibold">{t("Activity")}</h3>
          <InfoPopover title={t("Activity")}>
            {t(
              "Sign-in outcomes, the risk checks run on them, identities linked from your providers and the two-factor devices people have enrolled.",
            )}
          </InfoPopover>
        </div>
        {viewItems.length > 1 ? (
          <div className="max-w-full overflow-x-auto sm:ml-auto">
            <SegmentedControl
              aria-label={t("Activity view")}
              items={viewItems}
              value={activityView}
              onValueChange={handleActivityViewChange}
            />
          </div>
        ) : null}
      </header>
      {activityView === "auth" ? (
        <DataTableLazyComponent key="auth">
          <AuthEventsTable />
        </DataTableLazyComponent>
      ) : null}
      {activityView === "risk" ? (
        <DataTableLazyComponent key="risk">
          <RiskDecisionsTable />
        </DataTableLazyComponent>
      ) : null}
      {activityView === "identities" ? (
        <DataTableLazyComponent key="identities">
          <ExternalIdentitiesTable />
        </DataTableLazyComponent>
      ) : null}
      {activityView === "mfa" ? (
        <DataTableLazyComponent key="mfa">
          <MFAAuthenticatorsTable />
        </DataTableLazyComponent>
      ) : null}
    </section>
  );
}
