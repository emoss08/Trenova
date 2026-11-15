import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { CarrierInvoiceMatchingPage } from "@/routes/carrier-invoice-matching/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/carrier-settlements/invoice-matching")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.Brokerage),
    requirePermission(Resource.CarrierInvoiceMatch),
  ),
  component: CarrierInvoiceMatchingPage,
});
