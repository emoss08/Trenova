import { InputField } from "@/components/fields/input-field";
import { MoneyField } from "@/components/fields/money-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { updateAgentQualityControl, type AgentQualityControl } from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleAlertIcon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import {
  qualityControlSchema,
  runHourOptions,
  toFormValues,
  toUpdateInput,
  type QualityControlFormValues,
} from "./quality-model";

const FORM_ID = "agent-quality-control-form";

/**
 * When the nightly sweep runs and how far it may go: the hour in the
 * organization's timezone, how many cases each agent draws, what evaluation
 * may spend, whether a judge reads a sample, and what counts as a regression.
 * Budgets are spend, so saving them needs the right to change AI Control as
 * well as the golden set.
 */
export function SettingsPanel() {
  const t = useT();
  const control = useQuery({ ...queries.agentQuality.control() });

  return (
    <SectionPanel
      title={t("Settings")}
      help={t(
        "The sweep runs once a night at this hour and skips an agent whose instructions, tools, model and cases have not changed since its last run, until that run is older than the rerun period. Evaluation stops for the night when either budget is spent.",
      )}
    >
      {control.isError ? (
        <div className="p-3">
          <Alert variant="destructive" size="sm">
            <CircleAlertIcon />
            <AlertDescription>{t("The quality settings could not be loaded.")}</AlertDescription>
          </Alert>
        </div>
      ) : control.data ? (
        <SettingsForm control={control.data} />
      ) : (
        <div className="flex flex-col gap-2 p-3" aria-busy>
          <Skeleton className="h-8" />
          <Skeleton className="h-8" />
          <Skeleton className="h-8" />
        </div>
      )}
    </SectionPanel>
  );
}

function SettingsForm({ control }: { control: AgentQualityControl }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdateSuite } = usePermission(Resource.AgentEvalSuite, Operation.Update);
  const { allowed: canUpdateControl } = usePermission(Resource.AgentControl, Operation.Update);
  const canSave = canUpdateSuite && canUpdateControl;

  const defaults = useMemo(() => toFormValues(control), [control]);
  const form = useForm<QualityControlFormValues>({
    resolver: zodResolver(qualityControlSchema) as Resolver<QualityControlFormValues>,
    defaultValues: defaults,
    mode: "onChange",
  });
  const { control: formControl, handleSubmit, reset, formState } = form;
  const judgeEnabled = useWatch({ control: formControl, name: "judgeEnabled" });
  const hours = useMemo(() => runHourOptions(), []);

  useEffect(() => {
    reset(defaults);
  }, [defaults, reset]);

  const save = useApiMutation<
    AgentQualityControl,
    QualityControlFormValues,
    unknown,
    QualityControlFormValues
  >({
    mutationFn: (values) => updateAgentQualityControl(toUpdateInput(values, control.version)),
    onSuccess: async (saved) => {
      queryClient.setQueryData(queries.agentQuality.control().queryKey, saved);
      await queryClient.invalidateQueries({ queryKey: queries.agentQuality.overview().queryKey });
      toast.success(t("Quality settings saved"));
    },
    form,
    resourceName: t("Quality settings"),
  });

  return (
    <form
      id={FORM_ID}
      onSubmit={handleSubmit((values) => save.mutate(values))}
      className="flex flex-col"
    >
      <FormGroup cols={2} className="p-3">
        <FormControl cols="full">
          <SwitchField
            control={formControl}
            name="enabled"
            label={t("Run the nightly sweep")}
            description={t("A suite can still be run by hand from an agent's page.")}
            outlined
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={formControl}
            name="runHour"
            label={t("Hour it starts")}
            options={hours}
            isReadOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={formControl}
            name="timezone"
            label={t("Timezone")}
            description={t("Leave empty for the organization's timezone.")}
            placeholder={t("The organization's timezone")}
            maxLength={100}
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={formControl}
            name="maxCasesPerAgent"
            label={t("Most cases per agent")}
            min={1}
            max={500}
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={formControl}
            name="forceRerunDays"
            label={t("Days before an unchanged agent runs again")}
            min={1}
            max={90}
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <MoneyField
            control={formControl}
            name="nightlyBudgetCents"
            label={t("Nightly budget (USD)")}
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <MoneyField
            control={formControl}
            name="monthlyBudgetCents"
            label={t("Monthly budget (USD)")}
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={formControl}
            name="regressionThresholdPoints"
            label={t("Regression threshold (points)")}
            description={t("Points below the median of the recent runs.")}
            min={1}
            max={100}
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={formControl}
            name="minCases"
            label={t("Fewest cases to compare scores")}
            min={1}
            max={500}
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={formControl}
            name="judgeEnabled"
            label={t("Have a judge read a sample")}
            description={t(
              "A model reads some answers against their rubric. It never overrides a hard check, and what it costs counts toward the budget.",
            )}
            outlined
            readOnly={!canSave}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={formControl}
            name="judgeSamplePercent"
            label={t("Share of cases the judge reads")}
            suffix="%"
            min={0}
            max={100}
            readOnly={!canSave || !judgeEnabled}
          />
        </FormControl>
      </FormGroup>
      <div className="border-border flex items-center justify-between gap-2 border-t px-3 py-2">
        <span className="text-muted-foreground text-xs">
          {canSave
            ? t("Changes apply from the next sweep.")
            : t("Changing these needs the right to update AI Control and the golden set.")}
        </span>
        <Button
          type="submit"
          form={FORM_ID}
          size="sm"
          disabled={!canSave || !formState.isDirty || save.isPending}
        >
          {t("Save settings")}
        </Button>
      </div>
    </form>
  );
}
