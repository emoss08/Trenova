import { FormSaveDock } from "@/components/form-save-dock";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { usePermission } from "@/hooks/use-permission";
import { dataRetention } from "@/lib/queries/data-retention";
import { updateDataRetention } from "@/services/data-retention";
import type { DataRetention } from "@/types/data-retention";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Card } from "@trenova/shared/components/ui/card";
import { Form } from "@trenova/shared/components/ui/form";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { FormProvider, useForm } from "react-hook-form";
import {
  AGENT_EVAL_CASE_RETENTION_LIMIT_DAYS,
  AI_AUDIT_RETENTION_MIN_DAYS,
  AI_CORRECTION_RETENTION_MIN_DAYS,
  AI_FEEDBACK_RETENTION_MIN_DAYS,
  dataRetentionFormSchema,
  dataRetentionFormValues,
  EDI_RETENTION_LIMIT_DAYS,
  type DataRetentionFormValues,
} from "../data-retention-form";
import { RetentionPolicyRow } from "./retention-policy-row";

const COLUMN_COUNT = 4;

/**
 * The whole retention policy in one table: every kind of data the nightly
 * purges age out, how long each is kept, the date that reaches back to, and
 * what happens to what is older. Someone without update access reads the same
 * table with nothing to change.
 */
export default function DataRetentionForm({
  now = Math.floor(Date.now() / 1000),
}: {
  now?: number;
}) {
  const t = useT();
  const { allowed: canUpdate } = usePermission(Resource.Organization, Operation.Update);
  const { data } = useSuspenseQuery(dataRetention.get());

  const form = useForm<DataRetentionFormValues>({
    resolver: zodResolver(dataRetentionFormSchema),
    defaultValues: dataRetentionFormValues(data),
    mode: "onChange",
  });
  const { handleSubmit, reset } = form;

  const { mutateAsync } = useOptimisticMutation<
    DataRetention,
    DataRetentionFormValues,
    DataRetention,
    DataRetentionFormValues
  >({
    queryKey: dataRetention.get().queryKey,
    mutationFn: updateDataRetention,
    resourceName: "Data Retention",
    form,
    onSuccess: (saved) => reset(dataRetentionFormValues(saved)),
  });

  const readOnly = !canUpdate;
  const deleted = t("Deleted");
  const blanked = t("Raw content blanked");

  return (
    <FormProvider {...form}>
      <Form
        onSubmit={handleSubmit(async (values) => {
          await mutateAsync(values);
        })}
      >
        <div className="flex max-w-5xl flex-col gap-4 pb-14">
          {readOnly && (
            <Alert variant="info" size="sm">
              <AlertDescription>
                {t("You can view these settings but not change them.")}
              </AlertDescription>
            </Alert>
          )}

          <Card className="gap-0 py-0">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="pl-4">{t("Data")}</TableHead>
                  <TableHead className="w-48">{t("Kept for")}</TableHead>
                  <TableHead className="w-36">{t("Removes before")}</TableHead>
                  <TableHead className="w-40 pr-4">{t("Then")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <GroupHeading title={t("Audit")} />
                <RetentionPolicyRow
                  name="auditRetentionPeriod"
                  label={t("Audit entries")}
                  description={t(
                    "The record of who changed what, removed by the nightly audit purge.",
                  )}
                  minDays={1}
                  outcome={deleted}
                  now={now}
                  disabled={readOnly}
                />

                <GroupHeading title={t("EDI")} />
                <RetentionPolicyRow
                  name="ediInboundFileRetentionPeriod"
                  label={t("Inbound file contents")}
                  description={t("Raw inbound EDI files. The record of each file stays.")}
                  outcome={blanked}
                  noLimit={{ fallbackDays: EDI_RETENTION_LIMIT_DAYS }}
                  now={now}
                  disabled={readOnly}
                />
                <RetentionPolicyRow
                  name="ediMessageRetentionPeriod"
                  label={t("Message payloads")}
                  description={t(
                    "Raw X12 and payload snapshots of delivered and inbound messages. A blanked message can no longer be replayed.",
                  )}
                  outcome={blanked}
                  noLimit={{ fallbackDays: EDI_RETENTION_LIMIT_DAYS }}
                  now={now}
                  disabled={readOnly}
                />

                <GroupHeading title={t("AI")} />
                <RetentionPolicyRow
                  name="aiAuditRetentionPeriod"
                  label={t("AI audit trail")}
                  description={t(
                    "Signed events recording each AI decision, cut only where the chain was sealed. Seven years is what most audits of AI decisions ask for.",
                  )}
                  minDays={AI_AUDIT_RETENTION_MIN_DAYS}
                  outcome={deleted}
                  now={now}
                  disabled={readOnly}
                />
                <RetentionPolicyRow
                  name="aiFeedbackRetentionPeriod"
                  label={t("AI feedback")}
                  description={t(
                    "Ratings of AI output, with the question and answer each person saw.",
                  )}
                  minDays={AI_FEEDBACK_RETENTION_MIN_DAYS}
                  outcome={deleted}
                  now={now}
                  disabled={readOnly}
                />
                <RetentionPolicyRow
                  name="aiCorrectionRetentionPeriod"
                  label={t("AI corrections")}
                  description={t(
                    "What was read from a document beside what a person confirmed, used to measure extraction accuracy.",
                  )}
                  minDays={AI_CORRECTION_RETENTION_MIN_DAYS}
                  outcome={deleted}
                  now={now}
                  disabled={readOnly}
                />
                <RetentionPolicyRow
                  name="agentEvalCaseRetentionPeriod"
                  label={t("Agent evaluation cases")}
                  description={t(
                    "Captured cases and their replays, which keep a redacted copy of what the agent was given. Kept forever, they last until they expire or their conversation is deleted.",
                  )}
                  outcome={t("Purged with replays")}
                  noLimit={{ fallbackDays: AGENT_EVAL_CASE_RETENTION_LIMIT_DAYS }}
                  now={now}
                  disabled={readOnly}
                />
              </TableBody>
            </Table>
          </Card>

          {canUpdate && <FormSaveDock saveButtonContent={t("Save changes")} />}
        </div>
      </Form>
    </FormProvider>
  );
}

function GroupHeading({ title }: { title: string }) {
  return (
    <TableRow className="bg-sunken hover:bg-sunken">
      <TableCell
        colSpan={COLUMN_COUNT}
        className="text-foreground-muted py-1.5 pl-4 text-xs font-medium"
      >
        {title}
      </TableCell>
    </TableRow>
  );
}
