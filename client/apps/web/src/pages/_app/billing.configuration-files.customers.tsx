import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { CustomersPage } from "@/routes/customer/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/customers")({
  beforeLoad: gates(requirePermission(Resource.Customer)),
  component: CustomersPage,
});
