import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FactoringCompanyCreatePage } from "@/routes/factoring-company/new/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/factoring-companies_/new")({
  beforeLoad: gates(requirePermission(Resource.FactoringCompany, Operation.Create)),
  component: FactoringCompanyCreatePage,
});
