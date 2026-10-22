import { useT } from "@trenova/shared/i18n/use-t";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import type { ImportTemplatePayload, ImportTemplatesRequest } from "@/services/formula-template";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import type { ImportTemplatesResponse } from "@trenova/shared/types/formula-template";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { FileUploadIcon, Upload01Icon } from "@trenova/shared/components/icons";
import { useRef, useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { translate } from "@trenova/shared/i18n/runtime";
import { FORMULA_TEMPLATE_EXPORT_VERSION } from "@/lib/formula-template-export";
import { z } from "zod";

const exportedTemplateSchema = z.object({
  name: z.string().trim().min(1),
  description: z.string().optional(),
  source: z.string().refine((value) => value.trim() !== ""),
  sourceTemplateId: z.string().nullish(),
  sourceVersionNumber: z.number().int().nullish(),
});

type ParsedImport = {
  templates: ImportTemplatePayload[];
  exportVersion: typeof FORMULA_TEMPLATE_EXPORT_VERSION;
  filename: string;
};

/**
 * Reads an export file: one template or many, each a name and a Tariff source (its tests are
 * in it). The server checks each source again when it creates the template.
 */
export function parseImportFile(filename: string, raw: string): ParsedImport {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error(translate("The file is not a formula template export"));
  }
  if (typeof parsed !== "object" || parsed === null) {
    throw new Error(translate("The file is not a formula template export"));
  }

  const record = parsed as Record<string, unknown>;
  if (record.exportVersion !== FORMULA_TEMPLATE_EXPORT_VERSION) {
    throw new Error(
      translate(
        "This export is from another version of Trenova; export the template again and import that file.",
      ),
    );
  }

  const candidates: unknown[] = [];
  if (record.template && typeof record.template === "object") candidates.push(record.template);
  if (Array.isArray(record.templates)) candidates.push(...record.templates);
  if (candidates.length === 0) {
    throw new Error(translate("The export contains no templates"));
  }

  const templates = candidates.map((candidate): ImportTemplatePayload => {
    const result = exportedTemplateSchema.safeParse(candidate);
    if (!result.success) {
      throw new Error(translate("Every template needs a name and a source"));
    }
    return result.data;
  });

  return { templates, exportVersion: FORMULA_TEMPLATE_EXPORT_VERSION, filename };
}

type ImportTemplateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onImported?: (response: ImportTemplatesResponse) => void;
};

export function ImportTemplateDialog({
  open,
  onOpenChange,
  onImported,
}: ImportTemplateDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [parsed, setParsed] = useState<ParsedImport | null>(null);
  const [parseError, setParseError] = useState<string | null>(null);
  const [renameOnConflict, setRenameOnConflict] = useState(true);

  const { mutate, isPending } = useMutation<ImportTemplatesResponse, Error, ImportTemplatesRequest>(
    {
      mutationFn: (request) => apiService.formulaTemplateService.importTemplates(request),
      onSuccess: async (response) => {
        const renamedCount = Object.keys(response.renamed ?? {}).length;
        toast.success(
          t(
            "{0, plural, one {Imported # template} other {Imported # templates}}",
            response.created.length,
          ),
          {
            description:
              renamedCount > 0
                ? t(
                    "{0} renamed to avoid name conflicts. Imported templates start as drafts.",
                    renamedCount,
                  )
                : t("Imported templates start as drafts."),
          },
        );
        await invalidateFormulaTemplate(queryClient);
        onImported?.(response);
        handleClose();
      },
      onError: (error) => {
        toast.error(t("Import failed"), {
          description: error.message || translate("The export could not be imported."),
        });
      },
    },
  );

  const handleClose = () => {
    setParsed(null);
    setParseError(null);
    onOpenChange(false);
  };

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    try {
      const raw = await file.text();
      setParsed(parseImportFile(file.name, raw));
      setParseError(null);
    } catch (error) {
      setParsed(null);
      setParseError(error instanceof Error ? error.message : "The file could not be read");
    }
  };

  const handleImport = () => {
    if (!parsed) return;
    mutate({
      exportVersion: parsed.exportVersion,
      templates: parsed.templates,
      onConflict: renameOnConflict ? "rename" : "reject",
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) handleClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FileUploadIcon className="size-4" />
            {t("Import templates")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "Import a formula template export. Imported templates are created as drafts and must go through review before they can price shipments.",
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 py-1">
          <input
            ref={fileInputRef}
            type="file"
            accept="application/json,.json"
            onChange={(event) => void handleFileChange(event)}
            className="hidden"
          />
          <Button
            type="button"
            variant="outline"
            onClick={() => fileInputRef.current?.click()}
            className="w-full gap-2"
          >
            <Upload01Icon className="size-4" />
            {parsed ? parsed.filename : t("Choose export file...")}
          </Button>

          {parseError && <p className="text-destructive text-xs">{parseError}</p>}

          {parsed && (
            <div className="space-y-2">
              <div className="max-h-48 overflow-y-auto rounded-md border">
                {parsed.templates.map((template, index) => (
                  <div
                    key={`${template.name}-${index}`}
                    className="flex items-center justify-between gap-2 border-b px-3 py-1.5 text-xs last:border-b-0"
                  >
                    <span className="truncate font-medium">{template.name}</span>
                    {template.description && (
                      <span className="text-muted-foreground truncate">
                        {template.description}
                      </span>
                    )}
                  </div>
                ))}
              </div>

              <div className="flex items-center justify-between gap-2 rounded-md border px-3 py-2">
                <Label htmlFor="rename-on-conflict" className="text-xs">
                  {t("Rename when a template with the same name exists")}
                </Label>
                <Switch
                  id="rename-on-conflict"
                  size="sm"
                  checked={renameOnConflict}
                  onCheckedChange={setRenameOnConflict}
                />
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            onClick={handleImport}
            disabled={!parsed}
            isLoading={isPending}
            loadingText={t("Importing...")}
          >
            {t("Import")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
