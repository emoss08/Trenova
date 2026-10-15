import { usePermission } from "@/hooks/use-permission";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";

/** Whether the person has turned the assistant off in Settings. */
export function useAssistantHidden(): boolean {
  return useAuthStore((state) => state.user?.assistantHidden === true);
}

/**
 * Whether the assistant that lives on every page is there at all: the person
 * may use it and has not turned it off. When this is false nothing of it is
 * mounted, and nothing else offers to open it.
 */
export function useAssistantEnabled(): boolean {
  const { allowed } = usePermission(Resource.Assistant, Operation.Read);
  const hidden = useAssistantHidden();
  return allowed && !hidden;
}
