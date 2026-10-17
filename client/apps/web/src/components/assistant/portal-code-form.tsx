import { InputField } from "@/components/fields/input-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { portalCodePrompt } from "@/lib/portal-run";
import { agentExtension } from "@/lib/queries/agent-extension";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import {
  portalSignInCodeFormSchema,
  type PortalRun,
  type PortalSignInCodeFormValues,
} from "@/types/agent-extension";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { FormProvider, useForm, type Resolver } from "react-hook-form";

export type PortalCodeFormProps = {
  runId: string;
  /** How the portal asked for the code, in the browser's words. */
  prompt: string;
  /** When the browser stops waiting, already written for where the form sits. */
  deadline: string;
  /** Codes already entered for the task; one or more means the portal asked again. */
  attempts: number;
  className?: string;
};

/**
 * Where a person enters the sign-in code a portal texted or emailed them while a browser
 * waits on its code prompt. The code goes straight to that browser and never into the
 * conversation, so no model sees it.
 */
export function PortalCodeForm({
  runId,
  prompt,
  deadline,
  attempts,
  className,
}: PortalCodeFormProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<PortalSignInCodeFormValues>({
    resolver: zodResolver(portalSignInCodeFormSchema) as Resolver<PortalSignInCodeFormValues>,
    defaultValues: { code: "" },
  });
  const { control, handleSubmit, reset } = form;

  const { mutateAsync, isPending } = useApiMutation<
    PortalRun,
    PortalSignInCodeFormValues,
    unknown,
    PortalSignInCodeFormValues
  >({
    form,
    resourceName: t("Sign-in code"),
    mutationFn: (values) =>
      apiService.agentExtensionService.sendPortalSignInCode(runId, values.code),
    onSuccess: async () => {
      reset({ code: "" });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: assistant.waits._def }),
        queryClient.invalidateQueries({ queryKey: agentExtension.portalRuns._def }),
      ]);
    },
  });

  const said = portalCodePrompt(prompt);

  return (
    <FormProvider {...form}>
      <Form
        className={cn("flex flex-col gap-1.5", className)}
        aria-label={t("Enter the sign-in code")}
        onSubmit={(submitEvent) => {
          submitEvent.preventDefault();
          submitEvent.stopPropagation();
          void handleSubmit((values) => mutateAsync(values))(submitEvent);
        }}
      >
        <p className="text-muted-foreground line-clamp-3 text-xs" title={said || undefined}>
          <span>
            {attempts > 0
              ? t("The portal asked for another code.")
              : t("The portal sent you a sign-in code.")}
          </span>
          {said && <span> {said}</span>}
        </p>
        <div className="flex items-end gap-2">
          <InputField<PortalSignInCodeFormValues>
            control={control}
            name="code"
            label={t("Sign-in code")}
            placeholder="123456"
            autoComplete="one-time-code"
            spellCheck={false}
            maxLength={20}
            className="min-w-0 flex-1"
            rules={{ required: true }}
          />
          <Button type="submit" size="sm" isLoading={isPending} loadingText={t("Sending")}>
            {t("Send code")}
          </Button>
        </div>
        <p className="text-muted-foreground text-xs">
          {t("Enter it by {0}. It goes only to the portal, never into the conversation.", deadline)}
        </p>
      </Form>
    </FormProvider>
  );
}
