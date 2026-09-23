import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { ArrowUpRightIcon } from "lucide-react";
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
      <p className="text-muted-foreground p-4 text-sm">{t("This page has no link to open.")}</p>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 p-3">
      <div className="space-y-0.5">
        <p className="text-sm font-medium">{target.name}</p>
        {target.location !== "" && target.location !== target.name && (
          <p className="text-muted-foreground text-xs">{target.location}</p>
        )}
      </div>
      <Button size="sm" className="self-start" render={<Link to={target.path} />}>
        <ArrowUpRightIcon className="size-3.5" />
        {t("Open")}
      </Button>
    </div>
  );
}
