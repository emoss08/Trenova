import { usePermission } from "@/hooks/use-permission";
import { usePlanRestrictions } from "@/hooks/use-plan-restrictions";
import { isPlanRestricted, PlanCapability } from "@/lib/plan-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";

/**
 * Whether the signed-in person can add an AI provider: they may create one, may open
 * AI control (the page the providers tab is on), and the plan does not withhold agent
 * automation, which that page requires.
 */
export function useCanAddAIProvider(): boolean {
  const { allowed: canCreateProvider } = usePermission(Resource.AIProvider, Operation.Create);
  const { allowed: canOpenAgentControl } = usePermission(Resource.AgentControl, Operation.Read);
  const restrictions = usePlanRestrictions();

  return (
    canCreateProvider &&
    canOpenAgentControl &&
    !isPlanRestricted(restrictions, PlanCapability.AgentAutomation)
  );
}
