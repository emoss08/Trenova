import { InputField } from "@/components/fields/input-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import {
  AGENT_TOOL_RULE_LIST_KEY,
  AGENT_TOOL_SAFETY_LIST_KEY,
  fetchToolRules,
  saveAgentToolRule,
  type AgentToolPolicy,
  type AgentToolRuleImpact,
} from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import type { EditFields } from "../edit/change-review";
import { EditSheet, type EditSection } from "../edit/edit-sheet";
import { aicFieldTrigger } from "../edit/field-trigger";
import { useEditFlow } from "../edit/use-edit-flow";
import { Ic } from "../kit/ic";
import { Tile } from "../kit/marks";
import {
  EXTERNAL_READ_ORDER,
  TIER_ORDER,
  answerLabel,
  egressLabel,
  externalReadLabel,
  kindLabel,
  needsLabel,
  tierLabel,
  widestEgress,
} from "./safety-model";
import {
  externalReadAllowed,
  movedBy,
  reasonMissing,
  ruleHasTier,
  tierAllowed,
  toToolRuleForm,
  toToolRuleInput,
  toolRuleFormSchema,
  type ToolRuleFormValues,
} from "./tool-rule-form";

type ToolRuleEditorProps = {
  open: boolean;
  policy: AgentToolPolicy;
  holders: readonly AgentDefinitionRow[];
  onClose: () => void;
  /** The tool as saved, so whatever shows it next reads the new version. */
  onSaved?: (tool: AgentToolPolicy) => void;
};

/**
 * One tool's rule for this organization, in the shared editor. A rule here only ever
 * holds a tool lower than it declares, so looser choices are shown but closed. Before
 * saving it says which agents holding the tool would act differently, and a change to
 * the most freedom takes a reason, which goes to the audit trail with the change.
 */
export function ToolRuleEditor({ open, policy, holders, onClose, onSaved }: ToolRuleEditorProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const loaded = useMemo(() => toToolRuleForm(policy), [policy]);
  const form = useForm<ToolRuleFormValues>({
    resolver: zodResolver(toolRuleFormSchema),
    defaultValues: loaded,
    values: open ? loaded : undefined,
    resetOptions: { keepDirtyValues: true },
  });
  const values = useWatch({ control: form.control }) as ToolRuleFormValues;
  const input = toToolRuleInput(values);
  const impact = useQuery({
    ...queries.agentSafety.ruleImpact(policy.name, {
      maxTier: input.maxTier,
      readsExternal: input.readsExternal,
    }),
    enabled: open,
    placeholderData: keepPreviousData,
  });

  const save = useCallback(
    async (next: ToolRuleFormValues) => {
      const saved = await saveAgentToolRule(policy.name, next.version, toToolRuleInput(next));
      onSaved?.(saved.tool);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.agentSafety._def }),
        queryClient.invalidateQueries({ queryKey: [AGENT_TOOL_RULE_LIST_KEY] }),
        queryClient.invalidateQueries({ queryKey: [AGENT_TOOL_SAFETY_LIST_KEY] }),
      ]);
      toast.success(
        saved.affected.length > 0
          ? t(
              "{0} rule saved · {1, plural, one {# agent changed} other {# agents changed}}",
              policy.title,
              saved.affected.length,
            )
          : t("{0} rule saved", policy.title),
      );
      return toToolRuleForm(saved.tool);
    },
    [onSaved, policy.name, policy.title, queryClient, t],
  );

  const flow = useEditFlow({
    form,
    onSave: save,
    onClose,
    enabled: open,
    resourceName: policy.title,
    versionField: "version",
    invalid: reasonMissing(values, form.formState.defaultValues as ToolRuleFormValues)
      ? t("Add a reason for the audit trail")
      : null,
    loadLatest: async () => {
      const latest = (await fetchToolRules()).get(policy.name);
      return latest ? toToolRuleForm(latest) : loaded;
    },
  });

  const fields: EditFields = {
    maxTier: {
      label: t("Maximum"),
      format: (value) => tierLabel(t, value as ToolRuleFormValues["maxTier"]),
    },
    readsExternal: {
      label: t("Outside text"),
      format: (value) => externalReadLabel(t, value as ToolRuleFormValues["readsExternal"]),
    },
    reason: { label: t("Reason") },
  };

  const impacts = impact.data ?? [];
  const moved = movedBy(impacts);
  const sections: EditSection[] = [
    {
      id: "rule",
      label: t("Rule"),
      keys: ["maxTier", "readsExternal", "reason"],
      content: (
        <>
          <div className="kvs">
            <span>
              <em>{t("Reaches")}</em>
              <b>{egressLabel(t, widestEgress(policy.egress))}</b>
            </span>
            <span>
              <em>{t("Needs")}</em>
              <b>{needsLabel(t, policy)}</b>
            </span>
            <span>
              <em>{t("Kind")}</em>
              <b>{kindLabel(t, policy.kind)}</b>
            </span>
          </div>
          {ruleHasTier(policy) ? (
            <SegmentedField
              control={form.control}
              name="maxTier"
              label={t("Most freedom any agent gets")}
              description={
                TIER_ORDER.every((tier) => tierAllowed(tier, policy.declaredMaxTier))
                  ? t("An agent's own ceiling can hold it lower, never higher.")
                  : `${t("An agent's own ceiling can hold it lower, never higher.")} ${t(
                      "The tool's own rule allows at most {0}",
                      tierLabel(t, policy.declaredMaxTier),
                    )}.`
              }
              options={TIER_ORDER.map((tier) => ({
                value: tier,
                label: tierLabel(t, tier),
                disabled: !tierAllowed(tier, policy.declaredMaxTier),
              }))}
            />
          ) : (
            <p className="es-note">{t("Reads never change records, so they always run.")}</p>
          )}
          <SegmentedField
            control={form.control}
            name="readsExternal"
            label={t("Treat what it returns as outside text")}
            description={
              EXTERNAL_READ_ORDER.every((read) =>
                externalReadAllowed(read, policy.declaredReadsExternal),
              )
                ? t("Outside text can't trigger an automatic action in the same run.")
                : `${t("Outside text can't trigger an automatic action in the same run.")} ${t(
                    "The tool's own rule already treats more as outside text",
                  )}.`
            }
            options={EXTERNAL_READ_ORDER.map((read) => ({
              value: read,
              label: externalReadLabel(t, read),
              disabled: !externalReadAllowed(read, policy.declaredReadsExternal),
            }))}
          />
          <InputField
            control={form.control}
            name="reason"
            rules={values.maxTier !== loaded.maxTier ? { required: true } : undefined}
            label={t("Reason")}
            description={t("Saved to the audit trail with the change.")}
            placeholder={t("Two wrong releases last week")}
            inputClassProps={aicFieldTrigger}
          />
        </>
      ),
    },
    {
      id: "who",
      label: t("Who's affected"),
      actions: impact.data && (
        <span className="es-tk">{t("{0} of {1} change", moved.length, impacts.length)}</span>
      ),
      content: impact.isError ? (
        <p className="es-note">
          {t("Who's affected couldn't be worked out. Try again in a moment.")}
        </p>
      ) : !impact.data ? (
        <p className="es-note">{t("Working out who's affected…")}</p>
      ) : impacts.length > 0 ? (
        <AffectedList impacts={impacts} holders={holders} />
      ) : (
        <p className="es-note">{t("No agent holds this tool.")}</p>
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
          <Ic n="tool" s={15} />
        </span>
      }
      title={policy.title}
      subtitle={<span className="mono">{policy.name}</span>}
    />
  );
}

function AffectedList({
  impacts,
  holders,
}: {
  impacts: readonly AgentToolRuleImpact[];
  holders: readonly AgentDefinitionRow[];
}) {
  const t = useT();
  const byId = new Map(holders.map((agent) => [agent.id, agent]));

  return (
    <div className="aff">
      {impacts.map((impact) => {
        const changed = impact.before !== impact.after;
        return (
          <div key={impact.agentId} className={cn("aff-r", changed && "ch")}>
            <Tile agent={byId.get(impact.agentId)} s={22} />
            <b>{impact.agentName}</b>
            <span className="sp" />
            {changed ? (
              <span className="cr-v">
                <s>{answerLabel(t, impact.before)}</s>
                <Ic n="arrowR" s={11} />
                <b>{answerLabel(t, impact.after)}</b>
              </span>
            ) : (
              <span className="dim">{answerLabel(t, impact.before)}</span>
            )}
          </div>
        );
      })}
    </div>
  );
}
