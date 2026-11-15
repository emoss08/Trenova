import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { ShipmentImportPage } from "@/routes/shipment/import-page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/shipment-management/shipments_/import")({
  beforeLoad: gates(
    requirePermission(Resource.Shipment, Operation.Create),
    requirePlanFeature(PlanFeature.DocumentIntelligence),
  ),
  component: ShipmentImportPage,
});
