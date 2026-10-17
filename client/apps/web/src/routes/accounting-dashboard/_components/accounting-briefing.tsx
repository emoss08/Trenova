import { BriefAsOf } from "@/components/brief-as-of";
import { StreamedText, type StreamedSegment } from "@/components/streamed-text";
import { firstNameOf } from "@/lib/onboarding-copy";
import { ar } from "@/lib/queries/ar";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { AccountingBriefingSkeleton } from "./accounting-briefing-skeleton";
import { BriefingActions } from "./briefing-actions";
import { BRIEFING_LINK_LABELS, BRIEFING_LINK_TONES, briefingLinkPath } from "./briefing-links";

const BRIEFING_STALE_MS = 60_000;

/**
 * The day's brief about the receivables: what the headline figures do not show
 * (a missed promise, invoices about to pass 90 days, customers nobody has
 * called, what falls due next), streamed in behind a greeting, with the next
 * steps under it. Written for the organization before the workday and again
 * when its findings move. A phrase that names a finding opens the page that
 * holds it.
 */
export function AccountingBriefing() {
  const t = useT();
  const navigate = useNavigate();
  const userName = useAuthStore((state) => state.user?.name);
  const { data, isLoading, isError } = useQuery({
    ...ar.briefing(),
    staleTime: BRIEFING_STALE_MS,
  });

  if (isError) return null;
  if (isLoading || !data) return <AccountingBriefingSkeleton />;

  const segments: StreamedSegment[] = data.segments.map((segment) => {
    const path = segment.link ? briefingLinkPath(segment.link, segment.customerId) : null;
    if (!segment.link || !path) return { text: segment.text };

    return {
      text: segment.text,
      label: BRIEFING_LINK_LABELS[segment.link],
      tone: BRIEFING_LINK_TONES[segment.link],
      onActivate: () => void navigate(path),
    };
  });

  const firstName = firstNameOf(userName);

  return (
    <section aria-label={t("Accounting brief")} className="flex w-full min-w-0 flex-col gap-2">
      <StreamedText
        streamKey={`${data.generatedAt}:${data.generation}`}
        prefix={firstName ? `${t("Hello {0}.", firstName)} ` : `${t("Hello.")} `}
        segments={segments}
        className="text-foreground w-full text-lg leading-relaxed text-pretty"
      />
      <BriefingActions actions={data.actions} />
      <BriefAsOf generatedAt={data.generatedAt} narrated={data.narrated} />
    </section>
  );
}
