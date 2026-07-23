import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { OktaLogo } from "@/components/logos/okta";
import { FormSaveDock } from "@/components/form-save-dock";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { type OktaSSOConfig } from "@/types/organization";
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
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

type OktaSSOFormValues = OktaSSOConfig & {
  allowedDomainsText: string;
  scopesText: string;
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

export function OktaSSOCard({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const redirectUrl =
    typeof window !== "undefined" ? `${buildAPIOrigin()}/api/v1/auth/sso/callback/Okta` : "";
  const tenantLoginUrl =
    typeof window !== "undefined" ? `${window.location.origin}/login/{loginSlug}` : "";

  const configQuery = useQuery({
    ...queries.organization.oktaSSO(organizationId),
    enabled: Boolean(organizationId),
  });

  const form = useForm<OktaSSOFormValues>({
    defaultValues: {
      organizationId,
      enabled: false,
      enforceSso: false,
      issuerUrl: "",
      clientId: "",
      clientSecret: "",
      redirectUrl,
      scopes: ["openid", "profile", "email"],
      allowedDomains: [],
      allowedDomainsText: "",
      scopesText: "openid, profile, email",
      secretConfigured: false,
    },
  });

  const { control, handleSubmit, reset, setError, setValue } = form;
  const enabled = useWatch({ control, name: "enabled" });
  const enforceSso = useWatch({ control, name: "enforceSso" });

  if (!enabled && enforceSso) {
    setValue("enforceSso", false);
  }

  useEffect(() => {
    if (configQuery.data) {
      reset({
        ...configQuery.data,
        clientSecret: "",
        allowedDomainsText: configQuery.data.allowedDomains.join(", "),
        scopesText: configQuery.data.scopes.join(", "),
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
    mutationFn: (values: OktaSSOFormValues) =>
      apiService.organizationService.upsertOktaSSOConfig(organizationId, {
        ...values,
        allowedDomains: parseCommaSeparated(values.allowedDomainsText),
        scopes: parseCommaSeparated(values.scopesText),
        redirectUrl,
      }),
    setFormError: setError,
    resourceName: "Okta SSO",
    onSuccess: async (data) => {
      reset({
        ...data,
        clientSecret: "",
        allowedDomainsText: data.allowedDomains.join(", "),
        scopesText: data.scopes.join(", "),
        redirectUrl: data.redirectUrl || redirectUrl,
      });
      await queryClient.invalidateQueries({
        queryKey: queries.organization.oktaSSO(organizationId).queryKey,
      });
      toast.success("Okta SSO settings updated");
    },
  });

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div className="rounded-xl border border-border bg-background transition-shadow has-[[data-state=open]]:shadow-sm">
        <CollapsibleTrigger
          render={
            <button
              type="button"
              className="flex w-full items-center gap-4 rounded-xl px-5 py-4 text-left transition-colors hover:bg-muted/30"
            />
          }
        >
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg border border-border bg-background shadow-xs">
            <OktaLogo className="h-5 w-auto" />
          </div>
          <div className="min-w-0 flex-1">
            <span className="text-sm font-semibold tracking-tight">Okta</span>
            <p className="mt-0.5 text-xs text-muted-foreground">
              {enabled ? "Active" : "Not configured"} &middot; OpenID Connect
            </p>
          </div>
          <ChevronRightIcon
            className={cn(
              "size-4 shrink-0 text-muted-foreground transition-transform duration-200",
              open && "rotate-90",
            )}
          />
        </CollapsibleTrigger>

        <CollapsibleContent>
          <Separator />
          <div className="px-5 py-5">
            <FormProvider {...form}>
            <Form onSubmit={handleSubmit((values) => mutation.mutate(values))}>
              <div className="space-y-6">
                <Alert variant="info">
                  <InfoIcon />
                  <AlertDescription>
                    <p>
                      To configure SSO, create an OIDC application in the{" "}
                      <a
                        href="https://login.okta.com/"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-medium underline underline-offset-2"
                      >
                        Okta Admin Console
                      </a>
                      , copy the redirect URL below into the app&apos;s sign-in redirect URIs, then
                      paste the credentials here.
                    </p>
                  </AlertDescription>
                </Alert>

                <Separator />

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
                        label="Enable Okta sign-in"
                        description='Allow users to sign in with a "Continue with Okta" button.'
                        outlined
                      />
                    </FormControl>
                    {enabled && (
                      <FormControl cols="full">
                        <SwitchField
                          control={control}
                          name="enforceSso"
                          label="Require Okta SSO"
                          description="Disable password login and require all users to sign in with Okta."
                          outlined
                          warning={{
                            show: Boolean(enforceSso),
                            message: "All users will be required to sign in with Okta.",
                          }}
                        />
                      </FormControl>
                    )}
                  </FormGroup>
                  {enabled && enforceSso && (
                    <Alert variant="warning">
                      <AlertTriangleIcon />
                      <AlertTitle>Password login will be disabled</AlertTitle>
                      <AlertDescription>
                        Users without an Okta account linked to an allowed domain will be locked out.
                        Ensure all users have Okta accounts before enabling this.
                      </AlertDescription>
                    </Alert>
                  )}
                </div>

                {enabled && (
                  <>
                    <Separator />

                    <div className="space-y-3">
                      <SectionHeader
                        title="Service Provider"
                        description="Copy this value into your Okta application configuration."
                      />
                      <Alert variant="info">
                        <LinkIcon />
                        <AlertDescription>
                          Add this redirect URL to your Okta app under Sign-in redirect URIs.
                        </AlertDescription>
                      </Alert>
                      <CopyableInput value={redirectUrl} label="Redirect URL (OAuth Callback)" />
                    </div>

                    <Separator />

                    <div className="space-y-3">
                      <SectionHeader
                        title="Identity Provider"
                        description="Paste these values from your Okta application settings."
                      />
                      <FormGroup cols={1}>
                        <FormControl cols="full">
                          <InputField
                            control={control}
                            name="issuerUrl"
                            label="Okta Domain"
                            placeholder="https://your-domain.okta.com"
                            rules={{ required: enabled }}
                          />
                        </FormControl>
                        <FormControl cols="full">
                          <InputField
                            control={control}
                            name="clientId"
                            label="Client ID"
                            placeholder="0oa..."
                            rules={{ required: enabled }}
                          />
                        </FormControl>
                        <FormControl cols="full">
                          <SensitiveField
                            control={control}
                            name="clientSecret"
                            label="Client Secret"
                            placeholder="Paste a new client secret"
                            description={
                              configQuery.data?.secretConfigured
                                ? "A secret is already stored. Leave blank to keep it."
                                : "Required the first time you configure SSO."
                            }
                          />
                        </FormControl>
                        <FormControl cols="full">
                          <InputField
                            control={control}
                            name="scopesText"
                            label="Scopes"
                            placeholder="openid, profile, email"
                            description="Comma-separated list of OIDC scopes."
                          />
                        </FormControl>
                      </FormGroup>
                    </div>

                    <Separator />

                    <div className="space-y-3">
                      <SectionHeader
                        title="Domain Restrictions"
                        description="Limit which email domains can sign in with Okta."
                      />
                      <FormGroup cols={1}>
                        <FormControl cols="full">
                          <InputField
                            control={control}
                            name="allowedDomainsText"
                            label="Allowed Email Domains"
                            placeholder="company.com, contractor.com"
                            description="Comma-separated list. Leave blank to allow all Okta account domains."
                          />
                        </FormControl>
                      </FormGroup>
                    </div>

                    <Separator />

                    <div className="space-y-3">
                      <SectionHeader
                        title="Tenant Login URL"
                        description="Share this URL with your users for Okta SSO sign-in."
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

              </div>
              <FormSaveDock />
            </Form>
            </FormProvider>
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

function parseCommaSeparated(values: string) {
  return values
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);
}
