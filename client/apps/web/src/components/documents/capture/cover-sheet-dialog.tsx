import { ControlledDocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  captureDocumentCategory,
  captureRecordKindLabel,
  type CaptureRecordKind,
} from "@/lib/capture";
import { buildCoverSheetPdf, type CoverSheetPrint } from "@/lib/capture-cover-sheet";
import { createCaptureCoverSheets, type IssuedCoverSheet } from "@/lib/graphql/capture";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import {
  NumberField,
  NumberFieldDecrement,
  NumberFieldGroup,
  NumberFieldIncrement,
  NumberFieldInput,
} from "@trenova/shared/components/ui/number-field";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useState } from "react";
import { toast } from "sonner";

/** The most sheets one print run asks for; the server refuses more than fifty. */
const MAX_COPIES = 20;

/**
 * Opens the finished sheets in a new tab to print. The browser's own viewer is
 * what prints them; a download is offered only if the tab was blocked.
 */
function openForPrinting(bytes: Uint8Array, fileName: string) {
  const url = URL.createObjectURL(new Blob([new Uint8Array(bytes)], { type: "application/pdf" }));
  const tab = window.open(url, "_blank", "noopener");
  if (tab === null) {
    const link = document.createElement("a");
    link.href = url;
    link.download = fileName;
    link.click();
  }
  window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
}

/**
 * Prints cover sheets for this record. A sheet on top of a stack of paper
 * files everything after it here, until the next sheet, wherever the stack is
 * scanned. Each sheet's code is issued once; a lost sheet is replaced with a
 * new one, not reprinted.
 */
export function CoverSheetDialog({
  open,
  onOpenChange,
  kind,
  recordId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  kind: CaptureRecordKind;
  recordId: string;
}) {
  const t = useT();
  const [copies, setCopies] = useState(1);
  const [documentTypeId, setDocumentTypeId] = useState("");
  const [documentTypeName, setDocumentTypeName] = useState<string | null>(null);

  const print = useApiMutation({
    mutationFn: async () => {
      const issued = await createCaptureCoverSheets(
        Array.from({ length: copies }, () => ({
          targetType: kind,
          targetId: recordId,
          documentTypeId: documentTypeId === "" ? null : documentTypeId,
        })),
      );
      const sheets: CoverSheetPrint[] = issued.map((sheet: IssuedCoverSheet) => ({
        id: sheet.id,
        modules: sheet.qrCode.modules,
        record: sheet.target
          ? {
              kind: captureRecordKindLabel(t, kind),
              title: sheet.target.title,
              subtitle: sheet.target.subtitle,
            }
          : { kind: captureRecordKindLabel(t, kind), title: recordId, subtitle: "" },
        documentType: documentTypeName,
        expiresOn: formatUnixDateMedium(sheet.expiresAt),
      }));

      return buildCoverSheetPdf(sheets, {
        heading: t("Trenova cover sheet"),
        separatorTitle: t("Separator"),
        documentTypeLabel: t("Document type"),
        instructions: t(
          "Put this sheet on top of the pages that belong here and scan the stack. Every page after it is filed onto this record, until the next cover sheet.",
        ),
        separatorInstructions: t("Put this sheet between two documents to split the stack there."),
        expires: (date) => t("Use by {0}", date),
      });
    },
    onSuccess: (bytes) => {
      openForPrinting(bytes, `cover-sheets-${kind}.pdf`);
      toast.success(
        t(
          "{0, plural, one {Cover sheet ready to print} other {# cover sheets ready to print}}",
          copies,
        ),
      );
      onOpenChange(false);
    },
    resourceName: "Cover sheet",
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Print cover sheets")}</DialogTitle>
          <DialogDescription>
            {t(
              "Put a sheet on top of the paper for this record. When the stack is scanned, anywhere, the pages after it are filed here.",
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <span className="text-foreground-subtle text-xs font-medium">{t("Sheets")}</span>
            <NumberField
              value={copies}
              min={1}
              max={MAX_COPIES}
              step={1}
              onValueChange={(value) =>
                setCopies(Math.min(MAX_COPIES, Math.max(1, Math.trunc(value ?? 1))))
              }
            >
              <NumberFieldGroup>
                <NumberFieldDecrement />
                <NumberFieldInput aria-label={t("Sheets")} />
                <NumberFieldIncrement />
              </NumberFieldGroup>
            </NumberField>
          </div>
          <ControlledDocumentTypeAutocompleteField
            label={t("Document type")}
            placeholder={t("Optional")}
            category={captureDocumentCategory(kind)}
            value={documentTypeId}
            onValueChange={setDocumentTypeId}
            onOptionChange={(option) =>
              setDocumentTypeName(
                option ? selectOptionMetaString(option, "name") || option.label : null,
              )
            }
          />
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            onClick={() => print.mutate(undefined)}
            isLoading={print.isPending}
            loadingText={t("Preparing")}
          >
            {t("Print")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
