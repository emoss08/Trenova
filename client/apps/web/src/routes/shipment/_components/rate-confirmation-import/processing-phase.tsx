import { BorderBeam } from "@/components/ui/border-beam";
import { Button } from "@/components/ui/button";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { cn } from "@/lib/utils";
import type { Document, DocumentShipmentDraft } from "@/types/document";
import { m } from "motion/react";
import { AlertCircleIcon, CheckIcon, LoaderCircleIcon } from "lucide-react";

type ProcessingPhaseProps = {
  document: Document | null | undefined;
  draft: DocumentShipmentDraft | null | undefined;
  fileName?: string;
  onRetryExtraction: () => void;
  isRetrying: boolean;
  onReplaceFile: () => void;
};

type StepState = "pending" | "active" | "complete" | "error";

type Step = { label: string; state: StepState };

function getSteps(
  doc: Document | null | undefined,
  draft: DocumentShipmentDraft | null | undefined,
): Step[] {
  const hasFailed =
    doc?.contentStatus === "Failed" ||
    draft?.status === "Failed" ||
    (doc &&
      doc.shipmentDraftStatus === "Unavailable" &&
      doc.contentStatus !== "Pending" &&
      doc.contentStatus !== "Extracting");

  if (hasFailed) {
    const failAt = doc?.contentStatus === "Failed" ? 0 : draft?.status === "Failed" ? 2 : 1;
    return [
      { label: "Extracting text", state: failAt === 0 ? "error" : "complete" },
      { label: "Classifying document", state: failAt <= 1 ? (failAt === 1 ? "error" : "pending") : "complete" },
      { label: "Building shipment draft", state: failAt <= 2 ? (failAt === 2 ? "error" : "pending") : "complete" },
    ];
  }

  if (!doc) {
    return [
      { label: "Extracting text", state: "pending" },
      { label: "Classifying document", state: "pending" },
      { label: "Building shipment draft", state: "pending" },
    ];
  }

  if (doc.contentStatus === "Pending") {
    return [
      { label: "Extracting text", state: "active" },
      { label: "Classifying document", state: "pending" },
      { label: "Building shipment draft", state: "pending" },
    ];
  }

  if (doc.contentStatus === "Extracting") {
    return [
      { label: "Extracting text", state: "complete" },
      { label: "Classifying document", state: "active" },
      { label: "Building shipment draft", state: "pending" },
    ];
  }

  if (doc.shipmentDraftStatus === "Pending") {
    return [
      { label: "Extracting text", state: "complete" },
      { label: "Classifying document", state: "complete" },
      { label: "Building shipment draft", state: "active" },
    ];
  }

  return [
    { label: "Extracting text", state: "active" },
    { label: "Classifying document", state: "pending" },
    { label: "Building shipment draft", state: "pending" },
  ];
}

function getErrorMessage(
  doc: Document | null | undefined,
  draft: DocumentShipmentDraft | null | undefined,
): string | null {
  if (doc?.contentStatus === "Failed") {
    return doc.contentError || "Could not extract text from this document.";
  }
  if (draft?.status === "Failed") {
    return draft.failureMessage || "Extracted text but could not build a shipment draft.";
  }
  if (
    doc &&
    doc.shipmentDraftStatus === "Unavailable" &&
    doc.contentStatus !== "Pending" &&
    doc.contentStatus !== "Extracting"
  ) {
    return "This document did not produce a usable shipment draft.";
  }
  return null;
}

function StepDot({ state }: { state: StepState }) {
  return (
    <div className="relative flex items-center justify-center">
      <div
        className={cn(
          "flex size-5 items-center justify-center rounded-full border transition-all duration-500",
          state === "complete" && "border-emerald-500 bg-emerald-500",
          state === "active" && "border-foreground/25",
          state === "pending" && "border-border",
          state === "error" && "border-destructive/50 bg-destructive/10",
        )}
      >
        {state === "complete" && (
          <m.div initial={{ scale: 0 }} animate={{ scale: 1 }} transition={{ type: "spring", stiffness: 500, damping: 25 }}>
            <CheckIcon className="size-2.5 stroke-[3] text-white" />
          </m.div>
        )}
        {state === "active" && <LoaderCircleIcon className="size-2.5 animate-spin text-foreground/70" />}
        {state === "error" && <AlertCircleIcon className="size-2.5 text-destructive" />}
      </div>
      {/* Pulse ring on active */}
      {state === "active" && (
        <m.div
          className="absolute size-5 rounded-full border border-foreground/10"
          animate={{ scale: [1, 1.8], opacity: [0.3, 0] }}
          transition={{ duration: 1.5, repeat: Infinity, ease: "easeOut" }}
        />
      )}
    </div>
  );
}


export function ProcessingPhase({
  document: doc,
  draft,
  fileName,
  onRetryExtraction,
  isRetrying,
  onReplaceFile,
}: ProcessingPhaseProps) {
  const steps = getSteps(doc, draft);
  const errorMessage = getErrorMessage(doc, draft);
  const hasFailed = !!errorMessage;

  return (
    <div className="flex flex-1 flex-col items-center justify-center p-8">
      <m.div
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: "easeOut" }}
        className="w-full max-w-[360px]"
      >
        <div className="relative rounded-xl">
          {!hasFailed && <BorderBeam duration={3} />}

          <div className="relative rounded-xl border bg-background p-6 shadow-xs">
            {/* Header */}
            <div className="text-center">
              {!hasFailed ? (
                <TextShimmer as="span" className="text-[13px] font-medium" duration={2.5}>
                  Analyzing rate confirmation
                </TextShimmer>
              ) : (
                <span className="text-[13px] font-medium text-destructive">Extraction failed</span>
              )}
              {fileName && (
                <m.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.15 }}
                  className="mt-1 truncate text-2xs text-muted-foreground"
                >
                  {fileName}
                </m.p>
              )}
            </div>

            {/* Divider */}
            <div className="my-5 h-px bg-border" />

            {/* Steps */}
            <div className="flex flex-col items-center">
              <div className="inline-flex flex-col">
                {steps.map((step, i) => (
                  <m.div
                    key={step.label}
                    initial={{ opacity: 0, y: 6 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.1, duration: 0.25 }}
                    className="flex gap-3"
                  >
                    {/* Left column: dot + connector */}
                    <div className="flex flex-col items-center">
                      <StepDot state={step.state} />
                      {i < steps.length - 1 && (
                        <m.div
                          className="min-h-4 w-px flex-1 rounded-full"
                          animate={{
                            backgroundColor: step.state === "complete" ? "var(--color-emerald-400)" : "var(--color-border)",
                          }}
                          transition={{ duration: 0.4 }}
                        />
                      )}
                    </div>
                    {/* Right column: label */}
                    <div className={cn(i < steps.length - 1 ? "pb-3" : "")}>
                      <span
                        className={cn(
                          "text-[13px] leading-5 transition-all duration-500",
                          step.state === "complete" && "text-foreground",
                          step.state === "active" && "font-medium text-foreground",
                          step.state === "pending" && "text-muted-foreground/30",
                          step.state === "error" && "text-destructive",
                        )}
                      >
                        {step.label}
                      </span>
                    </div>
                  </m.div>
                ))}
              </div>
            </div>

            {/* Error */}
            {errorMessage && (
              <m.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="mt-5 space-y-3 border-t pt-4">
                <p className="text-xs text-muted-foreground">{errorMessage}</p>
                <div className="flex gap-2">
                  <Button variant="outline" size="sm" onClick={onRetryExtraction} disabled={isRetrying}>
                    {isRetrying && <LoaderCircleIcon className="size-3.5 animate-spin" />}
                    Retry
                  </Button>
                  <Button variant="ghost" size="sm" onClick={onReplaceFile}>
                    Replace file
                  </Button>
                </div>
              </m.div>
            )}
          </div>
        </div>

        {!hasFailed && (
          <m.p
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.6 }}
            className="mt-4 text-center text-2xs text-muted-foreground/40"
          >
            This usually takes a few seconds
          </m.p>
        )}
      </m.div>
    </div>
  );
}
