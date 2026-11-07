import { DatePickerField } from "@/components/fields/date-picker-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { AUDIT_EXPORT_LIST_KEY, requestAuditExport } from "@/lib/graphql/audit-log-admin";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { z } from "zod";

const SECONDS_PER_DAY = 86_400;
const MAX_RANGE_DAYS = 400;

function exportSchema(t: TranslateFn) {
  return z
    .object({
      format: z.enum(["CSV", "JSONL"]),
      rangeFrom: z.number({ error: t("Choose the first day") }),
      rangeTo: z.number({ error: t("Choose the last day") }),
      includeSecurity: z.boolean(),
    })
    .refine((v) => v.rangeTo >= v.rangeFrom, {
      path: ["rangeTo"],
      message: t("The last day must be on or after the first"),
    })
    .refine((v) => v.rangeTo - v.rangeFrom <= MAX_RANGE_DAYS * SECONDS_PER_DAY, {
      path: ["rangeTo"],
      message: t("An export covers at most 400 days"),
    });
}

type ExportFormValues = z.infer<ReturnType<typeof exportSchema>>;

function defaults(): ExportFormValues {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const end = Math.floor(today.getTime() / 1000);
  return {
    format: "CSV",
    rangeFrom: end - 30 * SECONDS_PER_DAY,
    rangeTo: end,
    includeSecurity: false,
  };
}

export function RequestAuditExportDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canIncludeSecurity } = usePermission(Resource.SecurityAuditLog, Operation.Read);
  const form = useForm<ExportFormValues>({
    resolver: zodResolver(exportSchema(t)) as Resolver<ExportFormValues>,
    defaultValues: defaults(),
  });

  useEffect(() => {
    if (open) {
      form.reset(defaults());
    }
  }, [open, form]);

  const request = useApiMutation({
    form,
    scope: { id: "audit-export-request" },
    resourceName: "Audit export",
    mutationFn: (values: ExportFormValues) =>
      requestAuditExport({
        format: values.format,
        rangeFrom: values.rangeFrom,
        rangeTo: values.rangeTo + SECONDS_PER_DAY,
        includeSecurity: canIncludeSecurity && values.includeSecurity,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [AUDIT_EXPORT_LIST_KEY] });
      onOpenChange(false);
    },
  });

  const submit = form.handleSubmit((values) => request.mutateAsync(values).catch(() => undefined));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Export the audit log")}</DialogTitle>
          <DialogDescription>
            {t(
              "The file is written in the background and only you can download it. It holds the entries you can read, with protected fields withheld as they are on screen.",
            )}
          </DialogDescription>
        </DialogHeader>
        <form
          id="audit-export-form"
          onSubmit={(event) => void submit(event)}
          className="grid gap-3"
        >
          <SelectField
            control={form.control}
            name="format"
            label={t("Format")}
            options={[
              { value: "CSV", label: t("CSV") },
              { value: "JSONL", label: t("JSON Lines, with every recorded state") },
            ]}
          />
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <DatePickerField control={form.control} name="rangeFrom" label={t("First day")} />
            <DatePickerField control={form.control} name="rangeTo" label={t("Last day")} />
          </div>
          {canIncludeSecurity ? (
            <SwitchField
              control={form.control}
              name="includeSecurity"
              label={t("Include security entries")}
              description={t("Access changes, refused requests and other security events")}
            />
          ) : null}
        </form>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form="audit-export-form" isLoading={request.isPending}>
            {t("Export")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
