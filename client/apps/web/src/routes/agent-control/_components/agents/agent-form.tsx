import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  saveAgentDefinitionRequestSchema,
  type AgentDefinition,
  type AgentTemplate,
  type AgentTemplateKind,
  type SaveAgentDefinitionRequest,
} from "@/types/assistant";
import { describeToolCall } from "@/routes/assistant/_components/tool-presentation";
import { useQuery } from "@tanstack/react-query";
import { ShieldAlertIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

type AgentFormProps = {
  agent: AgentDefinition | null;
  templates: AgentTemplate[];
  onClose: () => void;
  onSaved: () => Promise<void> | void;
};

type FormValues = Omit<SaveAgentDefinitionRequest, "toolNames"> & {
  toolNames: string[] | null;
};

function toFormValues(agent: AgentDefinition | null): FormValues {
  const base = saveAgentDefinitionRequestSchema.parse({
    name: agent?.name ?? "",
    description: agent?.description ?? "",
    template: agent?.template ?? null,
    instructions: agent?.instructions ?? "",
    guardrails: agent?.guardrails ?? [],
    toolNames: agent?.toolNames ?? [],
    toolTiers: agent?.toolTiers ?? {},
    autonomyCeiling: agent?.autonomyCeiling ?? "Propose",
    enabled: agent?.enabled ?? true,
    shadowMode: agent?.shadowMode ?? false,
    decisionTimeoutSeconds: agent?.decisionTimeoutSeconds ?? 86400,
    triggerMode: agent?.triggerMode ?? "Chat",
    cronExpression: agent?.cronExpression ?? "",
    cronTimezone: agent?.cronTimezone ?? "",
    eventKinds: agent?.eventKinds ?? [],
    intervalSeconds: agent?.intervalSeconds ?? 0,
    endsAt: agent?.endsAt ?? null,
    maxConcurrentRuns: agent?.maxConcurrentRuns ?? 1,
    runTimeoutSeconds: agent?.runTimeoutSeconds ?? 600,
    maxToolCalls: agent?.maxToolCalls ?? 12,
    contextProviders: agent?.contextProviders ?? [],
    outputMode: agent?.outputMode ?? "Conversational",
    preferredProviderId: agent?.preferredProviderId ?? "",
    version: agent?.version ?? 0,
  });

  return { ...base, toolNames: base.toolNames.length > 0 ? base.toolNames : null };
}

export function AgentForm({ agent, templates, onClose, onSaved }: AgentFormProps) {
  const t = useT();
  const catalogQuery = useQuery(queries.assistant.toolCatalog());

  const form = useForm<FormValues>({ defaultValues: toFormValues(agent) });
  const { control, handleSubmit, setValue, getValues } = form;

  const templateKind = useWatch({ control, name: "template" });
  const template = useMemo(
    () => templates.find((item) => item.template === templateKind),
    [templates, templateKind],
  );

  // A template is a starting point: picking one fills instructions and tools
  // that are still blank, and never overwrites what a person already wrote.
  const onTemplateChange = useCallback(
    (value: string) => {
      const picked = templates.find((item) => item.template === (value as AgentTemplateKind));
      if (!picked) {
        return;
      }
      if (getValues("instructions").trim() === "") {
        setValue("instructions", picked.starterInstructions, { shouldDirty: true });
      }
      if ((getValues("toolNames") ?? []).length === 0 && picked.starterTools.length > 0) {
        setValue("toolNames", picked.starterTools, { shouldDirty: true });
      }
      setValue("autonomyCeiling", picked.starterCeiling, { shouldDirty: true });
    },
    [getValues, setValue, templates],
  );

  const toolOptions = useMemo(
    () =>
      (catalogQuery.data?.tools ?? []).map((tool) => ({
        label: `${describeToolCall(tool.name, null).title} · ${tool.name}`,
        value: tool.name,
        description:
          tool.kind === "query"
            ? `${tool.description} ${t("Reads only.")}`
            : `${tool.description} ${t("Changes data.")}`,
      })),
    [catalogQuery.data?.tools, t],
  );

  const saveMutation = useApiMutation({
    mutationFn: (values: FormValues) => {
      const payload: SaveAgentDefinitionRequest = {
        ...values,
        toolNames: values.toolNames ?? [],
      };

      return agent
        ? apiService.agentDefinitionService.update(agent.id, payload)
        : apiService.agentDefinitionService.create(payload);
    },
    form,
    resourceName: "Agent",
    onSuccess: async () => {
      toast.success(agent ? t("Agent updated") : t("Agent added"));
      await onSaved();
      onClose();
    },
  });

  return (
    <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            name="name"
            control={control}
            label={t("Name")}
            placeholder={t("Night dispatch helper")}
            rules={{ required: t("Name is required") }}
            description={t("How this agent appears when someone starts a conversation.")}
          />
        </FormControl>

        <FormControl>
          <SelectField
            name="template"
            control={control}
            label={t("Start from a template")}
            options={templates.map((item) => ({ label: item.label, value: item.template }))}
            description={
              template?.description ??
              t("Optional. A template fills in instructions and tools you can change freely.")
            }
            onValueChange={onTemplateChange}
            isClearable
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            name="description"
            control={control}
            label={t("Description")}
            placeholder={t("Looks up shipments and drivers for the night dispatch team.")}
            description={t(
              "Shown to people choosing an agent, so say who it is for and what it can do.",
            )}
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            name="instructions"
            control={control}
            label={t("Instructions")}
            placeholder={t(
              "You support the night dispatch desk. Check hours of service before assigning anyone…",
            )}
            description={t(
              "Who this agent is, what it prioritises, the policies it follows and how it should talk. These instructions are authoritative; Trenova only adds its tenant and safety boundaries in front of them.",
            )}
          />
        </FormControl>
      </FormGroup>

      <FormGroup cols={1}>
        <FormControl cols="full">
          <MultiCheckboxField
            name="toolNames"
            control={control}
            label={t("Tools this agent may use")}
            description={t(
              "Any tool the system provides can be enabled. Leaving them all unchecked gives an agent that only answers from what it is told.",
            )}
            options={toolOptions}
          />
        </FormControl>
      </FormGroup>

      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            name="autonomyCeiling"
            control={control}
            label={t("Autonomy ceiling")}
            options={[
              { label: t("Propose only"), value: "Propose" },
              { label: t("Act with approval"), value: "ActWithApproval" },
              { label: t("Act automatically"), value: "AutoExecute" },
            ]}
            description={t(
              "The most any tool may do on its own. Each tool can be held below it, never above it.",
            )}
          />
        </FormControl>

        <FormControl>
          <SwitchField
            name="enabled"
            control={control}
            label={t("Enabled")}
            description={t("Disabled agents cannot be used or continued.")}
            outlined
          />
        </FormControl>
      </FormGroup>

      <Alert variant="info">
        <ShieldAlertIcon className="size-4" />
        <AlertTitle>
          {t("Every tool call is checked against the person using the agent")}
        </AlertTitle>
        <AlertDescription>
          {t(
            "The agent can only read or change what the person talking to it could read or change themselves. Below the automatic tier, a change becomes a proposal someone reviews first.",
          )}
        </AlertDescription>
      </Alert>

      <DialogFooter className="flex flex-row items-center sm:justify-between">
        <Button type="button" variant="outline" onClick={onClose}>
          {t("Cancel")}
        </Button>
        <Button
          type="submit"
          size="sm"
          isLoading={saveMutation.isPending}
          loadingText={t("Saving...")}
        >
          {agent ? t("Save changes") : t("Add agent")}
        </Button>
      </DialogFooter>
    </Form>
  );
}
