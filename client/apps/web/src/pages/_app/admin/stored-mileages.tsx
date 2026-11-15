import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { StoredMileagesPage } from "@/routes/admin/stored-mileages/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/stored-mileages")({
  beforeLoad: gates(requirePermission(Resource.StoredMileage)),
  component: StoredMileagesPage,
});
