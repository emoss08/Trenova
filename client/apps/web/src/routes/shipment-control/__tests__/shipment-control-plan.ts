import { PlanFeature } from "@/lib/plan-feature";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";

/** Signs the test in on a plan that leaves detention out. */
export function withoutDetention() {
  usePermissionStore.setState({
    manifest: {
      permissions: {},
      routeAccess: {},
      unavailableFeatures: [
        {
          feature: PlanFeature.Detention,
          featureName: "Detention",
          addOn: "OPS",
          addOnName: "Operations",
        },
      ],
    },
  } as never);
}

/** Back to a plan that includes everything. */
export function resetPlan() {
  usePermissionStore.setState({ manifest: null } as never);
}
