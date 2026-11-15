import { createFileRoute } from "@tanstack/react-router";
import { requireManagedInstance } from "@/lib/managed-instance";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { PlanPage, prefetch } from "@/routes/admin/plan/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/plan")({
  beforeLoad: gates(
    requireManagedInstance,
    requirePermission(Resource.Organization, Operation.Update),
  ),
  component: PlanPage,
  loader: pageLoader(prefetch),
});
