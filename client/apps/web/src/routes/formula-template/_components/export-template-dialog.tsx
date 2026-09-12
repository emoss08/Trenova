import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import {
  buildTemplateExport,
  downloadJson,
  getExportFilename,
} from "@/lib/formula-template-export";
import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { DownloadIcon, Loader2Icon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

type ExportTemplateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: Pick<FormulaTemplate, "id" | "name"> | null;
};

export function ExportTemplateDialog({ open, onOpenChange, template }: ExportTemplateDialogProps) {
  const t = useT();

  const [includeVersionHistory, setIncludeVersionHistory] = useState(false);
  const [isExporting, setIsExporting] = useState(false);

  const handleExport = useCallback(async () => {
    if (!template?.id) return;

    setIsExporting(true);
    const templatePromise = apiService.formulaTemplateService.get(template.id);
    const versionsPromise = includeVersionHistory
      ? apiService.formulaTemplateService
          .listVersions(template.id, { limit: 1000 })
          .then((response) => response.results)
      : Promise.resolve(undefined);
    const testCasesPromise = apiService.formulaTemplateService.listTestCases(template.id);

    await Promise.all([templatePromise, versionsPromise, testCasesPromise])
      .then(([fullTemplate, versions, testCases]) => {
        const exportData = buildTemplateExport(fullTemplate, { versions, testCases });
        const filename = getExportFilename(fullTemplate, includeVersionHistory);
        downloadJson(exportData, filename);

        toast.success(t("Template exported successfully"), {
          description: filename,
        });
        onOpenChange(false);
      })
      .catch(() => {
        toast.error(t("Export failed"), {
          description: t("Could not export the template. Please try again."),
        });
      })
      .finally(() => {
        setIsExporting(false);
      });
  }, [template, includeVersionHistory, onOpenChange, t]);

  const handleClose = () => {
    onOpenChange(false);
    setIncludeVersionHistory(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <DownloadIcon className="size-4" />
            {t("Export Template")}
          </DialogTitle>
          <DialogDescription>
            {t(
              "Export “{0}” as a JSON file, including its test scenarios. You can import this template later or share it with others.",
              template?.name,
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <label className="flex cursor-pointer items-center gap-3">
            <Checkbox
              checked={includeVersionHistory}
              onCheckedChange={(checked) => setIncludeVersionHistory(checked === true)}
            />
            <div className="flex flex-col">
              <span className="text-sm font-medium">{t("Include version history")}</span>
              <span className="text-muted-foreground text-xs">
                {t("Export all versions with change messages and timestamps")}
              </span>
            </div>
          </label>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            {t("Cancel")}
          </Button>
          <Button onClick={handleExport} disabled={isExporting}>
            {isExporting ? (
              <>
                <Loader2Icon className="size-4 animate-spin" />
                {t("Exporting...")}
              </>
            ) : (
              <>
                <DownloadIcon className="size-4" />
                {t("Export")}
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
