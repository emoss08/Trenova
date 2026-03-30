import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { MicrosoftLogo } from "@/components/logos/microsoft";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { API_BASE_URL } from "@/lib/constants";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import { type MicrosoftSSOConfig } from "@/types/organization";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  ChevronRightIcon,
  CopyIcon,
  InfoIcon,
  LinkIcon,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

type MicrosoftSSOFormValues = MicrosoftSSOConfig & {
  allowedDomainsText: string;
};

function buildAPIOrigin() {
  if (typeof window === "undefined") {
    return "";
  }

  const apiURL = new URL(API_BASE_URL, window.location.origin);
  return apiURL.origin;
}

function CopyableInput({ value, label }: { value: string; label: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="space-y-1.5">
      <Label>{label}</Label>
      <Input
        readOnly
        value={value}
        className="bg-muted/50 font-mono text-xs"
        rightElement={
          <button
            type="button"
            onClick={() => copy(value, { timeout: 3000, withToast: true })}
            className="flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            {isCopied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
          </button>
        }
      />
    </div>
  );
}

function SectionHeader({ title, description }: { title: string; description: string }) {
  return (
    <div>
      <h4 className="text-sm font-medium">{title}</h4>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  );
}

export function MicrosoftSSOCard({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const redirectUrl =
    typeof window !== "undefined" ? `${buildAPIOrigin()}/api/v1/auth/microsoft/callback` : "";
  const tenantLoginUrl =
    typeof window !== "undefined" ? `${window.location.origin}/login/{loginSlug}` : "";

  const configQuery = useQuery({
    ...queries.organization.microsoftSSO(organizationId),
    enabled: Boolean(organizationId),
  });

  const form = useForm<MicrosoftSSOFormValues>({
    defaultValues: {
      organizationId,
      enabled: false,
      enforceSso: false,
      tenantId: "",
      clientId: "",
      clientSecret: "",
      redirectUrl,
      allowedDomains: [],
      allowedDomainsText: "",
      secretConfigured: false,
    },
  });

  const { control, handleSubmit, reset, setError } = form;
  const enabled = useWatch({ control, name: "enabled" });
  const enforceSso = useWatch({ control, name: "enforceSso" });

  useEffect(() => {
    if (configQuery.data) {
      reset({
        ...configQuery.data,
        clientSecret: "",
        allowedDomainsText: configQuery.data.allowedDomains.join(", "),
        redirectUrl: configQuery.data.redirectUrl || redirectUrl,
      });
      return;
    }

    reset((current) => ({
      ...current,
      organizationId,
      redirectUrl,
    }));
  }, [configQuery.data, organizationId, redirectUrl, reset]);

  const mutation = useApiMutation({
    mutationFn: (values: MicrosoftSSOFormValues) =>
      apiService.organizationService.upsertMicrosoftSSOConfig(organizationId, {
        ...values,
        allowedDomains: parseAllowedDomains(values.allowedDomainsText),
        redirectUrl,
      }),
    setFormError: setError,
    resourceName: "Microsoft SSO",
    onSuccess: async (data) => {
      reset({
        ...data,
        clientSecret: "",
        allowedDomainsText: data.allowedDomains.join(", "),
        redirectUrl: data.redirectUrl || redirectUrl,
      });
      await queryClient.invalidateQueries({
        queryKey: queries.organization.microsoftSSO(organizationId).queryKey,
      });
      toast.success("Microsoft SSO settings updated");
    },
  });

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div className="rounded-xl border border-border bg-background transition-shadow has-[[data-state=open]]:shadow-sm">
        {/* Provider Header / Trigger */}
        <CollapsibleTrigger
          render={
            <button
              type="button"
              className="flex w-full items-center gap-4 rounded-xl px-5 py-4 text-left transition-colors hover:bg-muted/30"
            />
          }
        >
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg border border-border bg-background shadow-xs">
            <MicrosoftLogo className="size-6" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2.5">
              <span className="text-sm font-semibold tracking-tight">Microsoft Entra ID</span>
              <Badge variant={enabled ? "active" : "outline"}>
                {enabled ? "Active" : "Not configured"}
              </Badge>
            </div>
            <p className="mt-0.5 text-xs text-muted-foreground">
              OpenID Connect &middot; Single Sign-On
            </p>
          </div>
          <ChevronRightIcon
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform duration-200",
              open && "rotate-90",
            )}
          />
        </CollapsibleTrigger>

        {/* Collapsible Configuration Panel */}
        <CollapsibleContent>
          <Separator />
          <div className="px-5 py-5">
            <Form onSubmit={handleSubmit((values) => mutation.mutate(values))}>
              <div className="space-y-6">
                {/* Setup Guide */}
                <Alert variant="info">
                  <InfoIcon />
                  <AlertDescription>
                    <p>
                      To configure SSO, register an app in{" "}
                      <a
                        href="https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-medium underline underline-offset-2"
                      >
                        Microsoft Entra ID
                      </a>
                      , copy the redirect URL below into the app&apos;s authentication settings,
                      then paste the credentials here.
                    </p>
                  </AlertDescription>
                </Alert>

                <Separator />

                {/* Authentication Policy */}
                <div className="space-y-3">
                  <SectionHeader
                    title="Authentication Policy"
                    description="Control how users authenticate to this tenant."
                  />
                  <FormGroup cols={1}>
                    <FormControl cols="full">
                      <SwitchField
                        control={control}
                        name="enabled"
                        label="Enable Microsoft sign-in"
                        description='Allow users to sign in with a "Continue with Microsoft" button.'
                        outlined
                      />
                    </FormControl>
                    <FormControl cols="full">
                      <SwitchField
                        control={control}
                        name="enforceSso"
                        label="Require Microsoft SSO"
                        description="Disable password login and require all users to sign in with Microsoft."
                        outlined
                        warning={{
                          show: Boolean(enforceSso),
                          message: "All users will be required to sign in with Microsoft.",
                        }}
                      />
                    </FormControl>
                  </FormGroup>
                  {enforceSso && (
                    <Alert variant="warning">
                      <AlertTriangleIcon />
                      <AlertTitle>Password login will be disabled</AlertTitle>
                      <AlertDescription>
                        Users without a Microsoft account linked to an allowed domain will be locked
                        out. Ensure all users have Microsoft accounts before enabling this.
                      </AlertDescription>
                    </Alert>
                  )}
                </div>

                {enabled && (
                  <>
                    <Separator />

                    {/* Service Provider */}
                    <div className="space-y-3">
                      <SectionHeader
                        title="Service Provider"
                        description="Copy this value into your Microsoft app registration."
                      />
                      <Alert variant="info">
                        <LinkIcon />
                        <AlertDescription>
                          Add this redirect URL to your Microsoft app under Authentication &gt;
                          Redirect URIs.
                        </AlertDescription>
                      </Alert>
                      <CopyableInput value={redirectUrl} label="Redirect URL (OAuth Callback)" />
                    </div>

                    <Separator />

                    {/* Identity Provider */}
                    <div className="space-y-3">
                      <SectionHeader
                        title="Identity Provider"
                        description="Paste these values from your Microsoft Entra ID app registration."
                      />
                      <FormGroup cols={1}>
                        <FormControl cols="full">
                          <InputField
                            control={control}
                            name="tenantId"
                            label="Directory (Tenant) ID"
                            placeholder="00000000-0000-0000-0000-000000000000"
                            rules={{ required: enabled }}
                          />
                        </FormControl>
                        <FormControl cols="full">
                          <InputField
                            control={control}
                            name="clientId"
                            label="Application (Client) ID"
                            placeholder="00000000-0000-0000-0000-000000000000"
                            rules={{ required: enabled }}
                          />
                        </FormControl>
                        <FormControl cols="full">
                          <SensitiveField
                            control={control}
                            name="clientSecret"
                            label="Client Secret Value"
                            placeholder="Paste a new client secret"
                            description={
                              configQuery.data?.secretConfigured
                                ? "A secret is already stored. Leave blank to keep it."
                                : "Required the first time you configure SSO."
                            }
                          />
                        </FormControl>
                      </FormGroup>
                    </div>

                    <Separator />

                    {/* Domain Restrictions */}
                    <div className="space-y-3">
                      <SectionHeader
                        title="Domain Restrictions"
                        description="Limit which email domains can sign in with Microsoft."
                      />
                      <FormGroup cols={1}>
                        <FormControl cols="full">
                          <InputField
                            control={control}
                            name="allowedDomainsText"
                            label="Allowed Email Domains"
                            placeholder="company.com, contractor.com"
                            description="Comma-separated list. Leave blank to allow all Microsoft account domains."
                          />
                        </FormControl>
                      </FormGroup>
                    </div>

                    <Separator />

                    {/* Tenant Login URL */}
                    <div className="space-y-3">
                      <SectionHeader
                        title="Tenant Login URL"
                        description="Share this URL with your users for Microsoft SSO sign-in."
                      />
                      <CopyableInput value={tenantLoginUrl} label="Login URL" />
                      <p className="text-xs text-muted-foreground">
                        Replace{" "}
                        <code className="rounded bg-muted px-1 py-0.5 font-mono text-[11px]">
                          {"{loginSlug}"}
                        </code>{" "}
                        with your organization&apos;s login slug from General settings.
                      </p>
                    </div>
                  </>
                )}

                <Separator />

                <Button
                  type="submit"
                  className="w-full sm:w-auto"
                  isLoading={mutation.isPending}
                  loadingText="Saving..."
                >
                  Save Configuration
                </Button>
              </div>
            </Form>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

function parseAllowedDomains(values: string) {
  return values
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);
}
