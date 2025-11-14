import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { http } from "@/lib/http-client";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import type { Resource } from "@/types/audit-entry";
import { faDownload, faFileExcel } from "@fortawesome/pro-solid-svg-icons";
import { useMutation } from "@tanstack/react-query";
import React from "react";
import { toast } from "sonner";

type ReportFormat = "CSV" | "EXCEL";
type DeliveryMethod = "DOWNLOAD" | "EMAIL";

interface GenerateReportRequest {
  resourceType: string;
  name: string;
  format: ReportFormat;
  deliveryMethod: DeliveryMethod;
  filterState: FilterStateSchema;
}

async function generateReport(
  request: GenerateReportRequest,
): Promise<{ reportId: string }> {
  const response = await http.post("/reports/generate/", request);
  return response.data as { reportId: string };
}

interface DataTableExportProps {
  resource: Resource;
  filterState: FilterStateSchema;
  resourceName: string;
}

export function DataTableExport({
  resource,
  filterState,
  resourceName,
}: DataTableExportProps) {
  const [open, setOpen] = React.useState(false);
  const [format, setFormat] = React.useState<ReportFormat>("CSV");
  const [deliveryMethod, setDeliveryMethod] =
    React.useState<DeliveryMethod>("DOWNLOAD");

  const generateMutation = useMutation({
    mutationFn: generateReport,
    onSuccess: () => {
      toast.success("Export Started", {
        description:
          deliveryMethod === "EMAIL"
            ? "You'll receive an email when your report is ready"
            : "You'll receive a notification when your report is ready",
      });
      setOpen(false);
    },
    onError: (error: Error) => {
      toast.error("Export Failed", {
        description: error.message,
      });
    },
  });

  const handleExport = () => {
    const exportName = `${resourceName} Export - ${new Date().toLocaleDateString()}`;

    generateMutation.mutate({
      resourceType: resource.toLowerCase(),
      name: exportName,
      format,
      deliveryMethod,
      filterState,
    });
  };

  return (
    <>
      <Button variant="outline" onClick={() => setOpen(true)}>
        <Icon icon={faDownload} />
        Export
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Export Data</DialogTitle>
            <DialogDescription>
              Export the current filtered and sorted data to a file
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="format">Format</Label>
              <Select
                value={format}
                onValueChange={(value) => setFormat(value as ReportFormat)}
                disabled={generateMutation.isPending}
              >
                <SelectTrigger id="format">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="CSV">CSV (Comma-separated)</SelectItem>
                  <SelectItem value="EXCEL">
                    <div className="flex items-center gap-2">
                      <Icon icon={faFileExcel} className="text-green-600" />
                      Excel (XLSX)
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="delivery">Delivery Method</Label>
              <Select
                value={deliveryMethod}
                onValueChange={(value) =>
                  setDeliveryMethod(value as DeliveryMethod)
                }
                disabled={generateMutation.isPending}
              >
                <SelectTrigger id="delivery">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="DOWNLOAD">
                    Notification with download link
                  </SelectItem>
                  <SelectItem value="EMAIL">Send via email</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {filterState.filters.length > 0 && (
              <div className="rounded-md bg-muted p-3 text-sm">
                <p className="mb-1 font-medium">Active Filters:</p>
                <ul className="list-inside list-disc space-y-0.5">
                  {filterState.filters.map((filter, index) => (
                    <li key={index} className="text-muted-foreground">
                      {filter.field}: {filter.operator} {String(filter.value)}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {filterState.sort.length > 0 && (
              <div className="rounded-md bg-muted p-3 text-sm">
                <p className="mb-1 font-medium">Active Sorting:</p>
                <ul className="list-inside list-disc space-y-0.5">
                  {filterState.sort.map((sort, index) => (
                    <li key={index} className="text-muted-foreground">
                      {sort.field}: {sort.direction}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setOpen(false)}
              disabled={generateMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              onClick={handleExport}
              disabled={generateMutation.isPending}
            >
              {generateMutation.isPending ? (
                <>Starting export...</>
              ) : (
                <>
                  <Icon icon={faDownload} />
                  Export
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
