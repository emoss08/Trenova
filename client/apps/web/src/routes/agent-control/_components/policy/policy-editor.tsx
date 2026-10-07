import { describeToolCall } from "@/components/assistant/tool-presentation";
import {
  AGENT_CONTROL_QUERY_KEY,
  fetchAgentControl,
  updateAgentControl,
  type AgentControl,
} from "@/lib/graphql/agent-control";
import type { ToolPromotion } from "@/lib/graphql/ai-control";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { Controller, useForm, useWatch, type Control } from "react-hook-form";
import { toast } from "sonner";
import { personAllowanceOptions, promotionThresholdOptions } from "../agent-control-options";
import type { EditFields } from "../edit/change-review";
import { EditSheet, type EditSection } from "../edit/edit-sheet";
import { Callout, F } from "../edit/fields";
import { useEditFlow } from "../edit/use-edit-flow";
import { Ic } from "../kit/ic";
import { Seg, Switch } from "../kit/layout";
import { TrainingExportHistory } from "../training-export-history";
import {
  policyFormSchema,
  promotesOnSave,
  toControlInput,
  toPolicyForm,
  type PolicyFormValues,
} from "./policy-form";

type PolicyEditorProps = {
  open: boolean;
  control: AgentControl;
  onClose: () => void;
};

/**
 * The organization-wide settings that apply to every agent and override their own, in
 * the shared editor. Turning earned autonomy on says, before saving, which tools move up.
 */
export function PolicyEditor({ open, control, onClose }: PolicyEditorProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const loaded = useMemo(() => toPolicyForm(control), [control]);
  const form = useForm<PolicyFormValues>({
    resolver: zodResolver(policyFormSchema),
    defaultValues: loaded,
    values: open ? loaded : undefined,
    resetOptions: { keepDirtyValues: true },
  });

  const values = useWatch({ control: form.control }) as PolicyFormValues;
  const promotes = promotesOnSave(values, form.formState.defaultValues as PolicyFormValues);
  const preview = useQuery({
    ...queries.aiControl.promotionPreview(values.promotionThreshold),
    enabled: open && promotes,
    staleTime: 30_000,
  });

  const save = useCallback(
    async (next: PolicyFormValues) => {
      const before = form.formState.defaultValues as PolicyFormValues;
      const moved = promotesOnSave(next, before) ? (preview.data?.length ?? 0) : 0;
      const saved = await updateAgentControl(toControlInput(next, before, control.shadowMode));
      queryClient.setQueryData(AGENT_CONTROL_QUERY_KEY, saved);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
        queryClient.invalidateQueries({ queryKey: queries.assistant._def }),
      ]);
      toast.success(
        moved > 0
          ? moved === 1
            ? t("Saved · 1 tool moved up a tier")
            : t("Saved · {0} tools moved up a tier", moved)
          : t("Saved"),
      );
      return toPolicyForm(saved);
    },
    [control.shadowMode, form.formState.defaultValues, preview.data?.length, queryClient, t],
  );

  const flow = useEditFlow({
    form,
    onSave: save,
    onClose,
    enabled: open,
    resourceName: t("Organization-wide settings"),
    loadLatest: async () => toPolicyForm(await fetchAgentControl()),
  });

  const fields: EditFields = {
    earnedAutonomy: { label: t("Earned autonomy") },
    promotionThreshold: { label: t("Clean approvals") },
    learnsFromWork: { label: t("Learning") },
    personMonthlyMessages: {
      label: t("Allowance"),
      format: (value) =>
        typeof value === "number" && value > 0 ? value.toLocaleString() : t("Unlimited"),
    },
    aiTrainingConsent: { label: t("Share corrections") },
  };

  const sections: EditSection[] = [
    {
      id: "earned",
      label: t("Earned autonomy"),
      keys: ["earnedAutonomy", "promotionThreshold"],
      actions: (
        <HeaderSwitch control={form.control} name="earnedAutonomy" label={t("Earned autonomy")} />
      ),
      content: (
        <>
          <p className="es-note">
            {t(
              "A tool moves up a tier after a run of clean approvals — never past the agent's ceiling. When a tool's proposals are approved unchanged this many times in a row, it moves up one tier on that agent. A rejection or a failed run takes an earned tier back. Each change is audited and announced.",
            )}
          </p>
          {values.earnedAutonomy && (
            <F label={t("Clean approvals in a row")}>
              <Controller
                control={form.control}
                name="promotionThreshold"
                render={({ field }) => (
                  <Seg
                    v={field.value}
                    label={t("Clean approvals in a row")}
                    opts={promotionThresholdOptions(field.value).map(
                      (option) => [option.value, option.label] as const,
                    )}
                    onChange={field.onChange}
                  />
                )}
              />
            </F>
          )}
          {promotes && <PromotionPreview loading={preview.isLoading} promotions={preview.data} />}
        </>
      ),
    },
    {
      id: "learn",
      label: t("Learn from their work"),
      keys: ["learnsFromWork"],
      actions: (
        <HeaderSwitch
          control={form.control}
          name="learnsFromWork"
          label={t("Learn from their work")}
        />
      ),
      content: (
        <>
          <p className="es-note">
            {t(
              "When a conversation settles, the agent keeps the lesson as memory. If something went wrong, took several tries or a person corrected it, the agent keeps a preference, a fact or the steps that worked. Lessons shared beyond one person wait for approval, and anything drawn from outside content is only ever offered. Each agent also has its own switch.",
            )}
          </p>
          {!values.learnsFromWork && loaded.learnsFromWork && (
            <Callout tone="w">
              {t(
                "Every agent stops keeping lessons, whatever its own switch says. Memories already kept stay.",
              )}
            </Callout>
          )}
        </>
      ),
    },
    {
      id: "allowance",
      label: t("Monthly allowance per person"),
      keys: ["personMonthlyMessages"],
      content: (
        <>
          <p className="es-note">
            {t(
              "How many questions each person may ask the agents in a calendar month. Desk warns people as they get close and says when it refreshes. Each agent's own budget and daily limit still apply.",
            )}
          </p>
          <F label={t("Questions per person")}>
            <Controller
              control={form.control}
              name="personMonthlyMessages"
              render={({ field }) => (
                <Seg
                  v={field.value}
                  label={t("Questions per person")}
                  opts={personAllowanceOptions(field.value).map(
                    (value) => [value, value === 0 ? t("Unlimited") : value.toLocaleString()] as const,
                  )}
                  onChange={field.onChange}
                />
              )}
            />
          </F>
        </>
      ),
    },
    {
      id: "share",
      label: t("Share corrections for model training"),
      keys: ["aiTrainingConsent"],
      actions: (
        <HeaderSwitch
          control={form.control}
          name="aiTrainingConsent"
          label={t("Share corrections for model training")}
        />
      ),
      content: (
        <>
          <p className="es-note">
            {t(
              "Anonymized document corrections help improve extraction for every customer. Trenova keeps what was read from a document beside what a person confirmed, so accuracy can be measured. Turning this on lets those corrections be anonymized and used for training. Turning it off keeps them out of any training after the change.",
            )}
          </p>
          <TrainingExportHistory />
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
          <Ic n="cog" s={15} />
        </span>
      }
      title={t("Organization-wide")}
      subtitle={t("Applies to every agent and overrides their own settings")}
    />
  );
}

function HeaderSwitch({
  control,
  name,
  label,
}: {
  control: Control<PolicyFormValues>;
  name: "earnedAutonomy" | "learnsFromWork" | "aiTrainingConsent";
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

function PromotionPreview({
  loading,
  promotions,
}: {
  loading: boolean;
  promotions?: ToolPromotion[];
}) {
  const t = useT();

  if (loading) {
    return <Callout tone="i">{t("Checking which tools have earned it…")}</Callout>;
  }
  if (!promotions || promotions.length === 0) {
    return (
      <Callout tone="k">
        {t("No tool has a long enough streak yet. Nothing changes when you save.")}
      </Callout>
    );
  }

  const named = formatList(
    promotions.map((promotion) =>
      t("{0} on {1}", describeToolCall(promotion.toolName, null).title, promotion.agentName),
    ),
  );

  return (
    <Callout tone="i">
      {promotions.length === 1
        ? t("{0} has enough clean approvals already, so it moves up a tier when you save.", named)
        : t(
            "{0} have enough clean approvals already, so they move up a tier when you save.",
            named,
          )}
    </Callout>
  );
}
