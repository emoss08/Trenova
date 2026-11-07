import { PageLayout } from "@/components/navigation/sidebar-layout";
import { freightOutlook } from "@/lib/queries/freight-outlook";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { Link } from "react-router";
import { FreightOutlook } from "./_components/freight-outlook";
import { DEFAULT_HORIZON, HORIZON_OPTIONS, type Horizon } from "./_components/outlook-model";

const HORIZON_LABELS = defineLabels({ "12": "12 h", "24": "24 h", "48": "48 h" });

/** Warms the one read the page makes on mount. */
export function prefetch() {
  return [freightOutlook.outlook()];
}

export function FreightOutlookPage() {
  const t = useT();
  const [horizon, setHorizon] = useState<Horizon>(DEFAULT_HORIZON);

  return (
    <PageLayout
      fill
      bleed
      pageHeaderProps={{
        title: t("Freight outlook"),
        description: t(
          "Where trucks will be short or idle over the next two days, the moves that fix it and how far to trust the forecast",
        ),
        actions: (
          <div className="flex items-center gap-2">
            <SegmentedControl
              items={HORIZON_OPTIONS.map((option) => ({
                value: option.value,
                label: HORIZON_LABELS[option.value],
              }))}
              value={horizon}
              onValueChange={setHorizon}
              aria-label={t("How far ahead")}
            />
            <Button
              size="sm"
              variant="outline"
              nativeButton={false}
              render={<Link to="/dispatch/console" />}
            >
              {t("Open console")}
            </Button>
          </div>
        ),
      }}
    >
      <FreightOutlook horizon={horizon} />
    </PageLayout>
  );
}
