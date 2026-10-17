import { briefPhraseClass, type BriefPhraseTone } from "@/components/brief-phrase";
import { firstNameOf } from "@/lib/onboarding-copy";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  SecurityBriefing,
  SecurityFinding,
  SecurityFindingSeverity,
} from "@trenova/shared/types/iam";
import { Fragment, type ReactNode } from "react";
import { Link } from "react-router";
import {
  BRIEF_SENTENCE_LIMIT,
  briefClosing,
  briefSentence,
  type FindingGroups,
} from "./finding-copy";

const PHRASE_TONE: Record<SecurityFindingSeverity, BriefPhraseTone | undefined> = {
  critical: "danger",
  warning: "warning",
  notice: undefined,
  pass: undefined,
};

export type ReachableAction = (finding: SecurityFinding) => string | null;

/**
 * The brief at the head of the security workspace, drawn as every page brief is: a greeting and
 * a few sentences at reading size, the phrase in each that names a finding linking to where it
 * is settled, and the time it was checked; the recommendations under it are its next steps. It
 * is worked out from this organization's own records as it loads, so it is shown whole rather
 * than streamed in.
 */
export function BriefingBrief({
  briefing,
  groups,
  hrefFor,
  refreshing,
  onRefresh,
}: {
  briefing: SecurityBriefing;
  groups: FindingGroups;
  hrefFor: ReachableAction;
  refreshing: boolean;
  onRefresh: () => void;
}) {
  const t = useT();
  const rt = useRichT();
  const userName = useAuthStore((state) => state.user?.name);
  const firstName = firstNameOf(userName);
  const said = groups.open.slice(0, BRIEF_SENTENCE_LIMIT);
  const closing = briefClosing(t, groups);
  const asOf = formatToUserTimezone(briefing.generatedAt, {
    showDate: false,
    showTimeZone: false,
    showSeconds: false,
  });

  const sentences: ReactNode[] = said.map((finding) => {
    const href = hrefFor(finding);
    const tone = PHRASE_TONE[finding.severity];
    const phrase = (children: ReactNode) =>
      href === null ? (
        <b className="text-foreground font-semibold">{children}</b>
      ) : (
        <Link to={href} className={briefPhraseClass(tone)}>
          {children}
        </Link>
      );
    return briefSentence(rt, t, finding, briefing, phrase);
  });

  return (
    <section aria-label={t("Security brief")} className="flex min-w-0 max-w-full flex-col gap-2">
      <p className="text-foreground text-lg leading-relaxed text-pretty">
        {firstName ? t("Hello {0}.", firstName) : t("Hello.")}{" "}
        {sentences.map((sentence, index) => (
          <Fragment key={said[index]?.code ?? index}>
            {sentence}{" "}
          </Fragment>
        ))}
        {closing}
      </p>
      <span className="text-muted-foreground flex items-center gap-1 text-xs">
        {t("Brief as of {0}", asOf)}
        <span aria-hidden>·</span>
        <Button
          size="bare"
          variant="bare"
          className="text-muted-foreground hover:text-foreground text-xs font-normal underline-offset-4 hover:underline"
          disabled={refreshing}
          onClick={onRefresh}
        >
          {t("Check again")}
        </Button>
      </span>
    </section>
  );
}
