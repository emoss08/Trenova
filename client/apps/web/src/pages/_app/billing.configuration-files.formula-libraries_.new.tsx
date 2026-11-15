import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FormulaLibraryCreatePage, prefetch } from "@/routes/formula-library/new/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/formula-libraries_/new")({
  beforeLoad: gates(requirePermission(Resource.FormulaLibrary, Operation.Create)),
  staticData: { frame: "room", roomRail: "stands" },
  component: FormulaLibraryCreatePage,
  loader: pageLoader(prefetch),
});
