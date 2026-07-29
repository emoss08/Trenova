import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { cn } from "@trenova/shared/lib/utils";
import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { RotateCwIcon } from "lucide-react";
import { useState } from "react";
import { DetentionIntelligence } from "./_components/detention-intelligence";
import {
  DETENTION_WINDOW_OPTIONS,
  type DetentionWindowValue,
} from "./_components/use-detention-intelligence";

function RefreshAction() {
  const queryClient = useQueryClient();
  const isFetching = useIsFetching({ queryKey: queries.detention._def }) > 0;

  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      className="h-7"
      disabled={isFetching}
      aria-label="Recalculate detention intelligence"
      onClick={() =>
        void queryClient.invalidateQueries({ queryKey: queries.detention._def })
      }
    >
      <RotateCwIcon className={cn("mr-1.5 size-3.5", isFetching && "animate-spin")} />
      {isFetching ? "Recalculating…" : "Refresh"}
    </Button>
  );
}

export function DetentionIntelligencePage() {
  const [windowValue, setWindowValue] = useState<DetentionWindowValue>("90");

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Detention Intelligence",
        description:
          "Which facilities cost the most, which customers are unprofitable once driver pay is netted off, and where discretionary revenue is going",
        actions: (
          <>
            <SegmentedControl
              items={DETENTION_WINDOW_OPTIONS}
              value={windowValue}
              onValueChange={setWindowValue}
              aria-label="Analysis window"
            />
            <RefreshAction />
          </>
        ),
      }}
    >
      <DetentionIntelligence windowValue={windowValue} />
    </PageLayout>
  );
}
