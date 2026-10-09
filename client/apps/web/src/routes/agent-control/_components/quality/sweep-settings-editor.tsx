import { TimezoneField } from "@/components/fields/timezone-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchAgentQualityControl,
  updateAgentQualityControl,
  type AgentQualityControl,
} from "@/lib/graphql/agent-quality";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useMemo } from "react";
import { Controller, useForm, useWatch, type Control, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { withPresets } from "../agent-control-options";
import type { EditFields } from "../edit/change-review";
import { EditSheet, type EditSection } from "../edit/edit-sheet";
import { aicFieldTrigger } from "../edit/field-trigger";
import { useEditFlow } from "../edit/use-edit-flow";
import { Ic } from "../kit/ic";
import { Switch } from "../kit/layout";
import {
  formatUsd,
  qualityControlSchema,
  toFormValues,
  toUpdateInput,
  type QualityControlFormValues,
} from "./quality-model";
import {
  CASES_PER_AGENT_PRESETS,
  JUDGE_SHARE_PRESETS,
  MIN_CASES_PRESETS,
  MONTHLY_BUDGET_PRESETS,
  NIGHTLY_BUDGET_PRESETS,
  RERUN_DAYS_PRESETS,
  START_HOUR_PRESETS,
  THRESHOLD_PRESETS,
  hourLabel,
} from "./sweep-settings-model";

type SweepSettingsEditorProps = {
  open: boolean;
  control: AgentQualityControl;
  onClose: () => void;
};

type NumberName = Exclude<
  keyof QualityControlFormValues,
  "enabled" | "judgeEnabled" | "runHour" | "timezone" | "version"
>;

/**
 * The nightly replay of each agent's golden set, in the shared editor: when it starts,
 * how many cases each agent draws, whether a judge grades a sample, what counts as a
 * regression, and what evaluation may spend. Budgets are spend, so saving needs the
 * right to change AI control as well as the golden set.
 */
export function SweepSettingsEditor({ open, control, onClose }: SweepSettingsEditorProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdateSuite } = usePermission(Resource.AgentEvalSuite, Operation.Update);
  const { allowed: canUpdateControl } = usePermission(Resource.AgentControl, Operation.Update);
  const loaded = useMemo(() => toFormValues(control), [control]);
  const form = useForm<QualityControlFormValues>({
    resolver: zodResolver(qualityControlSchema) as Resolver<QualityControlFormValues>,
    defaultValues: loaded,
    values: open ? loaded : undefined,
    resetOptions: { keepDirtyValues: true },
  });
  const values = useWatch({ control: form.control }) as QualityControlFormValues;

  const save = useCallback(
    async (next: QualityControlFormValues) => {
      const saved = await updateAgentQualityControl(toUpdateInput(next));
      queryClient.setQueryData(queries.agentQuality.control().queryKey, saved);
      await queryClient.invalidateQueries({ queryKey: queries.agentQuality.overview().queryKey });
      toast.success(t("Sweep settings saved"));
      return toFormValues(saved);
    },
    [queryClient, t],
  );

  const flow = useEditFlow({
    form,
    onSave: save,
    onClose,
    enabled: open,
    resourceName: t("Sweep settings"),
    versionField: "version",
    invalid:
      canUpdateSuite && canUpdateControl
        ? null
        : t("Changing these needs the right to update AI control and the golden set."),
    loadLatest: async () => {
      return toFormValues(await fetchAgentQualityControl());
    },
  });

  const fields: EditFields = {
    enabled: { label: t("Nightly sweep") },
    runHour: { label: t("Start at"), format: (value) => hourLabel(Number(value)) },
    timezone: { label: t("Timezone") },
    maxCasesPerAgent: { label: t("Cases per agent") },
    forceRerunDays: { label: t("Rerun an unchanged agent after") },
    judgeEnabled: { label: t("Judge model") },
    judgeSamplePercent: { label: t("Share of answers"), format: (value) => `${String(value)}%` },
    regressionThresholdPoints: {
      label: t("Regression threshold"),
      format: (value) => t("{0} pts", String(value)),
    },
    minCases: { label: t("Fewest cases to compare") },
    nightlyBudgetCents: {
      label: t("Per night"),
      format: (value) => formatUsd(String(Number(value) / 100)),
    },
    monthlyBudgetCents: {
      label: t("Per month"),
      format: (value) => formatUsd(String(Number(value) / 100)),
    },
  };

  const sections: EditSection[] = [
    {
      id: "sweep",
      label: t("Nightly sweep"),
      keys: ["enabled", "runHour", "timezone", "maxCasesPerAgent", "forceRerunDays"],
      actions: <HeaderSwitch control={form.control} name="enabled" label={t("Nightly sweep")} />,
      note: t("Replays each agent's cases and scores the answers."),
      content: values.enabled ? (
        <>
          <SegmentedField<QualityControlFormValues, string>
            control={form.control}
            name="runHour"
            label={t("Start at")}
            options={withPresets(START_HOUR_PRESETS, Number(values.runHour)).map((hour) => ({
              value: String(hour),
              label: hourLabel(hour),
            }))}
          />
          <TimezoneField
            control={form.control}
            name="timezone"
            label={t("Timezone")}
            description={t("Leave empty for the organization's timezone.")}
            placeholder={t("The organization's timezone")}
            triggerClassName={aicFieldTrigger}
            isClearable
          />
          <PresetField
            control={form.control}
            name="maxCasesPerAgent"
            label={t("Cases per agent")}
            presets={CASES_PER_AGENT_PRESETS}
          />
          <PresetField
            control={form.control}
            name="forceRerunDays"
            label={t("Rerun an unchanged agent after")}
            hint={t(
              "An agent whose instructions, tools, model and cases have not changed is skipped until its last run is this old.",
            )}
            presets={RERUN_DAYS_PRESETS}
            format={(days) => t("{0, plural, one {# day} other {# days}}", days)}
          />
        </>
      ) : (
        <p className="es-note">{t("A suite can still be run by hand from an agent's sheet.")}</p>
      ),
    },
    {
      id: "judge",
      label: t("Judge model"),
      keys: ["judgeEnabled", "judgeSamplePercent"],
      actions: <HeaderSwitch control={form.control} name="judgeEnabled" label={t("Judge model")} />,
      note: t(
        "A second model grades a sample of answers against their rubric. It never overrides a hard check, and what it costs counts toward the budget.",
      ),
      content: values.judgeEnabled && (
        <PresetField
          control={form.control}
          name="judgeSamplePercent"
          label={t("Share of answers")}
          presets={JUDGE_SHARE_PRESETS}
          format={(share) => (share === 100 ? t("All") : `${share}%`)}
        />
      ),
    },
    {
      id: "threshold",
      label: t("Regression threshold"),
      keys: ["regressionThresholdPoints", "minCases"],
      note: t("A drop this large below the recent median flags the agent and tells its owners."),
      content: (
        <>
          <PresetField
            control={form.control}
            name="regressionThresholdPoints"
            label={t("Regression threshold")}
            presets={THRESHOLD_PRESETS}
            format={(points) => t("{0} pts", points)}
          />
          <PresetField
            control={form.control}
            name="minCases"
            label={t("Fewest cases to compare")}
            hint={t("A run that scored fewer cases is not compared with the median.")}
            presets={MIN_CASES_PRESETS}
          />
        </>
      ),
    },
    {
      id: "budget",
      label: t("Budget"),
      keys: ["nightlyBudgetCents", "monthlyBudgetCents"],
      note: t("Stops a sweep when either is reached."),
      content: (
        <>
          <PresetField
            control={form.control}
            name="nightlyBudgetCents"
            label={t("Per night")}
            presets={NIGHTLY_BUDGET_PRESETS}
            format={(cents) => formatUsd(String(cents / 100))}
          />
          <PresetField
            control={form.control}
            name="monthlyBudgetCents"
            label={t("Per month")}
            presets={MONTHLY_BUDGET_PRESETS}
            format={(cents) => formatUsd(String(cents / 100))}
          />
        </>
      ),
    },
  ];

  return (
    <EditSheet
      open={open}
      form={form}
      flow={flow}
      fields={fields}
      sections={sections}
      icon={
        <span className="src-i">
          <Ic n="gear" s={15} />
        </span>
      }
      title={t("Sweep settings")}
      subtitle={t("The nightly replay of each agent's golden set")}
    />
  );
}

function HeaderSwitch({
  control,
  name,
  label,
}: {
  control: Control<QualityControlFormValues>;
  name: "enabled" | "judgeEnabled";
  label: string;
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => <Switch on={field.value} onChange={field.onChange} label={label} />}
    />
  );
}

function PresetField({
  control,
  name,
  label,
  hint,
  presets,
  format = String,
}: {
  control: Control<QualityControlFormValues>;
  name: NumberName;
  label: string;
  hint?: string;
  presets: readonly number[];
  format?: (value: number) => string;
}) {
  const current = useWatch({ control, name });

  return (
    <SegmentedField<QualityControlFormValues, number>
      control={control}
      name={name}
      label={label}
      description={hint}
      options={withPresets(presets, current).map((value) => ({ value, label: format(value) }))}
    />
  );
}
