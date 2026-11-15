import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { EquipmentManufacturersPage } from "@/routes/equipment-manufacturer/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/equipment/configuration-files/equipment-manufacturers")(
  {
    beforeLoad: gates(requirePermission(Resource.EquipmentManufacturer)),
    component: EquipmentManufacturersPage,
  },
);
