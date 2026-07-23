import trenovaLogo from "@/assets/logo.webp";
import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { useTheme } from "@/components/theme-provider";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { API_BASE_URL } from "@/lib/constants";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, CopyIcon, InfoIcon, MailCheckIcon } from "lucide-react";
import { useEffect, useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

const resendLogoDark = "/integrations/logos/resend_logo_dark.svg";
const resendLogoLight = "/integrations/logos/resend_logo_light.svg";
const resendWebhookEvents = [
  "email.delivered",
  "email.bounced",
  "email.complained",
  "email.opened",
  "email.clicked",
];

export function ResendIntegrationForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...queries.integration.config("Resend"),
    enabled: open,
  });

  const { control, reset, handleSubmit, setError } = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: {
      enabled: false,
      configuration: {
        apiKey: "",
        baseUrl: "https://api.resend.com",
        webhookSigningSecret: "",
        webhookToken: "",
      },
    },
  });

  const response = configQuery.data;
  const hasApiKey = response?.fields?.some((f) => f.key === "apiKey" && f.hasValue) ?? false;
  const hasWebhookSecret =
    response?.fields?.some((f) => f.key === "webhookSigningSecret" && f.hasValue) ?? false;
  const webhookToken = useWatch({ control, name: "configuration.webhookToken" });
  const webhookURL = useMemo(() => buildResendWebhookURL(webhookToken), [webhookToken]);

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    reset({
      enabled: response.enabled,
      configuration: {
        apiKey: "",
        baseUrl:
          response.fields.find((field) => field.key === "baseUrl")?.value ||
          "https://api.resend.com",
        webhookSigningSecret: "",
        webhookToken: response.fields.find((field) => field.key === "webhookToken")?.value || "",
      },
    });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig("Resend", payload),
    setFormError: setError,
    resourceName: "Resend configuration",
    onSuccess: async () => {
      toast.success("Resend integration updated");
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config("Resend").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.catalog().queryKey,
        }),
      ]);
    },
  });

  return (
    <div className="space-y-4">
      <ResendFormHeader />
      <Alert variant="info">
        <MailCheckIcon className="size-4" />
        <AlertTitle>Transactional email provider</AlertTitle>
        <AlertDescription>
          Resend credentials are stored here. Sender profiles and purpose assignments are managed
          from Organization Email Profiles.
        </AlertDescription>
      </Alert>
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <SwitchField
              label="Enable Resend"
              control={control}
              name="enabled"
              description="Toggle transactional email delivery for this business unit."
              outlined
            />
          </FormControl>
          <FormControl cols="full">
            <SensitiveField
              name="configuration.apiKey"
              control={control}
              label={`API Key ${hasApiKey ? "(leave blank to keep existing key)" : ""}`}
              autoComplete="off"
              placeholder={hasApiKey ? "********" : "re_..."}
              description="Used by the server for Resend REST API calls."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="configuration.baseUrl"
              control={control}
              label="Base URL"
              placeholder="https://api.resend.com"
              description="Keep the default unless Resend changes the API endpoint."
            />
          </FormControl>
          <FormControl cols="full">
            <SensitiveField
              name="configuration.webhookSigningSecret"
              control={control}
              label={
                <span className="inline-flex items-center gap-1.5">
                  Webhook Signing Secret
                  {hasWebhookSecret ? " (leave blank to keep existing secret)" : ""}
                  <ResendWebhookHelpPopover webhookURL={webhookURL} />
                </span>
              }
              autoComplete="off"
              placeholder={hasWebhookSecret ? "********" : "whsec_..."}
              description="Svix signing secret from the Resend webhook endpoint."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="configuration.webhookToken"
              control={control}
              label="Webhook Token"
              readOnly
              placeholder="Generated after first save"
              description="Use this token in the Resend webhook URL path."
            />
          </FormControl>
        </FormGroup>
        <DialogFooter className="flex flex-row items-center sm:justify-between">
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            type="submit"
            isLoading={saveMutation.isPending}
            loadingText="Saving..."
            disabled={configQuery.isLoading}
          >
            Save Changes
          </Button>
        </DialogFooter>
      </Form>
    </div>
  );
}

function buildResendWebhookURL(webhookToken?: string) {
  if (!webhookToken || typeof window === "undefined") {
    return "";
  }

  return new URL(
    `${API_BASE_URL}/webhooks/email/resend/${encodeURIComponent(webhookToken)}/`,
    window.location.origin,
  ).toString();
}

function ResendWebhookHelpPopover({ webhookURL }: { webhookURL: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-3.5 p-0 text-muted-foreground hover:bg-transparent hover:text-foreground"
            aria-label="Resend webhook setup instructions"
          >
            <InfoIcon className="size-3" />
          </Button>
        }
      />
      <PopoverContent align="start" className="w-88">
        <PopoverHeader>
          <PopoverTitle>Resend webhook setup</PopoverTitle>
          <PopoverDescription>
            Keeps email logs current after delivery, bounces, complaints, and other provider events.
          </PopoverDescription>
        </PopoverHeader>
        <div className="space-y-2 text-xs">
          <p className="text-muted-foreground">
            In Resend, create a webhook endpoint and use this URL:
          </p>
          <code className="block max-w-full overflow-x-auto rounded-md bg-muted px-2 py-1.5 text-foreground">
            {webhookURL || "Save once to generate the webhook URL."}
          </code>
          <div className="space-y-1">
            <p className="text-muted-foreground">Listen for these Resend events:</p>
            <div className="flex flex-wrap gap-1">
              {resendWebhookEvents.map((event) => (
                <code
                  key={event}
                  className="rounded bg-muted px-1.5 py-0.5 text-[11px] text-foreground"
                >
                  {event}
                </code>
              ))}
            </div>
          </div>
          <p className="text-muted-foreground">
            Paste Resend&apos;s signing secret here to verify incoming webhook calls. The integration
            can be saved before the webhook is configured.
          </p>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="w-full"
            disabled={!webhookURL}
            onClick={() => void copy(webhookURL, { withToast: true })}
          >
            {isCopied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
            Copy webhook URL
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function ResendFormHeader() {
  const { theme } = useTheme();
  const resendLogo = theme === "dark" ? resendLogoDark : resendLogoLight;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
        </div>
        <LazyImage src={resendLogo} alt="Resend" className="h-8 w-24 object-contain" />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">Connect with Resend</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-xs text-muted-foreground">Create an API key and webhook in</p>
          <ExternalLink href="https://resend.com/api-keys" className="text-xs">
            Resend.
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
