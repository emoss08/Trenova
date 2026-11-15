import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { HoldReasonsPage } from "@/routes/hold-reason/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/hold-reasons")({
  beforeLoad: gates(requirePermission(Resource.HoldReason)),
  component: HoldReasonsPage,
});
