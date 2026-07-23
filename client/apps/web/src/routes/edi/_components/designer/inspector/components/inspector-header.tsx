import { Badge } from "@trenova/shared/components/ui/badge";
import { DatabaseIcon } from "lucide-react";
import type { InspectorContext } from "../inspector-context";

export default function InspectorHeader({
  context,
  fallbackTitle,
}: {
  context?: InspectorContext;
  fallbackTitle: string;
}) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-3 border-b p-4">
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <DatabaseIcon className="size-4 text-muted-foreground" />
          <h2 className="truncate text-base font-semibold">{context?.title ?? fallbackTitle}</h2>
          {context?.status ? (
            <Badge variant={context.status.variant}>{context.status.label}</Badge>
          ) : null}
        </div>
        <div className="mt-1 text-sm text-muted-foreground">
          {context?.subtitle ?? "Loading message details."}
        </div>
      </div>
      {context ? (
        <div className="flex flex-wrap gap-2 text-xs">
          {context.controlRows.slice(0, 3).map(([label, value]) => (
            <Badge key={label} variant="outline" className="font-mono">
              {controlPrefix(label)} {value}
            </Badge>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function controlPrefix(label: string) {
  if (label.startsWith("Interchange")) return "ISA";
  if (label.startsWith("Group")) return "GS";
  return "ST";
}
