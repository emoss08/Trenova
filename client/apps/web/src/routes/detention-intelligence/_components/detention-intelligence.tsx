import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { DetentionIntelligenceEmpty } from "./detention-intelligence-empty";
import { DetentionIntelligenceSkeleton } from "./detention-intelligence-skeleton";
import { FacilityScorecardSheet } from "./facility-scorecard-sheet";
import { FocusSection } from "./focus-section";
import { hasDetention } from "./intelligence-model";
import { IntelligenceBrief, IntelligenceFigures, NextSteps } from "./intelligence-overview";
import {
  FacilityHighlightProvider,
  scrollToSection,
  type NarrativeTargets,
} from "./intelligence-parts";
import { PaidSection } from "./paid-section";
import { TrendSection } from "./trend-section";
import { useDetentionIntelligence } from "./use-detention-intelligence";
import { WhenSection } from "./when-section";
import { WhySection } from "./why-section";

/**
 * Detention intelligence: what an organization needs to know to lower detention. Every figure,
 * the summary and the next steps come from one read, so they agree.
 */
export function DetentionIntelligence({
  days,
  onWiden,
}: {
  /** How many whole days of the organization's time zone the page reads, ending today. */
  days: number;
  /** Moves the page to its widest window; absent once it is there. */
  onWiden?: () => void;
}) {
  const t = useT();
  const query = useDetentionIntelligence(days);
  const [openFacility, setOpenFacility] = useState<string | null>(null);
  const data = query.data;

  if (query.isError && !data) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription>
          {describeError(query.error).detail ?? t("The detention analytics could not be loaded.")}
        </AlertDescription>
        <AlertAction>
          <Button
            type="button"
            size="xs"
            variant="outline"
            disabled={query.isFetching}
            onClick={() => void query.refetch()}
          >
            {t("Try again")}
          </Button>
        </AlertAction>
      </Alert>
    );
  }

  if (!data) {
    return <DetentionIntelligenceSkeleton />;
  }

  if (!hasDetention(data)) {
    return (
      <DetentionIntelligenceEmpty
        title={t("No detention in this window")}
        description={t(
          "{0, plural, one {No stop settled detention at any facility in the last # day. Look back further, or check that the detention engine is switched on for this organization.} other {No stop settled detention at any facility in the last # days. Look back further, or check that the detention engine is switched on for this organization.}}",
          days,
        )}
        onWiden={onWiden}
        widenLabel={t("Look back 180 days")}
      />
    );
  }

  const targets: NarrativeTargets = {
    onFacility: setOpenFacility,
    onCustomer: () => scrollToSection("customers"),
    onSection: scrollToSection,
  };
  const facility = data.facilities.find((row) => row.locationId === openFacility) ?? null;

  return (
    <FacilityHighlightProvider>
      <div
        className={cn(
          "flex flex-col gap-10 transition-opacity",
          query.isPlaceholderData && "opacity-70",
        )}
      >
        <div className="flex flex-col gap-5">
          <IntelligenceBrief data={data} targets={targets} updatedAt={query.dataUpdatedAt} />
          <IntelligenceFigures data={data} />
          <NextSteps steps={data.nextSteps} targets={targets} />
        </div>
        <FocusSection data={data} onOpenFacility={setOpenFacility} />
        <WhySection data={data} onOpenFacility={setOpenFacility} />
        <WhenSection data={data} />
        <PaidSection data={data} />
        <TrendSection data={data} onOpenFacility={setOpenFacility} />
      </div>
      <FacilityScorecardSheet
        page={data}
        facility={facility}
        days={days}
        onClose={() => setOpenFacility(null)}
      />
    </FacilityHighlightProvider>
  );
}
