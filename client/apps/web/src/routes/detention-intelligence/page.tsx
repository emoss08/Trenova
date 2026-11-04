import { PageLayout } from "@/components/navigation/sidebar-layout";
import { detention } from "@/lib/queries/detention";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { DetentionIntelligence } from "./_components/detention-intelligence";
import {
  DETENTION_WINDOW_OPTIONS,
  detentionWindowDays,
  type DetentionWindowValue,
} from "./_components/intelligence-model";
import { INTELLIGENCE_STALE_MS, intelligenceInput } from "./_components/use-detention-intelligence";

const DEFAULT_WINDOW: DetentionWindowValue = "90";

/** Warms the read the page makes on mount: the default window, ending today. */
export function prefetch() {
  return [
    {
      ...detention.intelligence(intelligenceInput(detentionWindowDays(DEFAULT_WINDOW))),
      staleTime: INTELLIGENCE_STALE_MS,
    },
  ];
}

export function DetentionIntelligencePage() {
  const t = useT();
  const [windowValue, setWindowValue] = useState<DetentionWindowValue>(DEFAULT_WINDOW);

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Detention intelligence"),
        description: t(
          "Where trucks wait past free time, who caused it, when it happens, how much of the money is recovered and what to do next",
        ),
        actions: (
          <SegmentedControl
            items={DETENTION_WINDOW_OPTIONS}
            value={windowValue}
            onValueChange={setWindowValue}
            aria-label={t("Analysis window")}
          />
        ),
      }}
    >
      <DetentionIntelligence
        days={detentionWindowDays(windowValue)}
        onWiden={windowValue === "180" ? undefined : () => setWindowValue("180")}
      />
    </PageLayout>
  );
}
