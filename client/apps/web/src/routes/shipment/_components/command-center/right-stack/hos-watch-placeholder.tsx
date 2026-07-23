import { Button } from "@/components/ui/button";
import { ClockIcon, ExternalLinkIcon } from "lucide-react";
import { Link } from "react-router";
import { ModuleCard } from "./module-card";

export function HosWatchPlaceholder() {
  return (
    <ModuleCard id="hos" title="HOS watch">
      <div className="flex flex-col items-center gap-2 px-4 py-6 text-center">
        <span className="inline-flex size-8 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <ClockIcon className="size-4" />
        </span>
        <p className="text-[11.5px] font-medium">Driver HOS visibility coming soon</p>
        <p className="max-w-55 text-[10.5px] leading-snug text-muted-foreground">
          Hours-of-service tracking lands when the worker-HOS service ships. Until then, view
          current driver status from the Workers area.
        </p>
        <Button
          variant="outline"
          size="xs"
          nativeButton={false}
          render={<Link to="/dispatch/configurations/workers" />}
        >
          <ExternalLinkIcon className="size-3" />
          Open Workers
        </Button>
      </div>
    </ModuleCard>
  );
}
