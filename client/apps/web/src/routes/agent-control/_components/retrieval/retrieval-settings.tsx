import { MoneyField } from "@/components/fields/money-field";
import { SwitchField } from "@/components/fields/switch-field";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  AI_RETRIEVAL_FAILED_LIST_KEY,
  updateAIRetrievalSettings,
  type AIRetrievalSettings,
  type AIRetrievalStatus,
} from "@/lib/graphql/ai-retrieval";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import {
  RETRIEVAL_SOURCE_TYPES,
  SOURCE_LABEL,
  SOURCE_SETTING,
  retrievalSettingsSchema,
  toSettingsFormValues,
  toSettingsPatch,
  type RetrievalSettingsFormValues,
} from "./retrieval-model";

const FORM_ID = "ai-retrieval-settings-form";

type RetrievalSettingsPanelProps = {
  settings: AIRetrievalSettings;
  canUpdate: boolean;
};

/**
 * What is indexed and what indexing may spend. A source turned off is still
 * found by its words; one turned on is indexed within the hour. Saving sends
 * only what changed, so it never undoes a change the indexer or another
 * person made in the meantime.
 */
export function RetrievalSettingsPanel({ settings, canUpdate }: RetrievalSettingsPanelProps) {
  const t = useT();

  return (
    <SectionPanel
      title={t("Settings")}
      help={t(
        "Indexing embeds each source's text with the model routed to the Embedding task. The budget covers indexing for the calendar month (UTC); when it is spent, indexing pauses until the next month or a higher budget.",
      )}
    >
      <SettingsForm settings={settings} canUpdate={canUpdate} />
    </SectionPanel>
  );
}

function SettingsForm({ settings, canUpdate }: RetrievalSettingsPanelProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const defaults = useMemo(() => toSettingsFormValues(settings), [settings]);
  const form = useForm<RetrievalSettingsFormValues>({
    resolver: zodResolver(retrievalSettingsSchema) as Resolver<RetrievalSettingsFormValues>,
    defaultValues: defaults,
    mode: "onChange",
  });
  const { control, handleSubmit, reset, formState } = form;

  useEffect(() => {
    reset(defaults);
  }, [defaults, reset]);

  const save = useApiMutation<
    AIRetrievalStatus,
    RetrievalSettingsFormValues,
    unknown,
    RetrievalSettingsFormValues
  >({
    mutationFn: (values) => updateAIRetrievalSettings(toSettingsPatch(values, defaults)),
    onSuccess: async (status) => {
      queryClient.setQueryData(queries.aiRetrieval.status().queryKey, status);
      await queryClient.invalidateQueries({ queryKey: [AI_RETRIEVAL_FAILED_LIST_KEY] });
      toast.success(t("Retrieval settings saved"));
    },
    form,
    resourceName: t("Retrieval settings"),
  });

  return (
    <form
      id={FORM_ID}
      onSubmit={handleSubmit((values) => save.mutate(values))}
      className="flex flex-col"
    >
      <FormGroup cols={1} className="p-3">
        {RETRIEVAL_SOURCE_TYPES.map((sourceType) => (
          <FormControl key={sourceType}>
            <SwitchField
              control={control}
              name={SOURCE_SETTING[sourceType]}
              label={t(SOURCE_LABEL[sourceType].label)}
              description={t(SOURCE_LABEL[sourceType].description)}
              outlined
              readOnly={!canUpdate}
            />
          </FormControl>
        ))}
        <FormControl>
          <MoneyField
            control={control}
            name="monthlyIndexingBudgetCents"
            label={t("Monthly indexing budget (USD)")}
            description={t(
              "Search queries are not counted here; they count against each agent's budget.",
            )}
            readOnly={!canUpdate}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="paused"
            label={t("Pause indexing")}
            description={t(
              "Nothing new is indexed and agents search by keyword only until indexing is resumed.",
            )}
            outlined
            readOnly={!canUpdate}
          />
        </FormControl>
      </FormGroup>
      <div className="border-border flex items-center justify-between gap-2 border-t px-3 py-2">
        <span className="text-muted-foreground text-xs">
          {canUpdate
            ? t("Changes apply from the indexer's next round.")
            : t("Changing these needs the right to update AI providers.")}
        </span>
        <Button
          type="submit"
          form={FORM_ID}
          size="sm"
          disabled={!canUpdate || !formState.isDirty || save.isPending}
          isLoading={save.isPending}
        >
          {t("Save settings")}
        </Button>
      </div>
    </form>
  );
}
