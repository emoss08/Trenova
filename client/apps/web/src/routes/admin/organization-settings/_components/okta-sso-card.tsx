import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { OktaLogo } from "@/components/logos/okta";
import { FormSaveDock } from "@/components/form-save-dock";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Separator } from "@trenova/shared/components/ui/separator";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import { type OktaSSOConfig } from "@trenova/shared/types/organization";
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
            className="text-muted-foreground hover:bg-accent hover:text-foreground flex size-7 items-center justify-center rounded-md transition-colors"
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
      <p className="text-muted-foreground text-xs">{description}</p>
    </div>
  );
}

export function OktaSSOCard({ organizationId }: { organizationId: string }) {
  const t = useT();

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

  const { control, handleSubmit, reset, setValue } = form;
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
    form,
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
      toast.success(t("Okta SSO settings updated"));
    },
  });

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div className="border-border bg-background rounded-xl border transition-shadow has-[[data-state=open]]:shadow-sm">
        <CollapsibleTrigger
          render={
            <button
              type="button"
              className="hover:bg-muted/30 flex w-full items-center gap-4 rounded-xl px-5 py-4 text-left transition-colors"
            />
          }
        >
          <div className="border-border bg-background flex size-10 shrink-0 items-center justify-center rounded-lg border shadow-xs">
            <OktaLogo className="h-5 w-auto" />
          </div>
          <div className="min-w-0 flex-1">
            <span className="text-sm font-semibold tracking-tight">{t("Okta")}</span>
            <p className="text-muted-foreground mt-0.5 text-xs">
              {t("{0} · OpenID Connect", enabled ? t("Active") : t("Not configured"))}
            </p>
          </div>
          <ChevronRightIcon
            className={cn(
              "text-muted-foreground size-4 shrink-0 transition-transform duration-200",
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
                        {t("To configure SSO, create an OIDC application in the")}{" "}
                        <a
                          href="https://login.okta.com/"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="font-medium underline underline-offset-2"
                        >
                          {t("Okta Admin Console")}
                        </a>
                        {t(
                          ", copy the redirect URL below into the app's sign-in redirect URIs, then paste the credentials here.",
                        )}
                      </p>
                    </AlertDescription>
                  </Alert>

                  <Separator />

                  <div className="space-y-3">
                    <SectionHeader
                      title={t("Authentication Policy")}
                      description={t("Control how users authenticate to this tenant.")}
                    />
                    <FormGroup cols={1}>
                      <FormControl cols="full">
                        <SwitchField
                          control={control}
                          name="enabled"
                          label={t("Enable Okta sign-in")}
                          description={t(
                            'Allow users to sign in with a "Continue with Okta" button.',
                          )}
                          outlined
                        />
                      </FormControl>
                      {enabled && (
                        <FormControl cols="full">
                          <SwitchField
                            control={control}
                            name="enforceSso"
                            label={t("Require Okta SSO")}
                            description={t(
                              "Disable password login and require all users to sign in with Okta.",
                            )}
                            outlined
                            warning={{
                              show: Boolean(enforceSso),
                              message: t("All users will be required to sign in with Okta."),
                            }}
                          />
                        </FormControl>
                      )}
                    </FormGroup>
                    {enabled && enforceSso && (
                      <Alert variant="warning">
                        <AlertTriangleIcon />
                        <AlertTitle>{t("Password login will be disabled")}</AlertTitle>
                        <AlertDescription>
                          {t(
                            "Users without an Okta account linked to an allowed domain will be locked out. Ensure all users have Okta accounts before enabling this.",
                          )}
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>

                  {enabled && (
                    <>
                      <Separator />

                      <div className="space-y-3">
                        <SectionHeader
                          title={t("Service Provider")}
                          description={t(
                            "Copy this value into your Okta application configuration.",
                          )}
                        />
                        <Alert variant="info">
                          <LinkIcon />
                          <AlertDescription>
                            {t(
                              "Add this redirect URL to your Okta app under Sign-in redirect URIs.",
                            )}
                          </AlertDescription>
                        </Alert>
                        <CopyableInput
                          value={redirectUrl}
                          label={t("Redirect URL (OAuth Callback)")}
                        />
                      </div>

                      <Separator />

                      <div className="space-y-3">
                        <SectionHeader
                          title={t("Identity Provider")}
                          description={t("Paste these values from your Okta application settings.")}
                        />
                        <FormGroup cols={1}>
                          <FormControl cols="full">
                            <InputField
                              control={control}
                              name="issuerUrl"
                              label={t("Okta Domain")}
                              placeholder="https://your-domain.okta.com"
                              rules={{ required: enabled }}
                            />
                          </FormControl>
                          <FormControl cols="full">
                            <InputField
                              control={control}
                              name="clientId"
                              label={t("Client ID")}
                              placeholder={t("0oa...")}
                              rules={{ required: enabled }}
                            />
                          </FormControl>
                          <FormControl cols="full">
                            <SensitiveField
                              control={control}
                              name="clientSecret"
                              label={t("Client Secret")}
                              placeholder={t("Paste a new client secret")}
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
                              label={t("Scopes")}
                              placeholder={t("openid, profile, email")}
                              description={t("Comma-separated list of OIDC scopes.")}
                            />
                          </FormControl>
                        </FormGroup>
                      </div>

                      <Separator />

                      <div className="space-y-3">
                        <SectionHeader
                          title={t("Domain Restrictions")}
                          description={t("Limit which email domains can sign in with Okta.")}
                        />
                        <FormGroup cols={1}>
                          <FormControl cols="full">
                            <InputField
                              control={control}
                              name="allowedDomainsText"
                              label={t("Allowed Email Domains")}
                              placeholder={t("company.com, contractor.com")}
                              description={t(
                                "Comma-separated list. Leave blank to allow all Okta account domains.",
                              )}
                            />
                          </FormControl>
                        </FormGroup>
                      </div>

                      <Separator />

                      <div className="space-y-3">
                        <SectionHeader
                          title={t("Tenant Login URL")}
                          description={t("Share this URL with your users for Okta SSO sign-in.")}
                        />
                        <CopyableInput value={tenantLoginUrl} label={t("Login URL")} />
                        <p className="text-muted-foreground text-xs">
                          {t("Replace")}{" "}
                          <code className="bg-muted rounded px-1 py-0.5 font-mono text-[11px]">
                            {t("{loginSlug}")}
                          </code>{" "}
                          {t("with your organization's login slug from General settings.")}
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
