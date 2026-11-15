import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { useCanAddAIProvider } from "@/hooks/use-can-add-ai-provider";
import { AI_SETUP_PATH } from "@/lib/ai-setup";
import { PlanFeature } from "@/lib/plan-feature";
import { documentControl } from "@/lib/queries/document-control";
import { extractionModelReasonSchema } from "@/types/document-control";
import { useQuery } from "@tanstack/react-query";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { Link } from "react-router";

/**
 * Whether uploaded documents are read. Reading is Document Intelligence: on a plan without it
 * nothing is read, and the server is not asked.
 */
export function useDocumentExtractionAvailable(): boolean {
  const onPlan = usePlanFeature(PlanFeature.DocumentIntelligence).available;
  const { data } = useQuery({ ...documentControl.extractionStatus(), enabled: onPlan });

  return onPlan && (data?.extractionAvailable ?? true);
}

/**
 * Why uploaded documents are not being read, on a plan that reads them. A plan without
 * Document Intelligence never reads them, so there is nothing to explain and no alert; the
 * fix it names is Nova's only on a plan with Trenova AI.
 */
export function ExtractionNeedsProviderAlert({ className }: { className?: string }) {
  const t = useT();
  const onPlan = usePlanFeature(PlanFeature.DocumentIntelligence).available;
  const novaOnPlan = usePlanFeature(PlanFeature.AIDesk).available;
  const { data } = useQuery({ ...documentControl.extractionStatus(), enabled: onPlan });
  const canAddProvider = useCanAddAIProvider();

  if (!onPlan || !data || data.extractionAvailable) {
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

  if (data.modelProviderReason === extractionModelReasonSchema.enum.off_on_server) {
    return (
      <Alert variant="warning" size="sm" className={className}>
        <AlertCircleIcon />
        <AlertTitle>{t("Extraction is turned off on this server")}</AlertTitle>
        <AlertDescription>
          <p>
            {t(
              "Document extraction is switched off in this server's configuration, so uploaded files are stored but not read and no shipment draft is built.",
            )}
          </p>
          <p>{t("Ask an administrator to set ai.documentExtraction to true.")}</p>
        </AlertDescription>
      </Alert>
    );
  }

  if (data.modelProviderReason === extractionModelReasonSchema.enum.not_in_plan) {
    return (
      <Alert variant="warning" size="sm" className={className}>
        <AlertCircleIcon />
        <AlertTitle>{t("Extraction is not in your plan")}</AlertTitle>
        <AlertDescription>
          <p>
            {t(
              "Your plan does not include document extraction, so uploaded files are stored but not read and no shipment draft is built.",
            )}
          </p>
        </AlertDescription>
      </Alert>
    );
  }

  if (!novaOnPlan) {
    return (
      <Alert variant="warning" size="sm" className={className}>
        <AlertCircleIcon />
        <AlertTitle>{t("Extraction needs an AI provider")}</AlertTitle>
        <AlertDescription>
          <p>
            {t(
              "No AI provider is chosen for reading documents, so uploaded files are stored but not read and no shipment draft is built.",
            )}
          </p>
          <p>{t("Ask an administrator to choose an AI provider.")}</p>
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Alert variant="warning" size="sm" className={className}>
      <AlertCircleIcon />
      <AlertTitle>{t("Extraction needs Nova")}</AlertTitle>
      <AlertDescription>
        <p>
          {t(
            "Nova reads uploaded documents, and no provider is chosen for Nova, so uploaded files are stored but not read and no shipment draft is built. Nova runs on an Anthropic provider's key.",
          )}
        </p>
        {canAddProvider ? (
          <Link to={AI_SETUP_PATH} className="text-foreground underline underline-offset-2">
            {t("Choose a provider for Nova")}
          </Link>
        ) : (
          <p>{t("Ask an administrator to choose a provider for Nova.")}</p>
        )}
      </AlertDescription>
    </Alert>
  );
}
