/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import { http } from "@/lib/http-client";
import { type TableSheetProps } from "@/types/data-table";
import { faFileCsv } from "@fortawesome/pro-solid-svg-icons";

type DataTableImportModalProps = TableSheetProps & {
  name: string;
  exportModelName: string;
};

export function DataTableImportModal({
  open,
  onOpenChange,
  name,
  exportModelName,
}: DataTableImportModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Import {name}</DialogTitle>
          <DialogDescription>Import {name}s from a file</DialogDescription>
        </DialogHeader>
        <DialogBody>
          <DataTableTemplateButton exportModelName={exportModelName} />
        </DialogBody>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button type="submit">Import</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function DataTableTemplateButton({
  exportModelName,
}: {
  exportModelName: string;
}) {
  const handleDownload = async () => {
    try {
      await http.downloadFile(`/reporting/template?entity=${exportModelName}`, {
        filename: `${exportModelName}-${crypto.randomUUID()}.csv`,
      });
    } catch (error) {
      console.error("Error downloading template:", error);
    }
  };

  return (
    <Button
      onClick={handleDownload}
      className="flex items-center gap-2"
      variant="ghost"
    >
      <Icon className="size-4" icon={faFileCsv} />
      <span>Download Template</span>
    </Button>
  );
}
