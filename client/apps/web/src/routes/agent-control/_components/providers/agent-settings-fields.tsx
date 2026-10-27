import { InputField } from "@/components/fields/input-field";
import {
  DEFAULT_ADVISOR_MODEL,
  type AIProviderAdvisorRefusal as AdvisorRefusal,
} from "@/types/ai-provider";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { useEffect } from "react";
import { useFormContext, useWatch, type UseFormReturn } from "react-hook-form";
import { aicFieldTrigger } from "../edit/field-trigger";
import { SwitchRow } from "../edit/setting-row";
import { advisorBlock, refusalInForce, scriptsAllowed } from "./agent-settings-model";
import { PLATFORM_LABELS } from "./platform-model";
import type { ProviderEditorValues } from "./provider-editor-model";

/**
 * Turns off an advisor or script setting the draft can no longer take, so the
 * change list shows it and script writes are never left on without scripts.
 */
export function useClearUnavailableAgentSettings(form: UseFormReturn<ProviderEditorValues>) {
  const [kind, platform, tasks, advisorEnabled, scripts, scriptWrites] = useWatch({
    control: form.control,
    name: ["kind", "platform", "tasks", "advisorEnabled", "scripts", "scriptWrites"],
  });
  const advisorAvailable = advisorBlock({ kind, platform, tasks }) === null;
  const scriptsAvailable = scriptsAllowed({ tasks });

  useEffect(() => {
    const clear = { shouldDirty: true, shouldValidate: true } as const;
    if (advisorEnabled && !advisorAvailable) form.setValue("advisorEnabled", false, clear);
    if (scripts && !scriptsAvailable) form.setValue("scripts", false, clear);
    if (scriptWrites && !(scripts && scriptsAvailable)) {
      form.setValue("scriptWrites", false, clear);
    }
  }, [advisorAvailable, advisorEnabled, form, scriptWrites, scripts, scriptsAvailable]);
}

/** Anthropic's advisor: the switch, the model it consults, and the API's refusal, if any. */
export function AdvisorFields({
  refusal,
  chatLabel,
}: {
  refusal: AdvisorRefusal | null;
  chatLabel: string;
}) {
  const t = useT();
  const { control } = useFormContext<ProviderEditorValues>();
  const [kind, platform, tasks, advisorEnabled, advisorModel, model] = useWatch({
    control,
    name: ["kind", "platform", "tasks", "advisorEnabled", "advisorModel", "model"],
  });
  const block = advisorBlock({ kind, platform, tasks });
  const refused = refusalInForce(refusal, { advisorEnabled, advisorModel, model });
  const why =
    block === "platform"
      ? t(
          "{0} does not serve the advisor; only the Claude API and Claude Platform on AWS do.",
          PLATFORM_LABELS[platform],
        )
      : block === "task"
        ? t("Only a provider that handles {0} consults an advisor.", chatLabel)
        : undefined;

  return (
    <div className="flex flex-col gap-4">
      {refused && (
        <Alert variant="warning" size="sm">
          <AlertCircleIcon />
          <AlertTitle>
            {t("The API refused {0} as the advisor for {1}", refused.advisor, refused.model)}
          </AlertTitle>
          <AlertDescription>
            <span>{refused.message}</span>
            <span>
              {t(
                "Since {0}, calls go out without the advisor. Change the advisor, the model or the platform to try again.",
                formatUnixDateTimeShort(refused.refusedAt),
              )}
            </span>
          </AlertDescription>
        </Alert>
      )}
      <SwitchRow<ProviderEditorValues>
        control={control}
        name="advisorEnabled"
        label={t("Advisor")}
        note={t(
          "On hard steps the model consults a stronger model mid-reply, on Anthropic's side.",
        )}
        disabled={block !== null}
        why={why}
      />
      {advisorEnabled && block === null && (
        <InputField<ProviderEditorValues>
          control={control}
          name="advisorModel"
          label={t("Advisor model")}
          description={t(
            "Empty consults {0}. It must be on this endpoint's model list.",
            DEFAULT_ADVISOR_MODEL,
          )}
          placeholder={DEFAULT_ADVISOR_MODEL}
          inputClassProps={aicFieldTrigger}
          layout="inline"
        />
      )}
    </div>
  );
}

/** Code mode: whether the model may answer with a read-only script, and propose writes from one. */
export function ScriptFields({ chatLabel }: { chatLabel: string }) {
  const t = useT();
  const { control } = useFormContext<ProviderEditorValues>();
  const [tasks, scripts] = useWatch({ control, name: ["tasks", "scripts"] });
  const allowed = scriptsAllowed({ tasks });

  return (
    <div className="flex flex-col">
      <SwitchRow<ProviderEditorValues>
        control={control}
        name="scripts"
        label={t("Scripts")}
        note={t(
          "Lets this provider's model answer fan-out, reconciliation and report questions with one short read-only Python script instead of many tool calls. On by default.",
        )}
        disabled={!allowed}
        why={t("Only a provider that handles {0} may answer with a script.", chatLabel)}
      />
      <SwitchRow<ProviderEditorValues>
        control={control}
        name="scriptWrites"
        label={t("Scripts may propose writes")}
        note={t(
          "Scripts may prepare changes, which are never made automatically: they're proposed as one batch a person approves, each change is checked again against the record when it's approved, and a script that read outside text cannot run anything.",
        )}
        disabled={!allowed || !scripts}
        why={t("Turn on Scripts first.")}
      />
    </div>
  );
}
