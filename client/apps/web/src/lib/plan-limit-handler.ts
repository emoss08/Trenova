import { decidePlanLimit, usePlanLimitStore } from "@/stores/plan-limit-store";
import { setPlanLimitHandler, type PlanLimitNotice } from "@trenova/shared/lib/plan-limit";
import { useAuthStore } from "@trenova/shared/stores/auth-store";

/** The handler the REST and GraphQL transports call with every plan-limit refusal. */
export function handlePlanLimit(notice: PlanLimitNotice): boolean {
  const store = usePlanLimitStore.getState();
  const decision = decidePlanLimit({
    notice,
    isAuthenticated: useAuthStore.getState().isAuthenticated,
    openNotice: store.notice,
    lastDismissed: store.lastDismissed,
    now: Date.now(),
  });

  if (decision === "open") {
    store.show(notice);
  }
  return decision !== "ignore";
}

export function installPlanLimitHandler(): void {
  setPlanLimitHandler(handlePlanLimit);
}
