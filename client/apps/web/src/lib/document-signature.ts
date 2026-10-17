import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";
import {
  documentSignatureStatusSchema,
  type Document,
  type DocumentSignatureStatus,
} from "@trenova/shared/types/document";
import { z } from "zod";

export const DOCUMENT_SIGNATURE_PHASE: Record<DocumentSignatureStatus, StatusPhase> = {
  Signed: "complete",
  Unsigned: "attention",
  Unknown: "awaiting",
};

export const DOCUMENT_SIGNATURE_LABEL: Record<DocumentSignatureStatus, string> = defineLabels({
  Signed: "Signed",
  Unsigned: "Unsigned",
  Unknown: "Signature not read",
});

export const DOCUMENT_SIGNATURE_STATUSES: readonly DocumentSignatureStatus[] = [
  "Signed",
  "Unsigned",
  "Unknown",
];

export function documentNeedsSignature(
  document: Pick<Document, "documentType" | "isCurrentVersion">,
): boolean {
  return document.isCurrentVersion && document.documentType?.requiresSignature === true;
}

export function documentSignatureSetByPerson(document: Pick<Document, "signatureSource">): boolean {
  return document.signatureSource === "Person";
}

export const documentSignatureFormSchema = z
  .object({
    status: documentSignatureStatusSchema,
    signedAt: z.number().nullable(),
  })
  .superRefine((values, ctx) => {
    if (values.status !== "Signed" && values.signedAt !== null) {
      ctx.addIssue({
        code: "custom",
        path: ["signedAt"],
        message: translate("Only a signed document has a signing date"),
      });
    }
  });

export type DocumentSignatureFormValues = z.infer<typeof documentSignatureFormSchema>;

export function documentSignatureFormValues(
  document: Pick<Document, "signatureStatus" | "signedAt">,
): DocumentSignatureFormValues {
  return {
    status: document.signatureStatus,
    signedAt: document.signatureStatus === "Signed" ? (document.signedAt ?? null) : null,
  };
}
