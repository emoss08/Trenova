const openAILogo = "/integrations/logos/openai_logo.svg";
import { useT } from "@trenova/shared/i18n/use-t";
import trenovaLogo from "@/assets/logo.webp";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { SparklesIcon } from "lucide-react";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

export function OpenAIIntegrationForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();

  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...queries.integration.config("OpenAI"),
    enabled: open,
  });

  const form = useForm({
    defaultValues: {
      enabled: false,
      configuration: {
        apiKey: "",
      },
    },
  });
  const { control, reset, handleSubmit } = form;

  const response = configQuery.data;
  const hasApiKey = response?.fields?.some((f) => f.key === "apiKey" && f.hasValue) ?? false;

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    reset({
      enabled: response.enabled,
      configuration: {
        apiKey: "",
      },
    });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig("OpenAI", payload),
    form,
    resourceName: "OpenAI configuration",
    onSuccess: async () => {
      toast.success(t("OpenAI integration updated"));
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config("OpenAI").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.catalog().queryKey,
        }),
      ]);
    },
  });

  return (
    <div className="space-y-4">
      <OpenAIFormHeader />
      <Alert variant="info">
        <SparklesIcon className="size-4" />
        <AlertTitle>{t("Document AI requires two layers")}</AlertTitle>
        <AlertDescription>
          {t("This integration stores the organization OpenAI credential. AI-assisted classification and extraction are still controlled separately in Document Controls.")}
        </AlertDescription>
      </Alert>
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <SwitchField
              label={t("Enable OpenAI")}
              control={control}
              name="enabled"
              description={t("Toggle AI provider availability for this business unit.")}
              outlined
            />
          </FormControl>
          <FormControl cols="full">
            <SensitiveField
              name="configuration.apiKey"
              control={control}
              label={`API Key ${hasApiKey ? "(leave blank to keep existing key)" : ""}`}
              autoComplete="off"
              placeholder={hasApiKey ? "********" : "Enter your OpenAI API key"}
              description={t("Used for AI-assisted document classification and structured extraction.")}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter className="flex flex-row items-center sm:justify-between">
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button
            size="sm"
            type="submit"
            isLoading={saveMutation.isPending}
            loadingText={t("Saving...")}
            disabled={configQuery.isLoading}
          >
            {t("Save Changes")}
          </Button>
        </DialogFooter>
      </Form>
    </div>
  );
}

function OpenAIFormHeader() {
  const t = useT();

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
        </div>
        <LazyImage src={openAILogo} alt={t("OpenAI Logo")} className="size-8" />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">{t("Connect with OpenAI")}</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-muted-foreground text-xs">{t("Create an API key in the")}</p>
          <ExternalLink href="https://platform.openai.com/api-keys" className="text-xs">
            {t("OpenAI dashboard.")}
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
