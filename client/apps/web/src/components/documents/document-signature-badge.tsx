import { useT } from "@trenova/shared/i18n/use-t";
import { DOCUMENT_SIGNATURE_LABEL, DOCUMENT_SIGNATURE_PHASE } from "@/lib/document-signature";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";

type DocumentSignatureBadgeProps = {
  document: Pick<Document, "signatureStatus" | "signedAt" | "signatureSource">;
  className?: string;
};

export function DocumentSignatureBadge({ document, className }: DocumentSignatureBadgeProps) {
  const t = useT();
  const status = document.signatureStatus;
  const signedOn =
    status === "Signed" && document.signedAt ? formatUnixDateMedium(document.signedAt) : null;
  const byPerson = document.signatureSource === "Person";
  const byExtraction = document.signatureSource === "Extraction";
  const title = signedOn
    ? byPerson
      ? t("Signed on {0}, marked by a person", signedOn)
      : t("Signed on {0}, read from the document", signedOn)
    : byPerson
      ? t("Marked by a person")
      : byExtraction
        ? t("Read from the document")
        : t("Nobody has read the signature yet");

  return (
    <Badge
      variant={phaseTone(DOCUMENT_SIGNATURE_PHASE[status])}
      title={title}
      className={cn("h-5 px-1.5 py-0 text-2xs", className)}
    >
      {DOCUMENT_SIGNATURE_LABEL[status]}
    </Badge>
  );
}
