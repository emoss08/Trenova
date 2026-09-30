import { CheckboxField } from "@/components/fields/checkbox-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { ExternalLink } from "@/components/link";
import { SectionPanel } from "@/components/section-panel";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { accountingWebhookUrl } from "@/lib/accounting-sync";
import type {
  AccountingAppSettings,
  AccountingConnection,
  AccountingProviderProfile,
} from "@/lib/graphql/accounting-sync";
import type {
  AccountingAppEnvironment,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { toSentenceFragment } from "@trenova/shared/lib/utils";
import { CheckIcon, CopyIcon, KeyRoundIcon } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import type { AccountingVendor } from "./accounting-vendors";
import {
  accountingAppFormDefaults,
  accountingAppFormSchema,
  type AccountingAppFormValues,
} from "./accounting-app-schema";
import { useAccountingAppActions } from "./use-accounting-connection";

type AccountingAppKeysProps = {
  vendor: AccountingVendor;
  app: AccountingAppSettings;
  profile: AccountingProviderProfile;
  connection: AccountingConnection | null;
  canManage: boolean;
};

export function AccountingAppKeys({
  vendor,
  app,
  profile,
  connection,
  canManage,
}: AccountingAppKeysProps) {
  const t = useT();
  const appName = t(profile.appName);
  const webhookKeyLabel = t(profile.webhookKeyLabel);
  const choosesEnvironment = profile.environments.length > 1;
  const [editing, setEditing] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const { remove } = useAccountingAppActions(vendor);

  const live = connection !== null && isLive(connection);
  const connectedThroughTenant = live && connection.appSource === "Tenant";
  const connectedThroughInstance = live && connection.appSource === "Instance";
  const tenantApp = app.tenantApp;
  const noApp = !app.instanceAppAvailable && !tenantApp;
  const showForm = canManage && (editing || noApp);

  const environmentLabels: Record<AccountingAppEnvironment, string> = {
    Sandbox: t("Sandbox"),
    Production: t("Production"),
  };

  return (
    <SectionPanel
      title={appName}
      icon={<KeyRoundIcon />}
      hint={
        app.activeSource === "Tenant"
          ? t("Your own app")
          : app.activeSource === "Instance"
            ? t("This server's app")
            : t("Not set up")
      }
    >
      <div className="space-y-3 p-3">
        {noApp ? (
          <p className="text-foreground-muted text-sm">
            {t(
              "This Trenova server has no {0} of its own, so this organization connects through one it registers with {1}. Create an app on the {1} developer portal and enter its keys below.",
              appName,
              vendor.developer,
            )}
          </p>
        ) : null}

        {!noApp && tenantApp && !showForm ? (
          <DescriptionList columns={2}>
            <DescriptionItem label={t("Client ID")}>
              <span className="font-mono text-xs break-all">{tenantApp.clientId}</span>
            </DescriptionItem>
            {choosesEnvironment ? (
              <DescriptionItem label={t("Environment")}>
                <Badge variant={environmentAccents[tenantApp.environment]}>
                  {environmentLabels[tenantApp.environment]}
                </Badge>
              </DescriptionItem>
            ) : null}
            <DescriptionItem label={webhookKeyLabel}>
              {tenantApp.hasWebhookVerifier ? t("Saved") : t("Not set")}
            </DescriptionItem>
            <DescriptionItem label={t("Last changed")} numeric>
              {formatUnixDateMedium(tenantApp.updatedAt)}
            </DescriptionItem>
          </DescriptionList>
        ) : null}

        {!noApp && !tenantApp && !showForm ? (
          <p className="text-foreground-muted text-sm">
            {app.instanceEnvironment && choosesEnvironment
              ? t(
                  "Connects through the {0} this server is configured with ({1}). Use your own app instead if your organization registers one with {2}.",
                  appName,
                  environmentLabels[app.instanceEnvironment],
                  vendor.developer,
                )
              : t(
                  "Connects through the {0} this server is configured with. Use your own app instead if your organization registers one with {1}.",
                  appName,
                  vendor.developer,
                )}
          </p>
        ) : null}

        {!app.redirectUrl ? (
          <Alert size="sm" variant="warning">
            <AlertDescription>
              {t(
                "This Trenova server has no web address configured (app.webBaseUrl), so {0} cannot send people back after they sign in. Ask the server's administrator to set it.",
                vendor.name,
              )}
            </AlertDescription>
          </Alert>
        ) : null}

        {showForm ? (
          <AccountingAppKeysForm
            vendor={vendor}
            app={app}
            profile={profile}
            locked={connectedThroughTenant}
            onDone={noApp ? undefined : () => setEditing(false)}
          />
        ) : null}

        {!showForm && canManage ? (
          <div className="space-y-2">
            {connectedThroughInstance && !tenantApp ? (
              <p className="text-foreground-muted text-xs">
                {t(
                  "Disconnect {0} before switching to your own app. Its authorization belongs to the app it was connected through.",
                  connection?.externalCompanyName || vendor.name,
                )}
              </p>
            ) : null}
            {connectedThroughTenant ? (
              <p className="text-foreground-muted text-xs">
                {t(
                  "While {0} is connected, only the client secret and {1} can change.",
                  connection?.externalCompanyName || vendor.name,
                  toSentenceFragment(webhookKeyLabel),
                )}
              </p>
            ) : null}
            <div className="flex justify-end gap-2">
              {tenantApp ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={connectedThroughTenant}
                  onClick={() => setConfirmRemove(true)}
                >
                  {t("Remove")}
                </Button>
              ) : null}
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={!tenantApp && connectedThroughInstance}
                onClick={() => setEditing(true)}
              >
                {tenantApp ? t("Change keys") : t("Use your own app")}
              </Button>
            </div>
          </div>
        ) : null}

        {!canManage && noApp ? (
          <Alert size="sm" variant="warning">
            <AlertDescription>
              {t(
                "Someone with manage access to the accounting integration has to enter the {0} keys.",
                appName,
              )}
            </AlertDescription>
          </Alert>
        ) : null}
      </div>

      <AlertDialog open={confirmRemove} onOpenChange={setConfirmRemove}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Remove the {0} keys?", appName)}</AlertDialogTitle>
            <AlertDialogDescription>
              {app.instanceAppAvailable
                ? t(
                    "The next connection goes through this server's {0} instead. You can enter your keys again later.",
                    appName,
                  )
                : t(
                    "This server has no {0} of its own, so nobody can connect {1} until keys are entered again.",
                    appName,
                    vendor.name,
                  )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={remove.isPending}>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              isLoading={remove.isPending}
              loadingText={t("Removing...")}
              onClick={() => {
                void remove.mutateAsync().then(
                  () => setConfirmRemove(false),
                  () => undefined,
                );
              }}
            >
              {t("Remove")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SectionPanel>
  );
}

const environmentAccents = {
  Sandbox: "accent-amber",
  Production: "accent-sky",
} as const satisfies Record<AccountingAppEnvironment, string>;

function isLive(connection: AccountingConnection): boolean {
  return connection.status !== "Disconnected" && connection.status !== "Revoked";
}

function AccountingAppKeysForm({
  vendor,
  app,
  profile,
  locked,
  onDone,
}: {
  vendor: AccountingVendor;
  app: AccountingAppSettings;
  profile: AccountingProviderProfile;
  locked: boolean;
  onDone?: () => void;
}) {
  const t = useT();
  const saved = app.tenantApp;
  const environments = profile.environments;
  const webhookKeyLabel = t(profile.webhookKeyLabel);
  const form = useForm<AccountingAppFormValues>({
    resolver: zodResolver(accountingAppFormSchema(saved, environments)),
    defaultValues: accountingAppFormDefaults(saved, environments),
  });
  const { control, handleSubmit, reset } = form;
  const environmentLabels: Record<AccountingAppEnvironment, string> = {
    Sandbox: t("Sandbox"),
    Production: t("Production"),
  };
  const environmentDescriptions: Record<AccountingAppEnvironment, string> = {
    Sandbox: t("Development keys, for test companies"),
    Production: t("Production keys, for real companies"),
  };
  const { save } = useAccountingAppActions(vendor, form);

  const onSubmit = async (values: AccountingAppFormValues) => {
    const accepted = await save.mutateAsync(values).then(
      () => true,
      () => false,
    );
    if (!accepted) {
      return;
    }
    reset({
      ...values,
      clientSecret: "",
      webhookVerifierToken: "",
      clearWebhookVerifierToken: false,
    });
    onDone?.();
  };

  return (
    <Form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <AccountingAppRegistration vendor={vendor} app={app} webhookKeyLabel={webhookKeyLabel} />
      <FormGroup cols={2}>
        {environments.length > 1 ? (
          <FormControl>
            <SelectField
              name="environment"
              control={control}
              label={t("Environment")}
              isReadOnly={locked}
              options={environments.map((environment) => ({
                value: environment,
                label: environmentLabels[environment],
                description: environmentDescriptions[environment],
              }))}
            />
          </FormControl>
        ) : null}
        <FormControl cols={environments.length > 1 ? undefined : "full"}>
          <InputField
            name="clientId"
            control={control}
            label={t("Client ID")}
            readOnly={locked}
            autoComplete="off"
            spellCheck={false}
          />
        </FormControl>
        <FormControl cols="full">
          <SensitiveField
            name="clientSecret"
            control={control}
            label={t("Client secret")}
            autoComplete="new-password"
            placeholder={saved ? t("Saved. Leave blank to keep it.") : undefined}
            description={
              environments.length > 1
                ? t(
                    "Stored encrypted and never shown again. Enter it again whenever the client ID or environment changes.",
                  )
                : t(
                    "Stored encrypted and never shown again. Enter it again whenever the client ID changes.",
                  )
            }
          />
        </FormControl>
        <FormControl cols="full">
          <SensitiveField
            name="webhookVerifierToken"
            control={control}
            label={t("{0} (optional)", webhookKeyLabel)}
            autoComplete="new-password"
            placeholder={
              saved?.hasWebhookVerifier ? t("Saved. Leave blank to keep it.") : undefined
            }
            description={t(
              "From the app's Webhooks page. Lets Trenova check that notices really come from {0}.",
              vendor.name,
            )}
          />
        </FormControl>
        {saved?.hasWebhookVerifier ? (
          <FormControl cols="full">
            <CheckboxField
              name="clearWebhookVerifierToken"
              control={control}
              label={t("Remove the saved {0}", toSentenceFragment(webhookKeyLabel))}
            />
          </FormControl>
        ) : null}
      </FormGroup>
      <div className="flex justify-end gap-2">
        {onDone ? (
          <Button type="button" variant="outline" onClick={onDone} disabled={save.isPending}>
            {t("Cancel")}
          </Button>
        ) : null}
        <Button
          type="submit"
          isLoading={save.isPending}
          loadingText={t("Checking with {0}...", vendor.developer)}
        >
          {t("Save keys")}
        </Button>
      </div>
    </Form>
  );
}

function AccountingAppRegistration({
  vendor,
  app,
  webhookKeyLabel,
}: {
  vendor: AccountingVendor;
  app: AccountingAppSettings;
  webhookKeyLabel: string;
}) {
  const t = useT();
  const appWebhookPath = app.tenantApp ? app.webhookPath : "";
  const webhookUrl = accountingWebhookUrl(appWebhookPath);
  const steps: Record<AccountingSystem, { create: string; redirect: string; keys: string }> = {
    QuickBooksOnline: {
      create: t(
        "Create an app on the Intuit developer portal with the com.intuit.quickbooks.accounting scope.",
      ),
      redirect: t(
        "Under Keys and credentials, add the redirect URI below exactly as written. Development keys accept http://localhost; production keys need https.",
      ),
      keys: t("Copy the client ID and client secret for the same environment into this form."),
    },
    Xero: {
      create: t("Create a Web app on the Xero developer portal."),
      redirect: t(
        "Under Configuration, add the redirect URI below exactly as written. Xero accepts http://localhost while developing; anything else needs https.",
      ),
      keys: t("Copy the client ID, generate a client secret, and enter both in this form."),
    },
  };
  const step = steps[vendor.system];

  return (
    <div className="space-y-3 rounded-md border p-3">
      <ol className="text-foreground-muted list-decimal space-y-1 pl-4 text-sm">
        <li>
          {step.create}{" "}
          <ExternalLink href={vendor.developerPortalUrl} className="text-sm">
            {t("Open the {0} developer portal", vendor.developer)}
          </ExternalLink>
        </li>
        <li>{step.redirect}</li>
        <li>
          {step.keys}{" "}
          <ExternalLink href={vendor.appKeysHelpUrl} className="text-sm">
            {t("Where to find them")}
          </ExternalLink>
        </li>
        <li>
          {webhookUrl
            ? t(
                "To have {0} tell Trenova about changes, add the webhook endpoint below to the app's webhooks and enter its {1} here.",
                vendor.name,
                toSentenceFragment(webhookKeyLabel),
              )
            : t(
                "To have {0} tell Trenova about changes, save the keys first: the webhook endpoint for your app appears here once they are saved.",
                vendor.name,
              )}
        </li>
      </ol>
      {app.redirectUrl ? <CopyRow label={t("Redirect URI")} value={app.redirectUrl} /> : null}
      {webhookUrl ? <CopyRow label={t("Webhook endpoint (optional)")} value={webhookUrl} /> : null}
    </div>
  );
}

function CopyRow({ label, value }: { label: string; value: string }) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      <div className="bg-muted flex items-center gap-2 rounded-md border p-2">
        <p className="min-w-0 flex-1 truncate font-mono text-xs" title={value}>
          {value}
        </p>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 px-2"
          onClick={() => void copy(value)}
        >
          {isCopied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
          <span className="sr-only">{t("Copy {0}", label)}</span>
        </Button>
      </div>
    </div>
  );
}
