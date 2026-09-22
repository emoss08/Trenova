import { CronCadenceField } from "@/components/fields/cron-cadence-field";
import { DateField } from "@/components/fields/date-field/date-field";
import { FieldWrapper } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextChipsField } from "@/components/fields/text-chips-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { toneVar } from "@/components/kpi/tone";
import { queries } from "@/lib/queries";
import type { AgentTemplate, AutonomyTier, OutputMode, TriggerMode } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { brandMarkFor } from "@trenova/shared/components/ui/logos/registry";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTimezoneLabel, listTimezones } from "@trenova/shared/lib/timezones";
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
import { providerBrandDomain } from "../providers/provider-brand";
import { toSaveRequest, type AgentFormValues } from "./agent-form-schema";
import { IdentityPicker } from "./identity-picker";
import { PromptPreviewSheet } from "./prompt-preview-sheet";
import { applyTemplateStarter } from "./template-fill";
import { TemplatePicker } from "./template-picker";
import { ToolSummary } from "./tool-summary";
import { AgentScorecardPanel } from "./scorecard";
import { TrackRecordSection } from "./track-record";
import { BudgetStatusSection, ToolLimitsField } from "./budget";

type AgentFormProps = {
  mode: "create" | "edit";
  /** The saved agent being edited; its ledger is read by this id. */
  agentId?: string;
  /** Set for the agents the platform itself fires; they cannot be deleted or re-triggered. */
  systemKey?: string;
};

/**
 * How long a change may sit undecided. The colour is the exposure, not the
 * duration: a proposal nobody has looked at for a week is a week of a decision
 * the agent thought was worth making going unmade.
 */
const DECISION_TIMEOUTS = [
  { label: "1 hour", value: 3600, tone: "success" },
  { label: "4 hours", value: 14400, tone: "success" },
  { label: "1 day", value: 86400, tone: "info" },
  { label: "3 days", value: 259200, tone: "warning" },
  { label: "7 days", value: 604800, tone: "danger" },
] as const;

export function AgentForm({ mode, agentId = "", systemKey = "" }: AgentFormProps) {
  const t = useT();
  const { control, setValue, getValues } = useFormContext<AgentFormValues>();

  const templatesQuery = useQuery(queries.assistant.agentTemplates());
  const catalogQuery = useQuery(queries.assistant.toolCatalog());
  const eventsQuery = useQuery(queries.assistant.eventKinds());
  const providersQuery = useQuery(queries.aiProvider.list());
  const providerCatalogQuery = useQuery(queries.aiProvider.catalog());

  const template = useWatch({ control, name: "template" });
  const name = useWatch({ control, name: "name" });
  const icon = useWatch({ control, name: "icon" });
  const accent = useWatch({ control, name: "accent" });
  const triggerMode = useWatch({ control, name: "triggerMode" });
  const ceiling = useWatch({ control, name: "autonomyCeiling" });
  const toolNames = useWatch({ control, name: "toolNames" });
  const toolTiers = useWatch({ control, name: "toolTiers" });
  const shadowMode = useWatch({ control, name: "shadowMode" });
  const { field: triggerField } = useController({ control, name: "triggerMode" });
  const { field: ceilingField } = useController({ control, name: "autonomyCeiling" });
  const { field: outputField } = useController({ control, name: "outputMode" });

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
        .map((provider) => {
          const Mark = brandMarkFor({
            domain: providerBrandDomain(provider, providerCatalogQuery.data?.presets ?? []),
          });

          return {
            label: provider.name,
            value: provider.id,
            description: provider.model,
            icon: Mark ? <Mark className="size-4 shrink-0" /> : undefined,
          };
        }),
    [providersQuery.data, providerCatalogQuery.data?.presets],
  );

  const triggerItems = [
    { value: "Chat" as const, label: t("Chat"), icon: MessageSquareIcon },
    { value: "Scheduled" as const, label: t("Scheduled"), icon: CalendarClockIcon },
    { value: "Event" as const, label: t("Event"), icon: BoltIcon },
    { value: "Continuous" as const, label: t("Continuous"), icon: RepeatIcon },
  ];

  return (
    <div className="flex flex-col gap-7">
      {isSystem && (
        <Alert variant="info" size="sm">
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
              description={t("What people see when they pick this agent.")}
            />
          </FormControl>
          <FormControl>
            <FieldWrapper
              label={t("Face")}
              description={t("Left alone, Trenova picks one and keeps it.")}
            >
              <IdentityPicker
                name={name}
                template={template}
                icon={icon}
                accent={accent}
                onIconChange={(value) => setValue("icon", value, { shouldDirty: true })}
                onAccentChange={(value) => setValue("accent", value, { shouldDirty: true })}
              />
            </FieldWrapper>
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              name="description"
              control={control}
              label={t("Description")}
              placeholder={t("Looks up shipments and drivers for the night dispatch team.")}
              minRows={2}
              maxRows={4}
              description={t("One line on what this agent is for, shown under its name.")}
            />
          </FormControl>
          {mode === "create" && (
            <FormControl cols="full">
              <TemplatePicker
                control={control}
                templates={templatesQuery.data?.templates ?? []}
                isLoading={templatesQuery.isLoading}
                onSelect={onTemplate}
              />
            </FormControl>
          )}
        </FormGroup>
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
          "What it may look up and what it may change. Reads run as soon as the agent asks; a change waits at its own tier, never above the ceiling below.",
        )}
      >
        <ToolSummary
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
        <FieldWrapper
          label={t("Ceiling")}
          description={t("No tool may go beyond this, whatever it is set to on its own.")}
        >
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
          <FormControl cols="full">
            <SelectField
              name="decisionTimeoutSeconds"
              control={control}
              label={t("Proposals expire after")}
              options={DECISION_TIMEOUTS.map((preset) => ({
                label: t(preset.label),
                value: preset.value,
                color: toneVar(preset.tone),
              }))}
              description={t(
                "A proposal nobody decides on in this time expires and its run is closed.",
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              name="shadowMode"
              control={control}
              label={t("Shadow mode")}
              description={t(
                "Runs and records its proposals for review, but nothing it asks for is offered for approval.",
              )}
              outlined
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              name="simulationMode"
              control={control}
              label={t("Simulation")}
              description={t(
                "Its writes are previewed and recorded as what they would have changed, never made. Approvals still count.",
              )}
              outlined
              position="left"
            />
          </FormControl>
        </FormGroup>
        {ceiling === "AutoExecute" && !shadowMode && (
          <Alert variant="warning" size="sm">
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
        title={t("Budget")}
        description={t(
          "What the agent may spend and do on its own. A cap that is reached stops it until the month or the day rolls over.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              name="monthlyBudgetUsd"
              control={control}
              label={t("Monthly budget")}
              sideText={t("USD")}
              min={0}
              decimalScale={2}
              step={5}
              placeholder={t("No cap")}
              description={t("Across every run and conversation of this agent.")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              name="dailyRunLimit"
              control={control}
              label={t("Runs per day")}
              min={0}
              max={10000}
              placeholder={t("No cap")}
              description={t("Zero means no cap.")}
            />
          </FormControl>
          <FormControl cols="full">
            <FieldWrapper
              label={t("Daily limit per change tool")}
              description={t("How many times each change may run in a day; empty means no limit.")}
            >
              <ToolLimitsField toolNames={toolNames} tools={catalogQuery.data?.tools ?? []} />
            </FieldWrapper>
          </FormControl>
        </FormGroup>
        {mode === "edit" && agentId !== "" && <BudgetStatusSection agentId={agentId} />}
      </FormSection>

      {mode === "edit" && agentId !== "" && (
        <FormSection
          title={t("How it has been doing")}
          description={t(
            "Counted from this agent's runs, its proposals and what it cost. Time saved is an estimate: it prices the clerical work a carried-out change replaces, never the decision to allow it.",
          )}
        >
          <AgentScorecardPanel agentDefinitionId={agentId} />
        </FormSection>
      )}

      {mode === "edit" && agentId !== "" && (
        <FormSection
          title={t("Track record")}
          description={t(
            "How people have decided on each tool's proposals. With earned autonomy on for the organization, a streak of clean approvals moves a tool up one tier; a rejection or a failed run takes an earned tier back.",
          )}
        >
          <TrackRecordSection
            agentId={agentId}
            tools={catalogQuery.data?.tools ?? []}
            tiers={toolTiers}
            ceiling={ceiling}
          />
        </FormSection>
      )}

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
              <DateField
                name="endsAt"
                control={control}
                label={t("Stop after")}
                placeholder={t("Runs until switched off")}
                clearable
                description={t("Optional. The agent switches itself off after this date.")}
              />
            </FormControl>
          </FormGroup>
        )}

        {triggerMode === "Event" && (
          <FormGroup cols={2}>
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
            <FormControl>
              <DateField
                name="endsAt"
                control={control}
                label={t("Stop after")}
                placeholder={t("Runs until switched off")}
                clearable
                description={t("Optional. The agent switches itself off after this date.")}
              />
            </FormControl>
          </FormGroup>
        )}
      </FormSection>

      <FormSection
        title={t("Model and limits")}
        description={t(
          "Which provider answers, what the agent is told about its surroundings, and what one run may spend.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SelectField
              name="preferredProviderId"
              control={control}
              label={t("Preferred provider")}
              options={providerOptions}
              placeholder={t("Automatic")}
              isClearable
              description={t("Tried first. Automatic follows the routing on the Providers tab.")}
            />
          </FormControl>
          <FormControl cols="full">
            <FieldWrapper
              label={t("Replies as")}
              description={t("A conversation answers in prose; a report in sections.")}
            >
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
                {
                  value: "Memory",
                  label: t("What the organization recorded"),
                  description: t("Standing instructions, facts and corrections from AI Control"),
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
              position="left"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
