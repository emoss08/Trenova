import { CheckboxField } from "@/components/fields/checkbox-field";
import { DateField } from "@/components/fields/date-field/date-field";
import { FieldWrapper } from "@/components/fields/field-components";
import { downloadAIAuditExport } from "@/lib/ai-audit-exports";
import {
  AI_AUDIT_EXPORT_LIST_KEY,
  requestAIAuditExport,
  type AIAuditExport,
} from "@/lib/graphql/ai-audit";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { CircleAlertIcon, CircleCheckIcon, HourglassIcon } from "lucide-react";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import {
  AUDIT_MAX_EXPORT_FILTERS,
  auditExportDefaults,
  auditExportFilterCount,
  auditExportFormSchema,
  buildAuditExportRequest,
  dayEndOf,
  dayStartOf,
  type AuditExportFormValues,
  type AuditTableState,
  type AuditTrailScope,
} from "./audit-model";

type ExportTrailDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  scope: AuditTrailScope;
  table: AuditTableState;
  onOpenExports: () => void;
};

type ExportOutcome =
  | { kind: "ready"; export: AIAuditExport; downloadError: string | null }
  | { kind: "running"; export: AIAuditExport }
  | { kind: "failed"; message: string };

/**
 * Writes the trail to a signed CSV or JSON file on the server. A small export
 * is written while the person waits and downloads at once; a large one is
 * written in the background and its requester is told when it is ready.
 */
export function ExportTrailDialog({
  open,
  onOpenChange,
  scope,
  table,
  onOpenExports,
}: ExportTrailDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        {open ? (
          <ExportTrailForm
            scope={scope}
            table={table}
            onClose={() => onOpenChange(false)}
            onOpenExports={() => {
              onOpenChange(false);
              onOpenExports();
            }}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

// Mounted only while the dialog is open, so each opening starts from the
// trail's current range rather than from what was chosen last time.
function ExportTrailForm({
  scope,
  table,
  onClose,
  onOpenExports,
}: {
  scope: AuditTrailScope;
  table: AuditTableState;
  onClose: () => void;
  onOpenExports: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [defaults] = useState(() => auditExportDefaults(scope, Math.floor(Date.now() / 1000)));
  const [outcome, setOutcome] = useState<ExportOutcome | null>(null);
  const form = useForm<AuditExportFormValues>({
    resolver: zodResolver(auditExportFormSchema),
    defaultValues: defaults,
  });
  const { control, handleSubmit, setError } = form;

  const redownload = useMutation({
    mutationFn: downloadAIAuditExport,
    onSuccess: () =>
      setOutcome((current) =>
        current?.kind === "ready" ? { ...current, downloadError: null } : current,
      ),
    onError: (error) =>
      setOutcome((current) =>
        current?.kind === "ready"
          ? {
              ...current,
              downloadError: graphQLErrorMessage(error, t("The file could not be downloaded.")),
            }
          : current,
      ),
  });

  const request = useMutation({
    mutationFn: requestAIAuditExport,
    onSuccess: async (created) => {
      void queryClient.invalidateQueries({ queryKey: [AI_AUDIT_EXPORT_LIST_KEY] });
      if (created.status === "Succeeded") {
        try {
          await downloadAIAuditExport(created.id);
          setOutcome({ kind: "ready", export: created, downloadError: null });
        } catch (error) {
          setOutcome({
            kind: "ready",
            export: created,
            downloadError: graphQLErrorMessage(error, t("The file could not be downloaded.")),
          });
        }
        return;
      }
      if (created.status === "Failed") {
        setOutcome({
          kind: "failed",
          message: created.errorMessage || t("The export could not be written."),
        });
        return;
      }
      setOutcome({ kind: "running", export: created });
    },
  });

  const submit = (values: AuditExportFormValues) => {
    const input = buildAuditExportRequest({
      format: values.format,
      from: dayStartOf(values.from),
      to: dayEndOf(values.to),
      useCurrentFilters: values.useCurrentFilters,
      scope,
      table,
    });
    if (auditExportFilterCount(input) > AUDIT_MAX_EXPORT_FILTERS) {
      setError("useCurrentFilters", {
        message: t(
          "An export takes at most {0} filters. Remove some, or export without them.",
          AUDIT_MAX_EXPORT_FILTERS,
        ),
      });
      return;
    }
    request.mutate(input);
  };

  if (outcome !== null) {
    return (
      <>
        <DialogHeader>
          <DialogTitle>{t("Export trail")}</DialogTitle>
        </DialogHeader>
        <ExportOutcomeNotice outcome={outcome} />
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Close")}
          </Button>
          {outcome.kind === "ready" ? (
            <Button
              type="button"
              onClick={() => redownload.mutate(outcome.export.id)}
              isLoading={redownload.isPending}
            >
              {outcome.downloadError ? t("Try the download again") : t("Download again")}
            </Button>
          ) : (
            <Button type="button" onClick={onOpenExports}>
              {t("Open exports")}
            </Button>
          )}
        </DialogFooter>
      </>
    );
  }

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        void handleSubmit(submit)(event);
      }}
    >
      <DialogHeader>
        <DialogTitle>{t("Export trail")}</DialogTitle>
        <DialogDescription>
          {t(
            "The file carries each row's place in the chain and its hash, and the export records the file's SHA-256, so it can be checked later. Requesting it is written to the audit log.",
          )}
        </DialogDescription>
      </DialogHeader>

      <Controller
        control={control}
        name="format"
        render={({ field }) => (
          <FieldWrapper label={t("Format")}>
            <SegmentedControl<AuditExportFormValues["format"]>
              aria-label={t("Format")}
              items={[
                { value: "CSV", label: t("CSV"), caption: t("One row per event") },
                { value: "JSON", label: t("JSON"), caption: t("With the chain's details") },
              ]}
              value={field.value}
              onValueChange={field.onChange}
              fullWidth
            />
          </FieldWrapper>
        )}
      />

      <div className="grid grid-cols-2 gap-3">
        <DateField control={control} name="from" label={t("From")} rules={{ required: true }} />
        <DateField control={control} name="to" label={t("To")} rules={{ required: true }} />
      </div>

      <CheckboxField
        control={control}
        name="useCurrentFilters"
        label={t("Use current filters")}
        description={t(
          "Carry the agent, person, evaluations and table filters into the file. Without them, the file holds every row in the range, which is the only file whose chain an auditor can check end to end.",
        )}
        outlined
      />
      {form.formState.errors.useCurrentFilters?.message ? (
        <p className="text-danger-foreground text-xs">
          {form.formState.errors.useCurrentFilters.message}
        </p>
      ) : null}

      {request.isError ? (
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {graphQLErrorMessage(request.error, t("The export could not be requested."))}
          </AlertDescription>
        </Alert>
      ) : null}

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={request.isPending}>
          {t("Cancel")}
        </Button>
        <Button type="submit" isLoading={request.isPending} loadingText={t("Exporting…")}>
          {t("Export")}
        </Button>
      </DialogFooter>
    </form>
  );
}

function ExportOutcomeNotice({ outcome }: { outcome: ExportOutcome }) {
  const t = useT();

  if (outcome.kind === "failed") {
    return (
      <Alert variant="destructive" size="sm">
        <CircleAlertIcon />
        <AlertTitle>{t("The export failed")}</AlertTitle>
        <AlertDescription>{outcome.message}</AlertDescription>
      </Alert>
    );
  }

  if (outcome.kind === "running") {
    return (
      <Alert variant="info" size="sm">
        <HourglassIcon />
        <AlertTitle>{t("Running — you'll be notified")}</AlertTitle>
        <AlertDescription>
          {t(
            "This export is too large to write while you wait. It is being written in the background; you'll be notified when the file is ready, and it will be under Exports.",
          )}
        </AlertDescription>
      </Alert>
    );
  }

  const file = outcome.export;
  return (
    <div className="flex flex-col gap-2">
      <Alert variant="success" size="sm">
        <CircleCheckIcon />
        <AlertTitle>{t("The file is ready")}</AlertTitle>
        <AlertDescription>
          {t(
            "{0} rows, {1}. {2}",
            file.rowCount.toLocaleString(),
            formatFileSize(file.byteSize),
            file.chainComplete
              ? t("Every row in the range is in the file, so its chain can be checked end to end.")
              : t("Filters left rows out, so the file's chain cannot be checked end to end."),
          )}
        </AlertDescription>
      </Alert>
      {file.sha256 ? (
        <p className="text-foreground-muted text-xs">
          {t("SHA-256")}: <span className="font-mono break-all">{file.sha256}</span>
        </p>
      ) : null}
      {outcome.downloadError ? (
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>{outcome.downloadError}</AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}
