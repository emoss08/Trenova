import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import type { ImportTemplatePayload, ImportTemplatesRequest } from "@/services/formula-template";
import { Badge } from "@trenova/shared/components/ui/badge";
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
import { FileUpIcon, UploadIcon } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "sonner";

type ParsedImport = {
  templates: ImportTemplatePayload[];
  exportVersion: string;
  filename: string;
};

function parseImportFile(filename: string, raw: string): ParsedImport {
  const parsed: unknown = JSON.parse(raw);
  if (typeof parsed !== "object" || parsed === null) {
    throw new Error("The file is not a formula template export");
  }

  const record = parsed as Record<string, unknown>;
  const exportVersion = typeof record.exportVersion === "string" ? record.exportVersion : "";
  if (exportVersion !== "1.0" && exportVersion !== "1.1") {
    throw new Error("Unsupported export version. Expected a Trenova formula template export.");
  }

  const templates: ImportTemplatePayload[] = [];
  if (record.template && typeof record.template === "object") {
    templates.push(record.template as ImportTemplatePayload);
  }
  if (Array.isArray(record.templates)) {
    templates.push(...(record.templates as ImportTemplatePayload[]));
  }

  if (templates.length === 0) {
    throw new Error("The export contains no templates");
  }

  for (const template of templates) {
    if (!template.name || !template.expression) {
      throw new Error("Every template needs a name and an expression");
    }
  }

  return { templates, exportVersion, filename };
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
          `Imported ${response.created.length} template${response.created.length === 1 ? "" : "s"}`,
          {
            description:
              renamedCount > 0
                ? `${renamedCount} renamed to avoid name conflicts. Imported templates start as drafts.`
                : "Imported templates start as drafts.",
          },
        );
        await invalidateFormulaTemplate(queryClient);
        onImported?.(response);
        handleClose();
      },
      onError: (error) => {
        toast.error("Import failed", {
          description: error.message || "The export could not be imported.",
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
      <DialogContent className="sm:max-w-[440px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FileUpIcon className="size-4" />
            Import Templates
          </DialogTitle>
          <DialogDescription>
            Import a formula template export. Imported templates are created as drafts and must go
            through review before they can price shipments.
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
            <UploadIcon className="size-4" />
            {parsed ? parsed.filename : "Choose export file..."}
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
                    <Badge variant="outline" className="text-2xs shrink-0">
                      {template.type}
                    </Badge>
                  </div>
                ))}
              </div>

              <div className="flex items-center justify-between gap-2 rounded-md border px-3 py-2">
                <Label htmlFor="rename-on-conflict" className="text-xs">
                  Rename when a template with the same name exists
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
          <Button type="button" variant="outline" size="sm" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={handleImport}
            disabled={!parsed}
            isLoading={isPending}
            loadingText="Importing..."
          >
            Import
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
