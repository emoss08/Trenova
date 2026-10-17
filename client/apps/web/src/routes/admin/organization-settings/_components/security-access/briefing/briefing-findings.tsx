import {
  AlertOctagonIcon,
  AlertTriangleIcon,
  ArrowRightIcon,
  CheckCircleIcon,
  ChevronDownIcon,
  InfoCircleIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  SecurityBriefing,
  SecurityFinding,
  SecurityFindingSeverity,
} from "@trenova/shared/types/iam";
import { Link } from "react-router";
import { EmptyState, SecuritySection } from "../shared";
import type { ReachableAction } from "./briefing-brief";
import { describeFinding, type FindingGroups } from "./finding-copy";

const SEVERITY_ICON: Record<SecurityFindingSeverity, IconComponent> = {
  critical: AlertOctagonIcon,
  warning: AlertTriangleIcon,
  notice: InfoCircleIcon,
  pass: CheckCircleIcon,
};

const SEVERITY_ICON_CLASS: Record<SecurityFindingSeverity, string> = {
  critical: "text-danger-foreground",
  warning: "text-warning-foreground",
  notice: "text-info-foreground",
  pass: "text-success-foreground",
};

/**
 * Every check that still wants doing, worst first, each a row that opens where it is settled,
 * and under them the checks that already hold, folded behind their count.
 */
export function BriefingFindings({
  briefing,
  groups,
  hrefFor,
}: {
  briefing: SecurityBriefing;
  groups: FindingGroups;
  hrefFor: ReachableAction;
}) {
  const t = useT();

  return (
    <SecuritySection
      title={t("Recommendations")}
      count={groups.open.length}
      help={t(
        "Each check reads this organization's own records as they are now. Check again after a change to see it reflected.",
      )}
    >
      {groups.open.length > 0 ? (
        <ul className="divide-border-subtle divide-y">
          {groups.open.map((finding) => (
            <li key={finding.code}>
              <FindingRow finding={finding} briefing={briefing} href={hrefFor(finding)} />
            </li>
          ))}
        </ul>
      ) : (
        <EmptyState
          icon={<CheckCircleIcon className="text-success-foreground" />}
          label={t("Nothing to recommend")}
          description={t("Every check the briefing runs has passed.")}
        />
      )}
      {groups.passed.length > 0 ? (
        <PassedChecks findings={groups.passed} briefing={briefing} />
      ) : null}
    </SecuritySection>
  );
}

function FindingRow({
  finding,
  briefing,
  href,
}: {
  finding: SecurityFinding;
  briefing: SecurityBriefing;
  href: string | null;
}) {
  const t = useT();
  const copy = describeFinding(t, finding, briefing);
  const Icon = SEVERITY_ICON[finding.severity];
  const body = (
    <>
      <Icon
        aria-hidden
        className={cn("mt-0.5 size-4 shrink-0", SEVERITY_ICON_CLASS[finding.severity])}
      />
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{copy.title}</p>
        <p className="text-muted-foreground mt-0.5 max-w-prose text-xs text-pretty">
          {copy.detail}
        </p>
      </div>
      {href !== null && copy.action ? (
        <span className="text-muted-foreground group-hover:text-foreground mt-0.5 hidden shrink-0 items-center gap-1 text-xs transition-colors sm:inline-flex">
          {copy.action.label}
          <ArrowRightIcon aria-hidden className="size-3" />
        </span>
      ) : null}
    </>
  );

  if (href === null || copy.action === null) {
    return <div className="flex gap-3 py-3">{body}</div>;
  }

  return (
    <Link
      to={href}
      aria-label={t("{0}: {1}", copy.title, copy.action.label)}
      className="group hover:bg-surface-hover ui-focus-ring -mx-2 flex gap-3 rounded-md px-2 py-3 transition-colors"
    >
      {body}
    </Link>
  );
}

function PassedChecks({
  findings,
  briefing,
}: {
  findings: SecurityFinding[];
  briefing: SecurityBriefing;
}) {
  const t = useT();

  return (
    <Collapsible className="border-border-subtle border-t">
      <CollapsibleTrigger className="group/passed ui-focus-ring text-muted-foreground hover:text-foreground flex w-full items-center gap-2 rounded-sm py-2.5 text-left text-xs transition-colors">
        <CheckCircleIcon aria-hidden className="text-success-foreground size-3.5 shrink-0" />
        <span className="flex-1">
          {t("{0, plural, one {# check passed} other {# checks passed}}", findings.length)}
        </span>
        <ChevronDownIcon
          aria-hidden
          className="size-3.5 shrink-0 transition-transform group-data-[panel-open]/passed:rotate-180"
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ul className="flex flex-col gap-2.5 pb-3">
          {findings.map((finding) => {
            const copy = describeFinding(t, finding, briefing);
            return (
              <li key={finding.code} className="flex gap-2.5 pl-0.5">
                <CheckCircleIcon
                  aria-hidden
                  className="text-success-foreground mt-px size-3.5 shrink-0"
                />
                <div className="min-w-0">
                  <p className="text-xs font-medium">{copy.title}</p>
                  <p className="text-muted-foreground text-xs text-pretty">{copy.detail}</p>
                </div>
              </li>
            );
          })}
        </ul>
      </CollapsibleContent>
    </Collapsible>
  );
}
