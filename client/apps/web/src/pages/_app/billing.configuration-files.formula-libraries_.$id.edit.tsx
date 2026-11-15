import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FormulaLibraryEditPage, prefetch } from "@/routes/formula-library/[id]/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute(
  "/_app/billing/configuration-files/formula-libraries_/$id/edit",
)({
  beforeLoad: gates(requirePermission(Resource.FormulaLibrary)),
  staticData: { frame: "room", roomRail: "stands" },
  component: FormulaLibraryEditPage,
  loader: pageLoader(prefetch),
});
