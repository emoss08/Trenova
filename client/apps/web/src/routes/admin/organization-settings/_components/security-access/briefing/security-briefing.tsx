import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import type { Tone } from "@/components/kpi/tone";
import { usePlanRestrictions } from "@/hooks/use-plan-restrictions";
import { isPlanRestricted, PlanCapability } from "@/lib/plan-capability";
import { organization } from "@/lib/queries/organization";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatPercent } from "@trenova/shared/lib/utils";
import { formatPeriod } from "@/lib/period";
import type {
  SecurityBriefing as SecurityBriefingData,
  SecurityFinding,
  SecurityFindingCode,
  SecurityFindingSeverity,
} from "@trenova/shared/types/iam";
import { useSecurityAccess } from "../use-security-access";
import { RecentActivityPanel, UnenrolledMembersPanel } from "./briefing-aside";
import { BriefingBrief, type ReachableAction } from "./briefing-brief";
import { BriefingFindings } from "./briefing-findings";
import { coveragePercent, describeFinding, findingHref, groupFindings } from "./finding-copy";
import { PostureRing } from "./posture-ring";
import { BRIEFING_COLUMNS, SecurityBriefingSkeleton } from "./security-briefing-skeleton";

const SEVERITY_TONE: Record<SecurityFindingSeverity, Tone> = {
  critical: "danger",
  warning: "warning",
  notice: "info",
  pass: "success",
};

function findingTone(
  briefing: SecurityBriefingData,
  code: SecurityFindingCode,
): Tone | undefined {
  const finding = briefing.findings.find((item) => item.code === code);
  return finding ? SEVERITY_TONE[finding.severity] : undefined;
}

/**
 * The first thing the security workspace shows, laid out as the other page briefings are: the
 * brief beside a ring of every check it ran, the figures it was drawn from (the server's session
 * limits among them), then every recommendation beside who has not turned on two-factor and the
 * latest sign-ins. Both rows share one grid, so the side column lines up from top to bottom.
 */
export function SecurityBriefing({ organizationId }: { organizationId: string }) {
  const t = useT();
  const access = useSecurityAccess();
  const planRestrictions = usePlanRestrictions();
  const briefingQuery = useQuery(organization.securityBriefing(organizationId));

  if (briefingQuery.isPending) {
    return <SecurityBriefingSkeleton />;
  }

  if (briefingQuery.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <AlertDescription>{t("The security briefing could not be loaded.")}</AlertDescription>
        <AlertAction>
          <Button size="xs" variant="outline" onClick={() => void briefingQuery.refetch()}>
            {t("Try again")}
          </Button>
        </AlertAction>
      </Alert>
    );
  }

  const briefing = briefingQuery.data;
  const groups = groupFindings(briefing.findings);
  const canOpenUsers = access.pages.users;
  const canOpenAPIKeys =
    access.pages.apiKeys && !isPlanRestricted(planRestrictions, PlanCapability.APIKeys);

  const hrefFor: ReachableAction = (finding: SecurityFinding) => {
    const action = describeFinding(t, finding, briefing).action;
    if (action === null) {
      return null;
    }
    const reachable =
      action.kind === "tab"
        ? access.tabs[action.tab]
        : action.requires === "users"
          ? canOpenUsers
          : canOpenAPIKeys;
    return reachable ? findingHref(action) : null;
  };

  return (
    <div className="flex flex-col gap-8">
      <div className={BRIEFING_COLUMNS}>
        <BriefingBrief
          briefing={briefing}
          groups={groups}
          hrefFor={hrefFor}
          refreshing={briefingQuery.isFetching}
          onRefresh={() => void briefingQuery.refetch()}
        />
        <PostureRing groups={groups} />
      </div>
      <BriefingFigures briefing={briefing} />
      <div className={BRIEFING_COLUMNS}>
        <BriefingFindings briefing={briefing} groups={groups} hrefFor={hrefFor} />
        <aside className="flex min-w-0 flex-col gap-8">
          {briefing.mfa ? (
            <UnenrolledMembersPanel mfa={briefing.mfa} canOpenUsers={canOpenUsers} />
          ) : null}
          <RecentActivityPanel
            organizationId={organizationId}
            canReadAuth={access.activity.auth}
            canReadRisk={access.activity.risk}
            canOpenActivity={access.tabs.activity}
          />
        </aside>
      </div>
    </div>
  );
}

function ssoValue(t: TranslateFn, sso: NonNullable<SecurityBriefingData["sso"]>): string {
  if (sso.enforced) {
    return t("Enforced");
  }
  return sso.enabledProviders > 0 ? t("Optional") : t("Not set up");
}

function BriefingFigures({ briefing }: { briefing: SecurityBriefingData }) {
  const t = useT();
  const { mfa, members, signIns, apiKeys, sso, session } = briefing;
  const idleOn = session.idleTimeoutSeconds > 0;

  const unsuccessful = signIns ? signIns.failed + signIns.denied + signIns.throttled : 0;

  return (
    <KpiStrip bare aria-label={t("Security figures")} minItemWidth="9rem">
      {mfa ? (
        <KpiStripItem
          label={t("Two-factor coverage")}
          value={formatPercent(coveragePercent(mfa.enrolled, mfa.total), 0)}
          sub={t("{0} of {1} members", mfa.enrolled, mfa.total)}
          tone={findingTone(briefing, "mfa_coverage")}
        />
      ) : null}
      {members ? (
        <KpiStripItem
          label={t("Active members")}
          value={String(members.active)}
          sub={
            members.dormant > 0
              ? t(
                  "{0} unused in {1, plural, one {# day} other {# days}}",
                  members.dormant,
                  briefing.dormantAfterDays,
                )
              : t("All in recent use")
          }
          tone={findingTone(briefing, "dormant_accounts")}
        />
      ) : null}
      {signIns ? (
        <KpiStripItem
          label={t("Failed sign-ins")}
          value={String(unsuccessful)}
          sub={t("Last {0, plural, one {# day} other {# days}}", briefing.signInWindowDays)}
          tone={findingTone(briefing, "failed_sign_ins")}
        />
      ) : null}
      {apiKeys ? (
        <KpiStripItem
          label={t("Active API keys")}
          value={String(apiKeys.active)}
          sub={
            apiKeys.withoutExpiry > 0
              ? t("{0} without an expiry", apiKeys.withoutExpiry)
              : apiKeys.active > 0
                ? t("All expire")
                : t("None issued")
          }
          tone={findingTone(briefing, "api_key_expiry")}
        />
      ) : null}
      <KpiStripItem
        label={t("Idle sign-out")}
        value={idleOn ? formatPeriod(t, session.idleTimeoutSeconds) : t("Off")}
        sub={
          session.maxAgeSeconds > 0
            ? t("Sessions last {0}", formatPeriod(t, session.maxAgeSeconds))
            : t("Set by the server")
        }
        hint={t(
          "Session limits are part of the server's configuration (security.session) and apply to every organization it hosts.",
        )}
        tone={findingTone(briefing, "idle_timeout")}
      />
      {sso ? (
        <KpiStripItem
          label={t("Single sign-on")}
          value={ssoValue(t, sso)}
          sub={
            sso.enabledProviders === 0
              ? t("No providers enabled")
              : t(
                  "{0, plural, one {# provider enabled} other {# providers enabled}}",
                  sso.enabledProviders,
                )
          }
          tone={findingTone(briefing, "sso")}
        />
      ) : null}
    </KpiStrip>
  );
}
