import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { usePermission } from "@/hooks/use-permission";
import { documentControl } from "@/lib/queries/document-control";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";

export function useDocumentExtractionAvailable(): boolean {
  const { data } = useQuery({ ...documentControl.extractionStatus() });

  return data?.extractionAvailable ?? true;
}

export function ExtractionNeedsProviderAlert({ className }: { className?: string }) {
  const t = useT();
  const available = useDocumentExtractionAvailable();
  const { allowed: canManageProviders } = usePermission(Resource.AgentControl, Operation.Read);

  if (available) {
    return null;
  }

  return (
    <Alert variant="warning" size="sm" className={className}>
      <AlertCircleIcon />
      <AlertTitle>{t("Extraction needs a model provider")}</AlertTitle>
      <AlertDescription>
        <p>
          {t(
            "No AI provider can read documents for your organization, so uploaded files are stored but not read and no shipment draft is built.",
          )}
        </p>
        {canManageProviders ? (
          <Link to="/admin/agent-control" className="text-foreground underline underline-offset-2">
            {t("Add an AI provider")}
          </Link>
        ) : (
          <p>{t("Ask an administrator to add an AI provider.")}</p>
        )}
      </AlertDescription>
    </Alert>
  );
}
