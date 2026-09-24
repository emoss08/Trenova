import { useT } from "@trenova/shared/i18n/use-t";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextChipsField } from "@/components/fields/text-chips-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { SectionPanel } from "@/components/section-panel";
import { fetchAgentDefinitions } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useQuery } from "@tanstack/react-query";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { triggerChoices } from "../../activity/agent-badges";
import { ArgumentRulesField } from "./argument-rules-field";
import { TOOL_MODE_LABEL, toolMatchModes, type EvalCaseFormValues } from "./eval-case-model";

export const EVAL_CASE_AGENT_CHOICES_KEY = "agent-eval-case-agents";

/**
 * Everything a case can be told to expect: the question and how it is judged,
 * the tools a good answer calls and with what, the proposals people approved or
 * turned down, and what the reply must and must not say.
 */
export function EvalCaseEditor({ creating }: { creating: boolean }) {
  return (
    <div className="flex flex-col gap-4">
      <QuestionSection creating={creating} />
      <ToolsSection />
      <ProposalsSection />
      <ReplySection />
    </div>
  );
}

function QuestionSection({ creating }: { creating: boolean }) {
  const t = useT();
  const { control } = useFormContext<EvalCaseFormValues>();
  const agents = useQuery({
    queryKey: [EVAL_CASE_AGENT_CHOICES_KEY],
    queryFn: ({ signal }) => fetchAgentDefinitions({}, { signal }),
    enabled: creating,
  });

  return (
    <SectionPanel
      title={t("The question")}
      help={t(
        "What the agent is asked on every replay. A case captured from a conversation keeps the conversation before it and what the person was looking at.",
      )}
    >
      <FormGroup cols={2} className="p-3">
        {creating ? (
          <>
            <FormControl>
              <SelectField
                control={control}
                name="agentDefinitionId"
                label={t("Agent")}
                placeholder={agents.isLoading ? t("Loading agents") : t("Pick an agent")}
                options={(agents.data ?? []).map((agent) => ({
                  value: agent.id,
                  label: agent.name,
                }))}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="trigger"
                label={t("Asked as")}
                description={t("A conversation, or a run the agent starts on its own.")}
                options={triggerChoices.map((choice) => ({
                  value: choice.value,
                  label: t(choice.label),
                }))}
              />
            </FormControl>
          </>
        ) : null}
        <FormControl cols="full">
          <InputField
            control={control}
            name="title"
            label={t("Title")}
            placeholder={t("Put an unpaid load on hold")}
            maxLength={200}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="input"
            label={t("Question")}
            rows={4}
            placeholder={t("Put S-100 on hold until Acme pays the open balance.")}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="rubric"
            label={t("Rubric")}
            rows={3}
            maxLength={4000}
            placeholder={t("A good answer names the balance and says when the hold lifts.")}
            description={t("What a judge weighs beyond the checks below. Optional.")}
          />
        </FormControl>
        <FormControl>
          <DateField
            control={control}
            name="expiresAt"
            label={t("Expires")}
            placeholder={t("Kept for the retention period")}
            description={t("After this day the case is purged.")}
            clearable
          />
        </FormControl>
      </FormGroup>
    </SectionPanel>
  );
}

function ToolsSection() {
  const t = useT();
  const { control } = useFormContext<EvalCaseFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "tools" });

  return (
    <SectionPanel
      title={t("Tools")}
      count={fields.length}
      help={t(
        "Calling a forbidden tool, or any tool the agent did not hold when the case was captured, fails the case outright.",
      )}
      action={
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => append({ name: "", args: [] })}
        >
          <PlusIcon />
          {t("Expect a tool")}
        </Button>
      }
    >
      <div className="flex flex-col gap-3 p-3">
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="toolMode"
              label={t("Tool choice")}
              options={toolMatchModes.map((mode) => ({
                value: mode,
                label: t(TOOL_MODE_LABEL[mode]),
              }))}
            />
          </FormControl>
        </FormGroup>
        {fields.map((field, index) => (
          <div key={field.id} className="border-border flex flex-col gap-2 rounded-md border p-3">
            <div className="flex items-start gap-2">
              <div className="flex-1">
                <InputField
                  control={control}
                  name={`tools.${index}.name`}
                  label={t("Tool")}
                  placeholder={t("place_shipment_hold")}
                />
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                className="mt-5"
                aria-label={t("Remove this tool")}
                onClick={() => remove(index)}
              >
                <Trash2Icon />
              </Button>
            </div>
            <ArgumentRulesField name={`tools.${index}.args`} withValues />
          </div>
        ))}
        <FormGroup cols={2}>
          <FormControl>
            <TextChipsField
              control={control}
              name="forbiddenTools"
              label={t("Forbidden tools")}
              placeholder={t("cancel_shipment")}
              description={t("Calling any of these fails the case.")}
            />
          </FormControl>
          <FormControl>
            <TextChipsField
              control={control}
              name="heldTools"
              label={t("Held tools")}
              placeholder={t("get_shipment")}
              description={t("What the agent held when the case was captured.")}
            />
          </FormControl>
        </FormGroup>
      </div>
    </SectionPanel>
  );
}

function ProposalsSection() {
  const t = useT();
  const { control } = useFormContext<EvalCaseFormValues>();
  const { fields, append, remove } = useFieldArray({ control, name: "proposals" });

  return (
    <SectionPanel
      title={t("Proposals")}
      count={fields.length}
      help={t(
        "An approved proposal is expected with the parameters the person approved, including any they corrected. A rejected one is expected to stay dropped.",
      )}
      action={
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() =>
            append({
              toolName: "",
              rejected: false,
              params: "{}",
              sourceProposalId: "",
              rules: [],
            })
          }
        >
          <PlusIcon />
          {t("Expect a proposal")}
        </Button>
      }
    >
      <div className="flex flex-col gap-3 p-3">
        {fields.length === 0 ? (
          <p className="text-muted-foreground text-xs">{t("No proposal is expected.")}</p>
        ) : null}
        {fields.map((field, index) => (
          <ProposalRow key={field.id} index={index} onRemove={() => remove(index)} />
        ))}
      </div>
    </SectionPanel>
  );
}

function ProposalRow({ index, onRemove }: { index: number; onRemove: () => void }) {
  const t = useT();
  const { control } = useFormContext<EvalCaseFormValues>();
  const rejected = useWatch({ control, name: `proposals.${index}.rejected` });

  return (
    <div className="border-border flex flex-col gap-2 rounded-md border p-3">
      <div className="flex items-start gap-2">
        <div className="grid flex-1 grid-cols-2 gap-2">
          <InputField
            control={control}
            name={`proposals.${index}.toolName`}
            label={t("Tool")}
            placeholder={t("update_rate")}
          />
          <SwitchField
            control={control}
            name={`proposals.${index}.rejected`}
            label={t("A person rejected it")}
            description={t("Expect the agent not to propose it again.")}
            outlined
          />
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          className="mt-5"
          aria-label={t("Remove this proposal")}
          onClick={onRemove}
        >
          <Trash2Icon />
        </Button>
      </div>
      <TextareaField
        control={control}
        name={`proposals.${index}.params`}
        label={rejected ? t("Parameters it proposed") : t("Parameters the person approved")}
        rows={4}
        className="font-mono"
      />
      <ArgumentRulesField name={`proposals.${index}.rules`} withValues={false} />
    </div>
  );
}

function ReplySection() {
  const t = useT();
  const { control } = useFormContext<EvalCaseFormValues>();

  return (
    <SectionPanel
      title={t("The reply")}
      help={t(
        "Mention rules match words in the reply, ignoring case. Every figure the reply cites must also appear in what the agent was given.",
      )}
    >
      <FormGroup cols={2} className="p-3">
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="expectRefusal"
            label={t("Expect a refusal")}
            description={t(
              "The agent should decline. Answering fails the case, as does refusing when this is off.",
            )}
            outlined
          />
        </FormControl>
        <FormControl>
          <TextChipsField
            control={control}
            name="mustMention"
            label={t("Must mention")}
            placeholder={t("on hold")}
          />
        </FormControl>
        <FormControl>
          <TextChipsField
            control={control}
            name="mustNotMention"
            label={t("Must not mention")}
            placeholder={t("cancelled")}
          />
        </FormControl>
      </FormGroup>
    </SectionPanel>
  );
}
