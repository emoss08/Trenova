import { describeToolCall } from "@/components/assistant/tool-presentation";
import { EditFieldRow } from "@/components/edit-sheet/field-row";
import type { EditFields } from "@/components/edit-sheet/change-review";
import { EditSheet, type EditSection } from "@/components/edit-sheet/edit-sheet";
import { useEditFlow } from "@/components/edit-sheet/use-edit-flow";
import {
  AGENT_CONTROL_QUERY_KEY,
  fetchAgentControl,
  updateAgentControl,
  type AgentControl,
} from "@/lib/graphql/agent-control";
import type { ToolPromotion } from "@/lib/graphql/ai-control";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { Settings01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { Controller, useForm, useWatch, type Control } from "react-hook-form";
import { toast } from "sonner";
import { personAllowanceOptions, promotionThresholdOptions } from "../agent-control-options";
import { tierLabel } from "../safety/safety-model";
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
            ? t("Saved. 1 tool moved up a tier")
            : t("Saved. {0} tools moved up a tier", moved)
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
      label: t("Monthly allowance"),
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
      actions: <HeaderSwitch control={form.control} name="earnedAutonomy" label={t("Earned autonomy")} />,
      note: t(
        "When a tool's proposals are approved unchanged this many times in a row, the tool moves up one tier on that agent, never above the agent's ceiling. A rejection or a failed run takes an earned tier back. Each change is audited and announced.",
      ),
      content: values.earnedAutonomy ? (
        <>
          <EditFieldRow label={t("Clean approvals")} hint={t("In a row, before a tool moves up")}>
            <Controller
              control={form.control}
              name="promotionThreshold"
              render={({ field }) => (
                <SegmentedControl<string>
                  aria-label={t("Clean approvals")}
                  value={String(field.value)}
                  onValueChange={(next) => field.onChange(Number(next))}
                  items={promotionThresholdOptions(field.value).map((option) => ({
                    value: String(option.value),
                    label: option.label,
                  }))}
                />
              )}
            />
          </EditFieldRow>
          {promotes && <PromotionPreview loading={preview.isLoading} promotions={preview.data} />}
        </>
      ) : null,
    },
    {
      id: "learn",
      label: t("Learn from their work"),
      keys: ["learnsFromWork"],
      actions: <HeaderSwitch control={form.control} name="learnsFromWork" label={t("Learn from their work")} />,
      note: t(
        "Once a conversation goes quiet or a background run settles, the agent looks back over it and keeps what it learned as memory. Each person's saving preference still applies, and lessons shared beyond one person wait for someone allowed to approve them.",
      ),
      content:
        !values.learnsFromWork && loaded.learnsFromWork ? (
          <Alert variant="warning" size="sm">
            <AlertDescription>
              {t(
                "Every agent stops keeping lessons, whatever its own switch says. Memories already kept stay.",
              )}
            </AlertDescription>
          </Alert>
        ) : null,
    },
    {
      id: "allowance",
      label: t("Monthly allowance"),
      keys: ["personMonthlyMessages"],
      note: t(
        "How many questions each person may ask the agents in a calendar month. Desk warns them as they get close and says when it refreshes. Each agent's own budget and daily limit still apply.",
      ),
      content: (
        <EditFieldRow label={t("Per person")} hint={t("Questions each month")}>
          <Controller
            control={form.control}
            name="personMonthlyMessages"
            render={({ field }) => (
              <SegmentedControl<string>
                aria-label={t("Monthly allowance")}
                value={String(field.value)}
                onValueChange={(next) => field.onChange(Number(next))}
                items={personAllowanceOptions(field.value).map((value) => ({
                  value: String(value),
                  label: value === 0 ? t("Unlimited") : value.toLocaleString(),
                }))}
              />
            )}
          />
        </EditFieldRow>
      ),
    },
    {
      id: "share",
      label: t("Share corrections"),
      keys: ["aiTrainingConsent"],
      actions: <HeaderSwitch control={form.control} name="aiTrainingConsent" label={t("Share corrections")} />,
      note: t(
        "When someone creates a shipment from a document, Trenova keeps what was read beside what they confirmed. Turn this on to let those corrections be anonymized and used to improve the models that read documents for every customer. Turning it off keeps this organization's corrections out of any training after the change.",
      ),
      content: (
        <>
          {control.aiTrainingConsentChangedAt ? (
            <p className="text-xs text-muted-foreground">
              {t("Last changed {0}", formatUnixDateTime(control.aiTrainingConsentChangedAt))}
            </p>
          ) : null}
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
        <span className="flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <Settings01Icon className="size-4" />
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
      render={({ field }) => (
        <Switch checked={field.value} onCheckedChange={field.onChange} aria-label={label} />
      )}
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
    return (
      <Alert variant="info" size="sm">
        <AlertDescription>{t("Checking which tools have earned it…")}</AlertDescription>
      </Alert>
    );
  }
  if (!promotions || promotions.length === 0) {
    return (
      <Alert variant="success" size="sm">
        <AlertDescription>
          {t("No tool has a long enough streak yet. Nothing changes when you save.")}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Alert variant="info" size="sm">
      <AlertDescription className="flex flex-col gap-1.5">
        <span>
          {promotions.length === 1
            ? t("1 tool has enough clean approvals already, so it moves up a tier when you save.")
            : t(
                "{0} tools have enough clean approvals already, so they move up a tier when you save.",
                promotions.length,
              )}
        </span>
        <ul className="flex flex-col gap-0.5">
          {promotions.map((promotion) => (
            <li key={`${promotion.agentName}:${promotion.toolName}`}>
              {t(
                "{0} on {1}: {2} → {3}",
                describeToolCall(promotion.toolName, null).title,
                promotion.agentName,
                tierLabel(t, promotion.from),
                tierLabel(t, promotion.to),
              )}
            </li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
  );
}
