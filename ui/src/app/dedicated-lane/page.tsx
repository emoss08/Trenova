import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { Separator } from "@/components/ui/separator";
import { queries } from "@/lib/queries";
import { useSuspenseQuery } from "@tanstack/react-query";
import { lazy } from "react";
import { DedicatedLaneSuggestions } from "./_components/dedicated-lane-suggestions";

const DedicatedLaneTable = lazy(
  () => import("./_components/dedicated-lane-table"),
);

export function DedicatedLane() {
  const { data: suggestions, isLoading: isLoadingSuggestions } =
    useSuspenseQuery({
      ...queries.dedicatedLaneSuggestion.getSuggestions(),
    });

  return (
    <div className="flex flex-col gap-y-3">
      <MetaTags title="Dedicated Lanes" description="Dedicated Lanes" />
      <Header />
      <QueryLazyComponent
        queryKey={queries.dedicatedLaneSuggestion.getSuggestions._def}
      >
        {suggestions.results.length > 0 && (
          <>
            <DedicatedLaneSuggestions
              suggestions={suggestions.results}
              isLoading={isLoadingSuggestions}
            />
            <Separator className="my-4" />
          </>
        )}
        <DedicatedLaneTable />
      </QueryLazyComponent>
    </div>
  );
}

function Header() {
  return (
    <div className="flex flex-col items-start">
      <h1 className="text-3xl font-bold tracking-tight">Dedicated Lanes</h1>
      <p className="text-muted-foreground">
        Dedicated lanes are a feature that allows you to assign a lane to a
        specific customer for a specific period of time.
      </p>
    </div>
  );
}
