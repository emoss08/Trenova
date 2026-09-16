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
import { apiService } from "@/services/api";
import type { AgentDefinition, AgentTemplate, SaveAgentDefinitionRequest } from "@/types/assistant";
import { InfoIcon, ShieldAlertIcon } from "lucide-react";
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

export function AgentForm({ agent, templates, onClose, onSaved }: AgentFormProps) {
  const t = useT();

  const form = useForm<FormValues>({
    defaultValues: agent
      ? {
          name: agent.name,
          description: agent.description,
          kind: agent.kind,
          focus: agent.focus,
          toolNames: agent.toolNames.length > 0 ? agent.toolNames : null,
          autonomyCeiling: agent.autonomyCeiling,
          enabled: agent.enabled,
          version: agent.version,
        }
      : {
          name: "",
          description: "",
          kind: "GeneralAssistant",
          focus: "",
          toolNames: null,
          autonomyCeiling: "Propose",
          enabled: true,
          version: 0,
        },
  });
  const { control, handleSubmit, setValue } = form;

  const kind = useWatch({ control, name: "kind" });
  const template = useMemo(() => templates.find((item) => item.kind === kind), [templates, kind]);

  // Switching template invalidates the tool selection, since a tool permitted by
  // one template is rejected by another. Clearing it here avoids a save that
  // fails with errors the person did not cause.
  const onKindChange = useCallback(() => {
    setValue("toolNames", null, { shouldDirty: true });
  }, [setValue]);

  const toolOptions = useMemo(
    () =>
      (template?.availableTools ?? []).map((tool) => ({
        label: tool.name,
        value: tool.name,
        description: tool.description,
      })),
    [template?.availableTools],
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

  const readOnlyTemplate = template !== undefined && !template.mutatingAllowed;

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
            name="kind"
            control={control}
            label={t("Template")}
            options={templates.map((item) => ({ label: item.label, value: item.kind }))}
            description={template?.description}
            onValueChange={onKindChange}
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            name="description"
            control={control}
            label={t("Description")}
            placeholder={t("What this agent is for")}
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            name="focus"
            control={control}
            label={t("Organization note (optional)")}
            placeholder={t("We prioritise reefer loads out of Laredo.")}
            description={t(
              "Background preference for this agent. It can narrow what the agent focuses on, but it cannot grant capabilities or override the assistant's rules.",
            )}
          />
        </FormControl>
      </FormGroup>

      {readOnlyTemplate ? (
        <Alert variant="info">
          <InfoIcon className="size-4" />
          <AlertTitle>{t("This template answers questions only")}</AlertTitle>
          <AlertDescription>
            {t(
              "A general assistant can look records up but cannot be given tools that change anything.",
            )}
          </AlertDescription>
        </Alert>
      ) : (
        <FormGroup cols={1}>
          <FormControl cols="full">
            <MultiCheckboxField
              name="toolNames"
              control={control}
              label={t("Tools this agent may use")}
              description={t(
                "Only the tools this template permits are listed. Leaving them all unchecked gives an agent that answers questions without changing anything.",
              )}
              options={toolOptions}
            />
          </FormControl>
        </FormGroup>
      )}

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
              "A cap, not a grant. Each tool keeps its own limit if that limit is stricter.",
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
        <AlertTitle>{t("Writes always wait for a person")}</AlertTitle>
        <AlertDescription>
          {t(
            "In a conversation, a tool that changes data is recorded as a proposal rather than run. Someone reviews it before anything happens.",
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
