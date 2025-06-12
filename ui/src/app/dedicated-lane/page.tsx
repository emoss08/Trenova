import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { Separator } from "@/components/ui/separator";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";
import { DedicatedLaneSuggestions } from "./_components/dedicated-lane-suggestions";

const DedicatedLaneTable = lazy(
  () => import("./_components/dedicated-lane-table"),
);

export function DedicatedLane() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="Dedicated Lanes" description="Dedicated Lanes" />
      <Header />
      <QueryLazyComponent
        queryKey={queries.dedicatedLaneSuggestion.getSuggestions._def}
      >
        <DedicatedLaneSuggestions />
        <Separator className="my-4" />
        <DedicatedLaneTable />
      </QueryLazyComponent>
    </div>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dedicated Lanes</h1>
        <p className="text-muted-foreground">
          Dedicated lanes are a feature that allows you to assign a lane to a
          specific customer for a specific period of time.
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
