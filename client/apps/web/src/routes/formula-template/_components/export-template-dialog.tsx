import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  buildTemplateExport,
  downloadJson,
  getExportFilename,
} from "@/lib/formula-template-export";
import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@/types/formula-template";
import { DownloadIcon, Loader2Icon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";

type ExportTemplateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: FormulaTemplate | null;
};

export function ExportTemplateDialog({ open, onOpenChange, template }: ExportTemplateDialogProps) {
  const [includeVersionHistory, setIncludeVersionHistory] = useState(false);
  const [isExporting, setIsExporting] = useState(false);

  const handleExport = useCallback(async () => {
    if (!template?.id) return;

    setIsExporting(true);
    const versionsPromise = includeVersionHistory
      ? apiService.formulaTemplateService
          .listVersions(template.id, { limit: 1000 })
          .then((response) => response.results)
      : Promise.resolve(undefined);

    await versionsPromise
      .then((versions) => {
        const exportData = buildTemplateExport(template, versions);
        const filename = getExportFilename(template, includeVersionHistory);
        downloadJson(exportData, filename);

        toast.success("Template exported successfully", {
          description: filename,
        });
        onOpenChange(false);
      })
      .catch(() => {
        toast.error("Export failed", {
          description: "Could not export the template. Please try again.",
        });
      })
      .finally(() => {
        setIsExporting(false);
      });
  }, [template, includeVersionHistory, onOpenChange]);

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
            Export Template
          </DialogTitle>
          <DialogDescription>
            Export &ldquo;{template?.name}&rdquo; as a JSON file. You can import this template later
            or share it with others.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <label className="flex cursor-pointer items-center gap-3">
            <Checkbox
              checked={includeVersionHistory}
              onCheckedChange={(checked) => setIncludeVersionHistory(checked === true)}
            />
            <div className="flex flex-col">
              <span className="text-sm font-medium">Include version history</span>
              <span className="text-xs text-muted-foreground">
                Export all versions with change messages and timestamps
              </span>
            </div>
          </label>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button onClick={handleExport} disabled={isExporting}>
            {isExporting ? (
              <>
                <Loader2Icon className="size-4 animate-spin" />
                Exporting...
              </>
            ) : (
              <>
                <DownloadIcon className="size-4" />
                Export
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
