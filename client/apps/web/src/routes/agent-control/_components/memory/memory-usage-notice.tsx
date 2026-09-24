import {
  agentMemoryUsageQueryKey,
  fetchAgentMemoryUsage,
  memoryUsageNearCap,
} from "@/lib/graphql/agent-memories";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * Says so once the organization keeps nearly as many memories as it should.
 * Every prompt reads from the same pool under a fixed budget, so past the cap
 * fewer of them reach each prompt; retiring what no longer holds is the fix.
 */
export function MemoryUsageNotice() {
  const t = useT();
  const usageQuery = useQuery({
    queryKey: agentMemoryUsageQueryKey,
    queryFn: ({ signal }) => fetchAgentMemoryUsage({ signal }),
  });
  const usage = usageQuery.data;

  if (!usage || !memoryUsageNearCap(usage)) {
    return null;
  }

  const over = usage.activeCount >= usage.activeSoftCap;

  return (
    <Alert size="sm" variant={over ? "destructive" : "warning"} data-testid="memory-usage-notice">
      <AlertTitle>
        {over
          ? t("More memories than agents can use well")
          : t("Nearing the most memories agents can use well")}
      </AlertTitle>
      <AlertDescription>
        {t(
          "Agents may read {0} active memories; an organization should keep at most {1}. Each prompt carries only what fits its budget, so retire what no longer holds.",
          formatNumber(usage.activeCount),
          formatNumber(usage.activeSoftCap),
        )}
      </AlertDescription>
    </Alert>
  );
}
