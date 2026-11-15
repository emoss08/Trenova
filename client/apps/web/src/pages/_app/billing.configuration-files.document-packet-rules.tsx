import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DocumentPacketRulesPage } from "@/routes/document-packet-rule/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/document-packet-rules")({
  beforeLoad: gates(requirePermission(Resource.DocumentType)),
  component: DocumentPacketRulesPage,
});
