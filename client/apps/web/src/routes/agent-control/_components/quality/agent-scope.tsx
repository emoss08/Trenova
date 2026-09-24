import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { XIcon } from "lucide-react";
import { QUALITY_STALE_MS } from "./quality-model";

/**
 * Says which agent a quality table is narrowed to and offers every agent
 * back. The agent is part of what the table asks the server, not one of its
 * filters, because the link that narrowed it came from outside the table.
 */
export function AgentScope({ agentId, onClear }: { agentId: string; onClear: () => void }) {
  const t = useT();
  const detail = useQuery({ ...queries.agentQuality.agent(agentId), staleTime: QUALITY_STALE_MS });
  const name =
    detail.data?.agentName ?? (detail.isError ? t("An agent that no longer exists") : "…");

  return (
    <div className="flex items-center justify-between gap-2">
      <p className="text-muted-foreground min-w-0 truncate text-sm">
        {t("Showing {0} only", name)}
      </p>
      <Button variant="ghost" size="xs" onClick={onClear}>
        <XIcon className="size-3.5" />
        {t("Show every agent")}
      </Button>
    </div>
  );
}
