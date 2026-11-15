import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FormulaTemplatesPage } from "@/routes/formula-template/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/formula-templates")({
  beforeLoad: gates(requirePermission(Resource.FormulaTemplate)),
  component: FormulaTemplatesPage,
});
