import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { useCanAddAIProvider } from "@/hooks/use-can-add-ai-provider";
import { AI_SETUP_PATH } from "@/lib/ai-setup";
import { documentControl } from "@/lib/queries/document-control";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";

export function useDocumentExtractionAvailable(): boolean {
  const { data } = useQuery({ ...documentControl.extractionStatus() });

  return data?.extractionAvailable ?? true;
}

export function ExtractionNeedsProviderAlert({ className }: { className?: string }) {
  const t = useT();
  const { data } = useQuery({ ...documentControl.extractionStatus() });
  const canAddProvider = useCanAddAIProvider();

  if (!data || data.extractionAvailable) {
    return null;
  }

  if (!data.converterAvailable) {
    return (
      <Alert variant="warning" size="sm" className={className}>
        <AlertCircleIcon />
        <AlertTitle>{t("Extraction is not set up on this server")}</AlertTitle>
        <AlertDescription>
          <p>
            {t(
              "The document converter is not configured, so uploaded files are stored but not read and no shipment draft is built.",
            )}
          </p>
          <p>{t("Ask an administrator to configure the document converter.")}</p>
        </AlertDescription>
      </Alert>
    );
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
        {canAddProvider ? (
          <Link to={AI_SETUP_PATH} className="text-foreground underline underline-offset-2">
            {t("Add an AI provider")}
          </Link>
        ) : (
          <p>{t("Ask an administrator to add an AI provider.")}</p>
        )}
      </AlertDescription>
    </Alert>
  );
}
