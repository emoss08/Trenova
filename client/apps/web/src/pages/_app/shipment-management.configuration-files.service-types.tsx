import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ServiceTypesPage } from "@/routes/service-type/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/shipment-management/configuration-files/service-types")(
  {
    beforeLoad: gates(requirePermission(Resource.ServiceType)),
    component: ServiceTypesPage,
  },
);
