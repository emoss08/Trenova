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
import { useToast } from "@/hooks/use-toast";
import { API_URL } from "@/lib/constants";
import type { FilterStateSchema } from "@/lib/schemas/table-configuration-schema";
import type { Resource } from "@/types/audit-entry";
import { faDownload, faFileExcel } from "@fortawesome/pro-solid-svg-icons";
import { useMutation, useQuery } from "@tanstack/react-query";
import React from "react";

type ReportFormat = "CSV" | "EXCEL";
type DeliveryMethod = "DOWNLOAD" | "EMAIL";

interface GenerateReportRequest {
  resourceType: string;
  name: string;
  format: ReportFormat;
  deliveryMethod: DeliveryMethod;
  filterState: FilterStateSchema;
}

interface Report {
  id: string;
  status: "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED";
  filePath: string;
  errorMessage?: string;
}

async function generateReport(
  request: GenerateReportRequest,
): Promise<{ reportId: string }> {
  const response = await fetch(`${API_URL}/reports/generate/`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    credentials: "include",
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || "Failed to generate report");
  }

  return response.json();
}

async function getReportStatus(reportId: string): Promise<Report> {
  const response = await fetch(`${API_URL}/reports/${reportId}/`, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error("Failed to get report status");
  }

  return response.json();
}

function downloadReport(reportId: string, fileName: string) {
  const downloadUrl = `${API_URL}/reports/${reportId}/download/`;
  const link = document.createElement("a");
  link.href = downloadUrl;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
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
  const [reportId, setReportId] = React.useState<string | null>(null);
  const { toast } = useToast();

  const generateMutation = useMutation({
    mutationFn: generateReport,
    onSuccess: (data) => {
      setReportId(data.reportId);
      toast({
        title: "Export Started",
        description: "Your report is being generated...",
      });
    },
    onError: (error: Error) => {
      toast({
        title: "Export Failed",
        description: error.message,
        variant: "destructive",
      });
    },
  });

  const { data: reportStatus, isLoading: isPolling } = useQuery({
    queryKey: ["report-status", reportId],
    queryFn: () => getReportStatus(reportId!),
    enabled: !!reportId,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === "COMPLETED" || status === "FAILED") {
        return false;
      }
      return 2000;
    },
  });

  React.useEffect(() => {
    if (reportStatus?.status === "COMPLETED") {
      toast({
        title: "Export Complete",
        description:
          deliveryMethod === "EMAIL"
            ? "The report has been sent to your email"
            : "Your report is ready for download",
      });

      if (deliveryMethod === "DOWNLOAD") {
        const fileName = `${resourceName}_export_${new Date().toISOString().split("T")[0]}.${format.toLowerCase()}`;
        downloadReport(reportId!, fileName);
      }

      setReportId(null);
      setOpen(false);
    } else if (reportStatus?.status === "FAILED") {
      toast({
        title: "Export Failed",
        description:
          reportStatus.errorMessage || "An error occurred during export",
        variant: "destructive",
      });
      setReportId(null);
    }
  }, [reportStatus, deliveryMethod, format, reportId, resourceName, toast]);

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

  const isProcessing = generateMutation.isPending || isPolling || !!reportId;

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
                disabled={isProcessing}
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
                disabled={isProcessing}
              >
                <SelectTrigger id="delivery">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="DOWNLOAD">
                    Download immediately
                  </SelectItem>
                  <SelectItem value="EMAIL">Send via email</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {filterState.filters.length > 0 && (
              <div className="rounded-md bg-muted p-3 text-sm">
                <p className="font-medium mb-1">Active Filters:</p>
                <ul className="list-disc list-inside space-y-0.5">
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
                <p className="font-medium mb-1">Active Sorting:</p>
                <ul className="list-disc list-inside space-y-0.5">
                  {filterState.sort.map((sort, index) => (
                    <li key={index} className="text-muted-foreground">
                      {sort.field}: {sort.direction}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {isProcessing && (
              <div className="rounded-md bg-blue-50 dark:bg-blue-950 p-3 text-sm">
                <p className="text-blue-900 dark:text-blue-100">
                  {reportStatus?.status === "PROCESSING"
                    ? "Generating your export..."
                    : "Starting export..."}
                </p>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setOpen(false)}
              disabled={isProcessing}
            >
              Cancel
            </Button>
            <Button onClick={handleExport} disabled={isProcessing}>
              {isProcessing ? (
                <>Processing...</>
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
