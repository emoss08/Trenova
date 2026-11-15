import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { AccessorialChargesPage } from "@/routes/accessorial-charge/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/accessorial-charges")({
  beforeLoad: gates(requirePermission(Resource.AccessorialCharge)),
  component: AccessorialChargesPage,
});
