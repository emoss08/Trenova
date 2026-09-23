import type { AssistantArtifact } from "@/types/assistant";
import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { ArrowUpRightIcon, CompassIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { navigationFrom } from "./artifact-payloads";

/**
 * Where the assistant took the person.
 *
 * The app followed it once, when it arrived. Read back later it is only a
 * card with the way there, so reopening a conversation never moves anybody.
 */
export function NavigationArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const target = useMemo(() => navigationFrom(artifact), [artifact]);

  if (target === null) {
    return (
      <ArtifactNotice kind={artifact.kind}>{t("This page has no link to open.")}</ArtifactNotice>
    );
  }

  return (
    <div className="animate-rise flex min-h-0 flex-1 flex-col p-4">
      <div className="border-border-subtle flex items-center gap-3 rounded-lg border p-3">
        <span className="bg-sunken text-foreground-muted flex size-9 shrink-0 items-center justify-center rounded-md">
          <CompassIcon className="size-4" />
        </span>
        <div className="min-w-0 flex-1 space-y-0.5">
          <p className="truncate text-sm font-medium">{target.name}</p>
          {target.location !== "" && target.location !== target.name && (
            <p className="text-muted-foreground truncate text-xs">{target.location}</p>
          )}
        </div>
        <Button size="sm" className="shrink-0" render={<Link to={target.path} />}>
          {t("Open")}
          <ArrowUpRightIcon className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}
