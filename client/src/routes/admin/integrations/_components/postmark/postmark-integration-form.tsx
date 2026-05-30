import trenovaLogo from "@/assets/logo.webp";
import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
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

const postmarkLogo = "/integrations/logos/postmark_all.png";
const postmarkWebhookEvents = ["Delivery", "Bounce", "SpamComplaint", "Open", "Click"];

export function PostmarkIntegrationForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...queries.integration.config("Postmark"),
    enabled: open,
  });

  const { control, reset, handleSubmit, setError } = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: {
      enabled: false,
      configuration: {
        serverToken: "",
        baseUrl: "https://api.postmarkapp.com",
        messageStream: "outbound",
        webhookToken: "",
      },
    },
  });

  const response = configQuery.data;
  const hasServerToken =
    response?.fields?.some((field) => field.key === "serverToken" && field.hasValue) ?? false;
  const webhookToken = useWatch({ control, name: "configuration.webhookToken" });
  const webhookURL = useMemo(() => buildPostmarkWebhookURL(webhookToken), [webhookToken]);

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    reset({
      enabled: response.enabled,
      configuration: {
        serverToken: "",
        baseUrl:
          response.fields.find((field) => field.key === "baseUrl")?.value ||
          "https://api.postmarkapp.com",
        messageStream:
          response.fields.find((field) => field.key === "messageStream")?.value || "outbound",
        webhookToken: response.fields.find((field) => field.key === "webhookToken")?.value || "",
      },
    });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig("Postmark", payload),
    setFormError: setError,
    resourceName: "Postmark configuration",
    onSuccess: async () => {
      toast.success("Postmark integration updated");
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config("Postmark").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.catalog().queryKey,
        }),
      ]);
    },
  });

  return (
    <div className="space-y-4">
      <PostmarkFormHeader />
      <Alert variant="info">
        <MailCheckIcon className="size-4" />
        <AlertTitle>Transactional email provider</AlertTitle>
        <AlertDescription>
          Postmark credentials are stored here. Sender profiles and purpose assignments are managed
          from Organization Email Profiles.
        </AlertDescription>
      </Alert>
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <SwitchField
              label="Enable Postmark"
              control={control}
              name="enabled"
              description="Toggle transactional email delivery for this business unit."
              outlined
            />
          </FormControl>
          <FormControl cols="full">
            <SensitiveField
              name="configuration.serverToken"
              control={control}
              label={`Server Token ${hasServerToken ? "(leave blank to keep existing token)" : ""}`}
              autoComplete="off"
              placeholder={hasServerToken ? "********" : "Postmark server token"}
              description="Used by the server for Postmark API calls."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="configuration.baseUrl"
              control={control}
              label="Base URL"
              placeholder="https://api.postmarkapp.com"
              description="Keep the default unless Postmark changes the API endpoint."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="configuration.messageStream"
              control={control}
              label="Message Stream"
              placeholder="outbound"
              description="Postmark message stream used for transactional sends."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="configuration.webhookToken"
              control={control}
              label={
                <span className="inline-flex items-center gap-1.5">
                  Webhook Token
                  <PostmarkWebhookHelpPopover webhookURL={webhookURL} />
                </span>
              }
              readOnly
              placeholder="Generated after first save"
              description="Use this token in the Postmark webhook URL path."
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

function buildPostmarkWebhookURL(webhookToken?: string) {
  if (!webhookToken || typeof window === "undefined") {
    return "";
  }

  return new URL(
    `${API_BASE_URL}/webhooks/email/postmark/${encodeURIComponent(webhookToken)}/`,
    window.location.origin,
  ).toString();
}

function PostmarkWebhookHelpPopover({ webhookURL }: { webhookURL: string }) {
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
            aria-label="Postmark webhook setup instructions"
          >
            <InfoIcon className="size-3" />
          </Button>
        }
      />
      <PopoverContent align="start" className="w-88">
        <PopoverHeader>
          <PopoverTitle>Postmark webhook setup</PopoverTitle>
          <PopoverDescription>
            Keeps email logs current after delivery, bounces, complaints, opens, and clicks.
          </PopoverDescription>
        </PopoverHeader>
        <div className="space-y-2 text-xs">
          <p className="text-muted-foreground">
            In Postmark, create a server webhook endpoint over HTTPS and use this URL:
          </p>
          <code className="block max-w-full overflow-x-auto rounded-md bg-muted px-2 py-1.5 text-foreground">
            {webhookURL || "Save once to generate the webhook URL."}
          </code>
          <div className="space-y-1">
            <p className="text-muted-foreground">Enable these Postmark triggers:</p>
            <div className="flex flex-wrap gap-1">
              {postmarkWebhookEvents.map((event) => (
                <code
                  key={event}
                  className="rounded bg-muted px-1.5 py-0.5 text-[11px] text-foreground"
                >
                  {event}
                </code>
              ))}
            </div>
          </div>
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

function PostmarkFormHeader() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
        </div>
        <LazyImage src={postmarkLogo} alt="Postmark" className="h-8 w-28 object-contain" />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">Connect with Postmark</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-xs text-muted-foreground">Create a server token and webhook in</p>
          <ExternalLink href="https://account.postmarkapp.com/servers" className="text-xs">
            Postmark.
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
