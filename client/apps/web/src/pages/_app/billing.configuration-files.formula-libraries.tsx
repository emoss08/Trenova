import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FormulaLibrariesPage } from "@/routes/formula-library/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/formula-libraries")({
  beforeLoad: gates(requirePermission(Resource.FormulaLibrary)),
  component: FormulaLibrariesPage,
});
