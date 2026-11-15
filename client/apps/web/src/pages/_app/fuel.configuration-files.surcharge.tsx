import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FuelManagementPage } from "@/routes/fuel-management/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/fuel/configuration-files/surcharge")({
  beforeLoad: gates(requirePermission(Resource.FuelSurchargeProgram)),
  component: FuelManagementPage,
});
