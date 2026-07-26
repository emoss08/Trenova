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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useRunReport } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { REPORT_FORMAT_CHOICES, type ReportParameterDef } from "@/types/report";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import {
  ParameterField,
  defaultParamValues,
  missingRequiredParams,
} from "./report-parameter-field";
import { SavedViewField } from "./saved-view-field";

export type RunReportTarget =
  | { definitionId: string; cannedKey?: never }
  | { cannedKey: string; definitionId?: never };

type RunReportDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  target: RunReportTarget | null;
  reportName: string;
  defaultFormat: string;
  parameters: ReportParameterDef[];
  /** Seeds the form with values the caller already collected (e.g. Explore). */
  initialParams?: Record<string, unknown>;
};

export function RunReportDialog({
  open,
  onOpenChange,
  target,
  reportName,
  defaultFormat,
  parameters,
  initialParams,
}: RunReportDialogProps) {
  const navigate = useNavigate();
  const runReport = useRunReport();
  const [format, setFormat] = useState(defaultFormat || "csv");
  const [paramValues, setParamValues] = useState<Record<string, unknown>>(() => ({
    ...defaultParamValues(parameters),
    ...initialParams,
  }));
  const [viewId, setViewId] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setFormat(defaultFormat || "csv");
      setParamValues({ ...defaultParamValues(parameters), ...initialParams });
      setViewId(null);
    }
  }, [open, defaultFormat, parameters, initialParams]);

  // Applying a view fills the form so its values stay visible and editable;
  // editing them afterwards is what detaches the run from the view.
  const handleSelectView = useCallback(
    (view: { id: string; params: Record<string, unknown>; format: string } | null) => {
      setViewId(view?.id ?? null);
      if (!view) return;
      setParamValues({ ...defaultParamValues(parameters), ...view.params });
      if (view.format) setFormat(view.format);
    },
    [parameters],
  );

  const handleParamChange = useCallback((name: string, value: unknown) => {
    setParamValues((prev) => ({ ...prev, [name]: value }));
    // A hand-edited value is no longer the saved view, and claiming otherwise
    // would stamp usage on a view this run does not represent.
    setViewId(null);
  }, []);

  const missingRequired = useMemo(
    () => missingRequiredParams(parameters, paramValues),
    [parameters, paramValues],
  );

  const handleSubmit = useCallback(() => {
    if (!target || missingRequired.length > 0) return;

    runReport.mutate(
      {
        definitionId: target.definitionId,
        cannedKey: target.cannedKey,
        format,
        params: Object.keys(paramValues).length > 0 ? paramValues : undefined,
        viewId: viewId ?? undefined,
      },
      {
        onSuccess: () => {
          onOpenChange(false);
          toast.success(`"${reportName}" queued for generation`, {
            action: {
              label: "View runs",
              onClick: () => navigate("/reports/runs"),
            },
          });
        },
        onError: (error) => {
          toast.error(graphQLErrorMessage(error, "Failed to queue the report run"));
        },
      },
    );
  }, [
    target,
    missingRequired,
    runReport,
    format,
    paramValues,
    viewId,
    onOpenChange,
    reportName,
    navigate,
  ]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xs">
        <DialogHeader>
          <DialogTitle>Run {reportName}</DialogTitle>
          <DialogDescription>
            The report is generated in the background — you&apos;ll be notified when it&apos;s ready
            to download.
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          {/* Views answer a report's parameters, so they are only meaningful on
              a saved report that declares some. */}
          {target?.definitionId && parameters.length > 0 && (
            <SavedViewField
              definitionId={target.definitionId}
              params={paramValues}
              format={format}
              selectedViewId={viewId}
              onSelect={handleSelectView}
            />
          )}
          {parameters.map((param) => (
            <ParameterField
              key={param.name}
              param={param}
              value={paramValues[param.name]}
              onChange={(value) => handleParamChange(param.name, value)}
            />
          ))}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="report-run-format">Format</Label>
            <Select
              value={format}
              onValueChange={(value) => {
                if (value) setFormat(value);
              }}
              items={REPORT_FORMAT_CHOICES}
            >
              <SelectTrigger className="w-full" id="report-run-format">
                <SelectValue placeholder="Select format" />
              </SelectTrigger>
              <SelectContent>
                {REPORT_FORMAT_CHOICES.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={runReport.isPending || missingRequired.length > 0}
          >
            {runReport.isPending ? "Queuing..." : "Run Report"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
