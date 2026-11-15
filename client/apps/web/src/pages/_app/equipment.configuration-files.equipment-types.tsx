import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { EquipmentTypesPage } from "@/routes/equipment-type/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/equipment/configuration-files/equipment-types")({
  beforeLoad: gates(requirePermission(Resource.EquipmentType)),
  component: EquipmentTypesPage,
});
