import { translate } from "@trenova/shared/i18n/runtime";
import type { Document, DocumentShipmentDraft } from "@trenova/shared/types/document";

export function isBlockedByProvider(
  document: Document | null | undefined,
  extractionAvailable: boolean,
): boolean {
  return (
    !extractionAvailable &&
    document?.shipmentDraftStatus === "Unavailable" &&
    document.contentStatus === "Indexed" &&
    !document.hasExtractedText
  );
}

export function processingSummary(
  document: Document | null | undefined,
  draft: DocumentShipmentDraft | null | undefined,
  extractionAvailable = true,
) {
  if (!document) {
    return {
      title: translate("Waiting for uploaded document"),
      description: translate("The upload must complete before extraction can start."),
      progress: 10,
      variant: "default" as const,
    };
  }

  if (document.contentStatus === "Failed") {
    return {
      title: translate("Extraction failed"),
      description:
        document.contentError || "We could not extract text from this rate confirmation.",
      progress: 100,
      variant: "error" as const,
    };
  }

  if (draft?.status === "Failed") {
    return {
      title: translate("Draft generation failed"),
      description:
        draft.failureMessage || "We extracted text but could not build a shipment draft.",
      progress: 100,
      variant: "error" as const,
    };
  }

  if (draft?.status === "Ready") {
    return {
      title: translate("Shipment draft ready"),
      description: translate("The extracted shipment draft is ready for review."),
      progress: 100,
      variant: "success" as const,
    };
  }

  if (isBlockedByProvider(document, extractionAvailable)) {
    return {
      title: translate("Extraction needs a model provider"),
      description: translate(
        "The file is stored, but no AI provider can read documents for your organization, so no shipment draft was built.",
      ),
      progress: 100,
      variant: "error" as const,
    };
  }

  if (
    document.shipmentDraftStatus === "Unavailable" &&
    document.contentStatus !== "Pending" &&
    document.contentStatus !== "Extracting"
  ) {
    return {
      title: translate("No shipment draft available"),
      description: translate("This file did not produce a usable shipment draft."),
      progress: 100,
      variant: "error" as const,
    };
  }

  if (document.contentStatus === "Pending") {
    return {
      title: translate("Preparing extraction"),
      description: translate(
        "We are queuing OCR and intelligence work for this rate confirmation.",
      ),
      progress: 35,
      variant: "default" as const,
    };
  }

  if (document.contentStatus === "Extracting") {
    return {
      title: translate("Extracting shipment details"),
      description: translate(
        "We are extracting text, classifying the document, and assembling the shipment draft.",
      ),
      progress: 70,
      variant: "default" as const,
    };
  }

  if (document.shipmentDraftStatus === "Pending") {
    return {
      title: translate("Building shipment draft"),
      description: translate(
        "Extraction finished. We are mapping the results into a shipment draft now.",
      ),
      progress: 85,
      variant: "default" as const,
    };
  }

  return {
    title: translate("Processing rate confirmation"),
    description: translate("We are still evaluating the uploaded document."),
    progress: 55,
    variant: "default" as const,
  };
}
