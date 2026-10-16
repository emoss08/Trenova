import { InputField } from "@/components/fields/input-field";
import { ExternalLink } from "@/components/link";
import { accountingSetupPath } from "@/lib/accounting-sync";
import type {
  AccountingAuthorizationResult,
  AccountingConnection,
  AccountingPreparedCertificate,
} from "@/lib/graphql/accounting-sync";
import { zodResolver } from "@hookform/resolvers/zod";
import { Download01Icon, RefreshCw01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { downloadTextFile } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import {
  AccountingCompanyChoice,
  type AccountingCompanyChoiceOffer,
} from "./accounting-company-choice";
import {
  accountingCertificateConnectSchema,
  accountingCertificateRenewSchema,
  type AccountingCertificateConnectValues,
  type AccountingCertificateRenewValues,
} from "./accounting-certificate-schema";
import type { AccountingVendor } from "./accounting-vendors";
import { useAccountingCertificateActions } from "./use-accounting-connection";

const PEM_CONTENT_TYPE = "application/x-pem-file";
const CLIENT_CREDENTIALS_HELP_URL =
  "https://docs.oracle.com/en/cloud/saas/netsuite/ns-online-help/section_162686838198.html";

export function certificateFileName(vendor: AccountingVendor, preparedAt: number): string {
  const day = new Date(preparedAt * 1000).toISOString().slice(0, 10);
  return `trenova-${vendor.system.toLowerCase()}-certificate-${day}.pem`;
}

type AccountingCertificateFileProps = {
  vendor: AccountingVendor;
  pending: AccountingPreparedCertificate | null;
  canManage: boolean;
};

export function AccountingCertificateFile({
  vendor,
  pending,
  canManage,
}: AccountingCertificateFileProps) {
  const t = useT();
  const { prepare } = useAccountingCertificateActions(vendor);

  if (!pending) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border p-3">
        <p className="text-foreground-muted min-w-0 flex-1 text-sm">
          {t(
            "Trenova generates a key pair and a certificate valid for two years. The private key never leaves Trenova; only the certificate is uploaded to {0}.",
            vendor.name,
          )}
        </p>
        <Button
          type="button"
          size="sm"
          onClick={() => prepare.mutate()}
          disabled={!canManage}
          isLoading={prepare.isPending}
          loadingText={t("Generating...")}
        >
          {t("Generate certificate")}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border p-3">
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="text-sm font-medium">{t("Certificate ready to upload")}</p>
        <p className="text-foreground-muted text-xs">
          {t(
            "Generated {0}; expires {1}.",
            formatUnixDateMedium(pending.preparedAt),
            formatUnixDateMedium(pending.expiresAt),
          )}
        </p>
      </div>
      <div className="flex gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() => prepare.mutate()}
          disabled={!canManage}
          isLoading={prepare.isPending}
          loadingText={t("Generating...")}
        >
          <RefreshCw01Icon aria-hidden className="size-3.5" />
          {t("Generate a new one")}
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={() =>
            downloadTextFile(
              certificateFileName(vendor, pending.preparedAt),
              pending.certificatePem,
              PEM_CONTENT_TYPE,
            )
          }
        >
          <Download01Icon aria-hidden className="size-3.5" />
          {t("Download certificate")}
        </Button>
      </div>
    </div>
  );
}

export function AccountingCertificateSteps({
  vendor,
  appName,
}: {
  vendor: AccountingVendor;
  appName: string;
}) {
  const t = useT();

  return (
    <ol className="text-foreground-muted list-decimal space-y-1 pl-4 text-sm">
      <li>{t("Generate a certificate here and download it.")}</li>
      <li>
        {t(
          "In {0}, go to Setup > Integration > OAuth 2.0 Client Credentials (M2M) Setup and choose Create New.",
          vendor.name,
        )}{" "}
        <ExternalLink href={CLIENT_CREDENTIALS_HELP_URL} className="text-sm">
          {t("How the mapping works")}
        </ExternalLink>
      </li>
      <li>
        {t(
          "Choose the employee and role Trenova acts as, choose the {0} Trenova connects through, upload the certificate, and save.",
          appName,
        )}
      </li>
      <li>
        {t(
          "Copy the certificate ID {0} shows for the new mapping into the form below.",
          vendor.name,
        )}
      </li>
    </ol>
  );
}

type AccountingCertificateConnectProps = {
  vendor: AccountingVendor;
  appName: string;
  pending: AccountingPreparedCertificate | null;
  canManage: boolean;
  disabled: boolean;
  onCancel: () => void;
};

export function AccountingCertificateConnect({
  vendor,
  appName,
  pending,
  canManage,
  disabled,
  onCancel,
}: AccountingCertificateConnectProps) {
  const t = useT();
  const navigate = useNavigate();
  const [choice, setChoice] = useState<AccountingCompanyChoiceOffer | null>(null);
  const form = useForm<AccountingCertificateConnectValues>({
    resolver: zodResolver(accountingCertificateConnectSchema),
    defaultValues: { accountId: "", certificateId: "" },
  });
  const { control, handleSubmit } = form;
  const { connect, refresh } = useAccountingCertificateActions(vendor, { connect: form });

  const connected = async () => {
    await refresh();
    void navigate(`${accountingSetupPath(vendor.system)}&setup=connected`, { replace: true });
  };

  const onResult = (result: AccountingAuthorizationResult) => {
    if (result.kind === "connected") {
      void connected();
      return;
    }
    if (result.kind === "choose") {
      setChoice(result);
      return;
    }
    toast.error(t("{0} did not offer any company to connect", vendor.name));
  };

  if (choice) {
    return (
      <AccountingCompanyChoice
        vendor={vendor}
        choice={choice}
        onConnected={() => void connected()}
        cancelAction={
          <Button type="button" variant="outline" onClick={() => setChoice(null)}>
            {t("Back")}
          </Button>
        }
        expired={
          <Alert size="sm" variant="warning">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>
                {t("The time to choose ran out, so nothing was connected. Connect again.")}
              </span>
              <Button type="button" size="sm" variant="outline" onClick={() => setChoice(null)}>
                {t("Back")}
              </Button>
            </AlertDescription>
          </Alert>
        }
      />
    );
  }

  return (
    <Form
      onSubmit={handleSubmit((values) =>
        connect.mutateAsync(values).then(onResult, () => undefined),
      )}
      className="space-y-4"
    >
      <AccountingCertificateSteps vendor={vendor} appName={appName} />
      <AccountingCertificateFile vendor={vendor} pending={pending} canManage={canManage} />
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            name="accountId"
            control={control}
            label={t("Account ID")}
            placeholder="1234567"
            autoComplete="off"
            spellCheck={false}
            description={t("Setup > Company > Company Information. Sandboxes end in _SB1.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            name="certificateId"
            control={control}
            label={t("Certificate ID")}
            autoComplete="off"
            spellCheck={false}
            description={t("Shown on the mapping after the certificate is uploaded.")}
          />
        </FormControl>
      </FormGroup>
      <div className="flex justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("Cancel")}
        </Button>
        <Button
          type="submit"
          disabled={disabled || !canManage || pending === null}
          isLoading={connect.isPending}
          loadingText={t("Checking with {0}...", vendor.name)}
        >
          {t("Connect with the certificate")}
        </Button>
      </div>
    </Form>
  );
}

type AccountingCertificateRenewDialogProps = {
  vendor: AccountingVendor;
  appName: string;
  connection: AccountingConnection;
  pending: AccountingPreparedCertificate | null;
  canManage: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function AccountingCertificateRenewDialog({
  vendor,
  appName,
  connection,
  pending,
  canManage,
  open,
  onOpenChange,
}: AccountingCertificateRenewDialogProps) {
  const t = useT();
  const form = useForm<AccountingCertificateRenewValues>({
    resolver: zodResolver(accountingCertificateRenewSchema),
    defaultValues: { certificateId: "" },
  });
  const { control, handleSubmit, reset } = form;
  const { renew } = useAccountingCertificateActions(vendor, { renew: form });

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          reset();
        }
        onOpenChange(next);
      }}
    >
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Renew the {0} certificate", vendor.name)}</DialogTitle>
          <DialogDescription>
            {t(
              "Upload a new certificate to the same mapping in {0}. The current certificate keeps working until {0} accepts the new one.",
              vendor.name,
            )}
          </DialogDescription>
        </DialogHeader>
        <Form
          onSubmit={handleSubmit((values) =>
            renew.mutateAsync({ connection, values }).then(
              () => {
                reset();
                onOpenChange(false);
              },
              () => undefined,
            ),
          )}
          className="space-y-4"
        >
          <AccountingCertificateSteps vendor={vendor} appName={appName} />
          <AccountingCertificateFile vendor={vendor} pending={pending} canManage={canManage} />
          <FormGroup cols={1}>
            <FormControl>
              <InputField
                name="certificateId"
                control={control}
                label={t("New certificate ID")}
                autoComplete="off"
                spellCheck={false}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t("Cancel")}
            </Button>
            <Button
              type="submit"
              disabled={!canManage || pending === null}
              isLoading={renew.isPending}
              loadingText={t("Checking with {0}...", vendor.name)}
            >
              {t("Renew certificate")}
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
