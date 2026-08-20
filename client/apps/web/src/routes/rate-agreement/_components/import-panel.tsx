import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  canCommit,
  changeKindLabel,
  changeTone,
  commitWarnings,
  describeFieldChange,
  importHeadline,
  meaningfulChanges,
  type ChangeTone,
} from "@trenova/shared/lib/rate-import";
import type { RateImportBatch } from "@trenova/shared/types/rate";
import { CircleAlertIcon, LoaderCircleIcon, UploadIcon } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";
import { useLaneKeyLabels } from "./use-lane-scope-labels";

const TONE_CLASS: Record<ChangeTone, string> = {
  danger: "text-destructive",
  warning: "text-warning",
  neutral: "text-foreground",
  quiet: "text-muted-foreground",
};

type ImportPanelProps = {
  /** Absent while the agreement is being created — there is nothing to import into. */
  readonly rateAgreementId?: string;
};

/**
 * Upload a rate sheet, read what it would do, then apply it.
 *
 * The dry run between the upload and the commit is the whole point. An import
 * that went straight from a file to a live contract is what bulk ingest is most
 * often blamed for: a tariff nobody read replacing one somebody negotiated.
 */
export function ImportPanel({ rateAgreementId }: ImportPanelProps) {
  const queryClient = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);

  const [staged, setStaged] = useState<RateImportBatch | undefined>();
  const [effectiveFrom, setEffectiveFrom] = useState(() => todayAsDateInput());

  const { data: history } = useQuery({
    queryKey: ["rate-imports", rateAgreementId],
    queryFn: () => apiService.rateImportService.listForAgreement(rateAgreementId as string),
    enabled: Boolean(rateAgreementId),
  });

  const batch = staged ?? history?.[0];

  const { mutate: upload, isPending: isUploading } = useMutation({
    mutationFn: (file: File) =>
      apiService.rateImportService.upload({
        rateAgreementId: rateAgreementId as string,
        effectiveFrom: Math.floor(new Date(effectiveFrom).getTime() / 1000),
        file,
      }),
    onSuccess: async (created) => {
      setStaged(created);
      toast.success("Sheet read — review what it would change");
      await queryClient.invalidateQueries({ queryKey: ["rate-imports", rateAgreementId] });
    },
    onError: () => toast.error("That file could not be read"),
  });

  const { mutate: commit, isPending: isCommitting } = useMutation({
    mutationFn: () => apiService.rateImportService.commit(batch?.id as string),
    onSuccess: async (applied) => {
      setStaged(applied);
      toast.success("Rates applied to the agreement");
      await queryClient.invalidateQueries({ queryKey: ["rate-agreement-list"] });
      await queryClient.invalidateQueries({ queryKey: ["rate-imports", rateAgreementId] });
    },
    onError: () => toast.error("The rates could not be applied"),
  });

  const { mutate: discard } = useMutation({
    mutationFn: () => apiService.rateImportService.discard(batch?.id as string),
    onSuccess: async (discarded) => {
      setStaged(discarded);
      await queryClient.invalidateQueries({ queryKey: ["rate-imports", rateAgreementId] });
    },
    onError: () => toast.error("The import could not be discarded"),
  });

  const chooseFile = useCallback(() => fileInput.current?.click(), []);

  const changes = meaningfulChanges(batch?.changes);
  const displayLaneKey = useLaneKeyLabels(changes.map((change) => change.laneKey));

  const onFileChosen = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (file) upload(file);

      // Clearing lets the same file be re-uploaded after it has been fixed,
      // which is the ordinary next step after reading the errors.
      event.target.value = "";
    },
    [upload],
  );

  if (!rateAgreementId) {
    return (
      <p className="text-muted-foreground text-sm">
        Save the agreement first. A rate sheet is imported into a contract, and there has to be a
        contract to import into.
      </p>
    );
  }

  const warnings = commitWarnings(batch);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs">These rates take effect</span>
          <Input
            type="date"
            className="h-8 w-44"
            value={effectiveFrom}
            onChange={(event) => setEffectiveFrom(event.target.value)}
          />
        </div>
        <Button type="button" size="sm" disabled={isUploading} onClick={chooseFile}>
          {isUploading ? (
            <LoaderCircleIcon className="mr-1 size-3.5 animate-spin" />
          ) : (
            <UploadIcon className="mr-1 size-3.5" />
          )}
          Upload a rate sheet
        </Button>
        <input
          ref={fileInput}
          type="file"
          accept=".csv,.xlsx,.xlsm"
          className="hidden"
          onChange={onFileChosen}
        />
      </div>

      <p className="text-muted-foreground text-xs">
        CSV or XLSX. Columns are matched by name — nothing is applied until you have read what it
        would change.
      </p>

      {batch && (
        <div className="rounded-md border p-3">
          <div className="mb-2 flex items-center gap-2">
            <Badge variant={batch.status === "Failed" ? "warning" : "secondary"}>
              {batch.status}
            </Badge>
            <span className="font-mono text-xs">{batch.fileName}</span>
            <span className="text-muted-foreground text-xs">{batch.rowCount} rows</span>
          </div>

          <p className="text-sm">{importHeadline(batch)}</p>

          {canCommit(batch) && (
            <div className="mt-3 flex items-center gap-2">
              <Button type="button" size="sm" disabled={isCommitting} onClick={() => commit()}>
                {isCommitting && <LoaderCircleIcon className="mr-1 size-3.5 animate-spin" />}
                Apply these rates
              </Button>
              <Button type="button" size="sm" variant="outline" onClick={() => discard()}>
                Discard
              </Button>
            </div>
          )}
        </div>
      )}

      {warnings.map((warning) => (
        <Alert key={warning}>
          <CircleAlertIcon className="size-4" />
          <AlertDescription>{warning}</AlertDescription>
        </Alert>
      ))}

      {changes.length > 0 && (
        <div className="overflow-x-auto rounded-md border">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="bg-muted/50 text-muted-foreground text-xs">
                <th className="border-b px-3 py-2 text-left font-medium">Lane</th>
                <th className="border-b px-3 py-2 text-left font-medium">What happens</th>
                <th className="border-b px-3 py-2 text-left font-medium">Changes</th>
              </tr>
            </thead>
            <tbody>
              {changes.map((change) => (
                <tr key={`${change.kind}-${change.laneKey}`}>
                  <td className="border-b px-3 py-2">
                    <span className="font-medium">
                      {change.label || displayLaneKey(change.laneKey)}
                    </span>
                    <p className="text-muted-foreground font-mono text-xs">
                      {displayLaneKey(change.laneKey)}
                    </p>
                  </td>
                  <td
                    className={`border-b px-3 py-2 text-xs ${TONE_CLASS[changeTone(change.kind)]}`}
                  >
                    {changeKindLabel(change.kind)}
                  </td>
                  <td className="text-muted-foreground border-b px-3 py-2 text-xs">
                    {(change.fields ?? []).map((field) => (
                      <p key={field.field}>{describeFieldChange(field)}</p>
                    ))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

/** Today, in the format a date input takes. */
function todayAsDateInput(): string {
  return new Date().toISOString().slice(0, 10);
}
