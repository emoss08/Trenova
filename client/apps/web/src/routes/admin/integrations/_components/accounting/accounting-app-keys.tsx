import { CheckboxField } from "@/components/fields/checkbox-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { ExternalLink } from "@/components/link";
import { SectionPanel } from "@/components/section-panel";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { accountingWebhookUrl } from "@/lib/accounting-sync";
import type { AccountingAppSettings, AccountingConnection } from "@/lib/graphql/accounting-sync";
import type { AccountingAppEnvironment } from "@trenova/graphql/generated/graphql";
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
import { CheckIcon, CopyIcon, KeyRoundIcon } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import type { AccountingVendor } from "./accounting-vendors";
import {
  ACCOUNTING_APP_ENVIRONMENTS,
  accountingAppFormDefaults,
  accountingAppFormSchema,
  type AccountingAppFormValues,
} from "./accounting-app-schema";
import { useAccountingAppActions } from "./use-accounting-connection";

type AccountingAppKeysProps = {
  vendor: AccountingVendor;
  app: AccountingAppSettings;
  connection: AccountingConnection | null;
  canManage: boolean;
};

export function AccountingAppKeys({ vendor, app, connection, canManage }: AccountingAppKeysProps) {
  const t = useT();
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
      title={vendor.appName}
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
              "This Trenova server has no {0} of its own, so this organization connects through one it registers with Intuit. Create an app on the Intuit developer portal and enter its keys below.",
              vendor.appName,
            )}
          </p>
        ) : null}

        {!noApp && tenantApp && !showForm ? (
          <DescriptionList columns={2}>
            <DescriptionItem label={t("Client ID")}>
              <span className="font-mono text-xs break-all">{tenantApp.clientId}</span>
            </DescriptionItem>
            <DescriptionItem label={t("Environment")}>
              <Badge variant={environmentAccents[tenantApp.environment]}>
                {environmentLabels[tenantApp.environment]}
              </Badge>
            </DescriptionItem>
            <DescriptionItem label={t("Webhook verifier token")}>
              {tenantApp.hasWebhookVerifier ? t("Saved") : t("Not set")}
            </DescriptionItem>
            <DescriptionItem label={t("Last changed")} numeric>
              {formatUnixDateMedium(tenantApp.updatedAt)}
            </DescriptionItem>
          </DescriptionList>
        ) : null}

        {!noApp && !tenantApp && !showForm ? (
          <p className="text-foreground-muted text-sm">
            {app.instanceEnvironment
              ? t(
                  "Connects through the {0} this server is configured with ({1}). Use your own app instead if your organization registers one with Intuit.",
                  vendor.appName,
                  environmentLabels[app.instanceEnvironment],
                )
              : t("Connects through the {0} this server is configured with.", vendor.appName)}
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
                  "While {0} is connected, only the client secret and webhook verifier token can change.",
                  connection?.externalCompanyName || vendor.name,
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
                vendor.appName,
              )}
            </AlertDescription>
          </Alert>
        ) : null}
      </div>

      <AlertDialog open={confirmRemove} onOpenChange={setConfirmRemove}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Remove the {0} keys?", vendor.appName)}</AlertDialogTitle>
            <AlertDialogDescription>
              {app.instanceAppAvailable
                ? t(
                    "The next connection goes through this server's {0} instead. You can enter your keys again later.",
                    vendor.appName,
                  )
                : t(
                    "This server has no {0} of its own, so nobody can connect {1} until keys are entered again.",
                    vendor.appName,
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
  locked,
  onDone,
}: {
  vendor: AccountingVendor;
  app: AccountingAppSettings;
  locked: boolean;
  onDone?: () => void;
}) {
  const t = useT();
  const saved = app.tenantApp;
  const form = useForm<AccountingAppFormValues>({
    resolver: zodResolver(accountingAppFormSchema(saved)),
    defaultValues: accountingAppFormDefaults(saved),
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
      <AccountingAppRegistration vendor={vendor} app={app} />
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            name="environment"
            control={control}
            label={t("Environment")}
            isReadOnly={locked}
            options={ACCOUNTING_APP_ENVIRONMENTS.map((environment) => ({
              value: environment,
              label: environmentLabels[environment],
              description: environmentDescriptions[environment],
            }))}
          />
        </FormControl>
        <FormControl>
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
            description={t(
              "Stored encrypted and never shown again. Enter it again whenever the client ID or environment changes.",
            )}
          />
        </FormControl>
        <FormControl cols="full">
          <SensitiveField
            name="webhookVerifierToken"
            control={control}
            label={t("Webhook verifier token (optional)")}
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
              label={t("Remove the saved webhook verifier token")}
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
        <Button type="submit" isLoading={save.isPending} loadingText={t("Checking with Intuit...")}>
          {t("Save keys")}
        </Button>
      </div>
    </Form>
  );
}

function AccountingAppRegistration({
  vendor,
  app,
}: {
  vendor: AccountingVendor;
  app: AccountingAppSettings;
}) {
  const t = useT();
  const webhookUrl = accountingWebhookUrl(app.webhookPath);

  return (
    <div className="space-y-3 rounded-md border p-3">
      <ol className="text-foreground-muted list-decimal space-y-1 pl-4 text-sm">
        <li>
          {t("Create an app on the")}{" "}
          <ExternalLink href={vendor.developerPortalUrl} className="text-sm">
            {t("Intuit developer portal")}
          </ExternalLink>{" "}
          {t("with the com.intuit.quickbooks.accounting scope.")}
        </li>
        <li>
          {t(
            "Under Keys and credentials, add the redirect URI below exactly as written. Development keys accept http://localhost; production keys need https.",
          )}
        </li>
        <li>
          {t("Copy the client ID and client secret for the same environment into this form.")}{" "}
          <ExternalLink href={vendor.appKeysHelpUrl} className="text-sm">
            {t("Where to find them")}
          </ExternalLink>
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
