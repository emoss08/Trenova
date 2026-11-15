import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DocumentTypesPage } from "@/routes/document-type/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/document-types")({
  beforeLoad: gates(requirePermission(Resource.DocumentType)),
  component: DocumentTypesPage,
});
