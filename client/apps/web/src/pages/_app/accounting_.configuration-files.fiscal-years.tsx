import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FiscalYearsPage } from "@/routes/fiscal-year/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/configuration-files/fiscal-years")({
  beforeLoad: gates(requirePermission(Resource.FiscalYear)),
  component: FiscalYearsPage,
});
