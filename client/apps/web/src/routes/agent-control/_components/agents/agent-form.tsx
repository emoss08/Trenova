import { useT } from "@trenova/shared/i18n/use-t";
import { CronCadenceField } from "@/components/fields/cron-cadence-field";
import { DatePickerField } from "@/components/fields/date-picker-field";
import { FieldWrapper } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextChipsField } from "@/components/fields/text-chips-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { formatTimezoneLabel, listTimezones } from "@trenova/shared/lib/timezones";
import { queries } from "@/lib/queries";
import type { AgentTemplate, AutonomyTier, OutputMode, TriggerMode } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import {
  BoltIcon,
  CalendarClockIcon,
  FileTextIcon,
  LockIcon,
  MessageSquareIcon,
  RepeatIcon,
  ShieldAlertIcon,
} from "lucide-react";
import { useCallback, useMemo } from "react";
import { useController, useFormContext, useWatch } from "react-hook-form";
import { toSaveRequest, type AgentFormValues } from "./agent-form-schema";
import { PromptPreviewSheet } from "./prompt-preview-sheet";
import { applyTemplateStarter } from "./template-fill";
import { TemplatePicker } from "./template-picker";
import { ToolPicker } from "./tool-picker";

type AgentFormProps = {
  mode: "create" | "edit";
  /** Set for the agents the platform itself fires; they cannot be deleted or re-triggered. */
  systemKey?: string;
};

const DECISION_TIMEOUTS = [
  { label: "1 hour", value: 3600 },
  { label: "4 hours", value: 14400 },
  { label: "1 day", value: 86400 },
  { label: "3 days", value: 259200 },
  { label: "7 days", value: 604800 },
];

export function AgentForm({ mode, systemKey = "" }: AgentFormProps) {
  const t = useT();
  const { control, setValue, getValues } = useFormContext<AgentFormValues>();

  const templatesQuery = useQuery(queries.assistant.agentTemplates());
  const catalogQuery = useQuery(queries.assistant.toolCatalog());
  const eventsQuery = useQuery(queries.assistant.eventKinds());
  const providersQuery = useQuery(queries.aiProvider.list());

  const template = useWatch({ control, name: "template" });
  const triggerMode = useWatch({ control, name: "triggerMode" });
  const ceiling = useWatch({ control, name: "autonomyCeiling" });
  const toolNames = useWatch({ control, name: "toolNames" });
  const toolTiers = useWatch({ control, name: "toolTiers" });
  const shadowMode = useWatch({ control, name: "shadowMode" });
  const { field: triggerField } = useController({ control, name: "triggerMode" });
  const { field: ceilingField } = useController({ control, name: "autonomyCeiling" });
  const { field: outputField } = useController({ control, name: "outputMode" });
  const { field: endsAtField, fieldState: endsAtState } = useController({
    control,
    name: "endsAt",
  });

  const isSystem = systemKey !== "";

  const onTemplate = useCallback(
    (picked: AgentTemplate | null) => {
      const patch = applyTemplateStarter(getValues(), picked);
      for (const [key, value] of Object.entries(patch)) {
        setValue(key as keyof AgentFormValues, value as never, { shouldDirty: true });
      }
    },
    [getValues, setValue],
  );

  const timezoneOptions = useMemo(
    () => listTimezones().map((zone) => ({ label: formatTimezoneLabel(zone), value: zone })),
    [],
  );

  const eventOptions = useMemo(
    () =>
      (eventsQuery.data?.events ?? []).map((event) => ({
        label: event.label,
        value: event.kind,
        description: event.description,
      })),
    [eventsQuery.data?.events],
  );

  const providerOptions = useMemo(
    () =>
      (providersQuery.data ?? [])
        .filter((provider) => provider.enabled && provider.tasks.includes("AssistantChat"))
        .map((provider) => ({
          label: `${provider.name} · ${provider.model}`,
          value: provider.id,
        })),
    [providersQuery.data],
  );

  const triggerItems = [
    { value: "Chat" as const, label: t("Chat"), icon: MessageSquareIcon },
    { value: "Scheduled" as const, label: t("Scheduled"), icon: CalendarClockIcon },
    { value: "Event" as const, label: t("Event"), icon: BoltIcon },
    { value: "Continuous" as const, label: t("Continuous"), icon: RepeatIcon },
  ];

  return (
    <div className="flex flex-col gap-7 pb-14">
      {isSystem && (
        <Alert variant="info">
          <LockIcon className="size-4" />
          <AlertTitle>{t("A system agent")}</AlertTitle>
          <AlertDescription>
            {t(
              "Trenova starts this agent itself. You can change what it is told, which tools it may use and how much it may do, but not when it runs, and it cannot be removed.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <FormSection
        title={t("Identity")}
        description={t("How this agent appears to the people who use it.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              name="name"
              control={control}
              label={t("Name")}
              placeholder={t("Night dispatch helper")}
              rules={{ required: t("Name is required") }}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <TextareaField
              name="description"
              control={control}
              label={t("Description")}
              placeholder={t("Looks up shipments and drivers for the night dispatch team.")}
              minRows={1}
              maxRows={3}
            />
          </FormControl>
        </FormGroup>
        {mode === "create" && (
          <FieldWrapper
            label={t("Start from a template")}
            description={t(
              "A template fills in instructions, tools and a trigger you can change freely. It never limits what the agent may do.",
            )}
          >
            <TemplatePicker
              templates={templatesQuery.data?.templates ?? []}
              value={template}
              isLoading={templatesQuery.isLoading}
              onSelect={onTemplate}
            />
          </FieldWrapper>
        )}
      </FormSection>

      <FormSection
        title={t("Instructions")}
        description={t(
          "Who this agent is, what it prioritises, the policies it follows and how it should talk. These are authoritative; Trenova only adds its tenant and safety boundary in front of them.",
        )}
        action={<PromptPreviewSheet getRequest={() => toSaveRequest(getValues())} />}
      >
        <FormGroup cols={1}>
          <FormControl cols="full">
            <TextareaField
              name="instructions"
              control={control}
              label={t("System instructions")}
              placeholder={t(
                "You support the night dispatch desk. Check hours of service before assigning anyone. Prefer drivers already near the pickup. When something is uncertain, say so and ask.",
              )}
              minRows={8}
              maxRows={24}
              className="font-mono text-xs leading-relaxed"
            />
          </FormControl>
          <FormControl cols="full">
            <TextChipsField
              name="guardrails"
              control={control}
              label={t("Never")}
              description={t(
                "Hard lines the agent must not cross, one per chip. Press Enter after each.",
              )}
              placeholder={t("Promise a delivery time to a customer")}
              maxItems={20}
              maxLength={300}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Tools")}
        titleCount={toolNames.length}
        description={t(
          "Everything the system offers is here. Reads run as soon as the agent asks; a change carries its own autonomy, never above the ceiling below.",
        )}
      >
        <ToolPicker
          tools={catalogQuery.data?.tools ?? []}
          isLoading={catalogQuery.isLoading}
          selected={toolNames}
          tiers={toolTiers}
          ceiling={ceiling}
          onSelectedChange={(next) =>
            setValue("toolNames", next, { shouldDirty: true, shouldValidate: true })
          }
          onTiersChange={(next) =>
            setValue("toolTiers", next, { shouldDirty: true, shouldValidate: true })
          }
        />
      </FormSection>

      <FormSection
        title={t("Autonomy")}
        description={t(
          "The most any tool may do on its own, and what happens while you are still deciding.",
        )}
      >
        <FieldWrapper label={t("Ceiling")}>
          <SegmentedControl<AutonomyTier>
            fullWidth
            value={ceilingField.value}
            onValueChange={(value) => {
              ceilingField.onChange(value);
              setValue("toolTiers", getValues("toolTiers"), { shouldValidate: true });
            }}
            aria-label={t("Autonomy ceiling")}
            items={[
              {
                value: "Propose",
                label: t("Propose only"),
                caption: t("Every change waits for a person"),
              },
              {
                value: "ActWithApproval",
                label: t("Act with approval"),
                caption: t("Changes run once someone approves"),
              },
              {
                value: "AutoExecute",
                label: t("Act automatically"),
                caption: t("Chosen tools run on their own"),
              },
            ]}
          />
        </FieldWrapper>
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              name="decisionTimeoutSeconds"
              control={control}
              label={t("Proposals expire after")}
              options={DECISION_TIMEOUTS.map((preset) => ({
                label: t(preset.label),
                value: preset.value,
              }))}
              description={t(
                "A proposal nobody decides on in this time expires and its run is closed.",
              )}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              name="shadowMode"
              control={control}
              label={t("Shadow mode")}
              description={t(
                "Runs and records its proposals for review, but nothing it asks for is offered for approval.",
              )}
              outlined
            />
          </FormControl>
        </FormGroup>
        {ceiling === "AutoExecute" && !shadowMode && (
          <Alert variant="warning">
            <ShieldAlertIcon className="size-4" />
            <AlertTitle>{t("This agent can change records without asking")}</AlertTitle>
            <AlertDescription>
              {t(
                "Only tools set to Automatic in the list above run on their own, and only within what the person or schedule that started the agent is permitted to do. Consider shadow mode first.",
              )}
            </AlertDescription>
          </Alert>
        )}
      </FormSection>

      <FormSection
        title={t("When it runs")}
        description={t(
          "Talk to it, run it on a schedule, fire it from an event, or keep it going.",
        )}
      >
        <SegmentedControl<TriggerMode>
          fullWidth
          value={triggerField.value}
          onValueChange={(value) => {
            if (isSystem) return;
            triggerField.onChange(value);
          }}
          aria-label={t("Trigger")}
          items={triggerItems.map((item) => ({
            ...item,
            disabled: isSystem && item.value !== triggerField.value,
          }))}
        />

        {triggerMode === "Chat" && (
          <p className="text-muted-foreground text-xs">
            {t("People start conversations with this agent from the assistant on any page.")}
          </p>
        )}

        {triggerMode === "Scheduled" && (
          <FormGroup cols={2}>
            <FormControl cols="full">
              <CronCadenceField control={control} name="cronExpression" verb={t("Run")} />
            </FormControl>
            <FormControl>
              <SelectField
                name="cronTimezone"
                control={control}
                label={t("Time zone")}
                options={timezoneOptions}
                placeholder="UTC"
                description={t("The schedule is read in this zone.")}
              />
            </FormControl>
            <FormControl>
              <FieldWrapper
                label={t("Stop after")}
                description={t("Optional. The agent switches itself off after this date.")}
                error={endsAtState.error?.message}
              >
                <DatePickerField
                  date={endsAtField.value ? new Date(endsAtField.value * 1000) : undefined}
                  setDate={(next) => endsAtField.onChange(next ?? null)}
                />
              </FieldWrapper>
            </FormControl>
          </FormGroup>
        )}

        {triggerMode === "Event" && (
          <FormGroup cols={1}>
            <FormControl cols="full">
              <MultiCheckboxField
                name="eventKinds"
                control={control}
                label={t("Starts when")}
                description={t("Each event starts a run about the record it concerns.")}
                options={eventOptions}
              />
            </FormControl>
          </FormGroup>
        )}

        {triggerMode === "Continuous" && (
          <FormGroup cols={2}>
            <FormControl>
              <NumberField
                name="intervalSeconds"
                control={control}
                label={t("Every")}
                sideText={t("seconds")}
                min={60}
                description={t(
                  "A bounded run starts this often, for as long as the agent is enabled.",
                )}
              />
            </FormControl>
            <FormControl>
              <NumberField
                name="maxConcurrentRuns"
                control={control}
                label={t("At most")}
                sideText={t("runs at once")}
                min={1}
                max={10}
              />
            </FormControl>
            <FormControl cols="full">
              <FieldWrapper
                label={t("Stop after")}
                description={t("Optional. The agent switches itself off after this date.")}
                error={endsAtState.error?.message}
              >
                <DatePickerField
                  date={endsAtField.value ? new Date(endsAtField.value * 1000) : undefined}
                  setDate={(next) => endsAtField.onChange(next ?? null)}
                />
              </FieldWrapper>
            </FormControl>
          </FormGroup>
        )}

        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              name="runTimeoutSeconds"
              control={control}
              label={t("Run timeout")}
              sideText={t("seconds")}
              min={60}
              max={3600}
              description={t("A run that takes longer is stopped.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              name="maxToolCalls"
              control={control}
              label={t("Tool calls per run")}
              min={1}
              max={64}
              description={t("The budget one run may spend looking things up and acting.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Model and context")}
        description={t(
          "Which provider answers, and what the agent is told about its surroundings.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              name="preferredProviderId"
              control={control}
              label={t("Preferred provider")}
              options={providerOptions}
              placeholder={t("Automatic")}
              isClearable
              description={t(
                "Tried first for this agent. Leave on automatic to follow the routing on the Providers tab.",
              )}
            />
          </FormControl>
          <FormControl>
            <FieldWrapper label={t("Replies as")}>
              <SegmentedControl<OutputMode>
                fullWidth
                value={outputField.value}
                onValueChange={outputField.onChange}
                aria-label={t("Output")}
                items={[
                  { value: "Conversational", label: t("Conversation"), icon: MessageSquareIcon },
                  { value: "Report", label: t("Report"), icon: FileTextIcon },
                ]}
              />
            </FieldWrapper>
          </FormControl>
          <FormControl cols="full">
            <MultiCheckboxField
              name="contextProviders"
              control={control}
              label={t("Tell the agent about")}
              description={t("Leave everything unchecked to include all of it.")}
              options={[
                {
                  value: "Organization",
                  label: t("The organization"),
                  description: t("Name, business unit and time zone"),
                },
                {
                  value: "Clock",
                  label: t("The current time"),
                  description: t("So 'today' and 'tomorrow' mean something"),
                },
                {
                  value: "User",
                  label: t("The person asking"),
                  description: t("Name and roles, in a conversation"),
                },
                {
                  value: "Page",
                  label: t("The page they are on"),
                  description: t("The record open when they asked"),
                },
                {
                  value: "Tools",
                  label: t("Its own tools"),
                  description: t("A summary of what each tool does and needs"),
                },
              ]}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection title={t("Availability")}>
        <FormGroup cols={1}>
          <FormControl cols="full">
            <SwitchField
              name="enabled"
              control={control}
              label={t("Enabled")}
              description={t("Disabled agents cannot be talked to, scheduled or fired by events.")}
              outlined
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
