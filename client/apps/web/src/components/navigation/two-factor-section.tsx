import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
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
import { AlertTriangleIcon, Copy01Icon, Download01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { mfaService } from "@trenova/shared/services/mfa";
import {
  recoveryCodesFileContents,
  type MFAStatus,
  type TOTPEnrollment,
} from "@trenova/shared/types/mfa";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

export const MFA_STATUS_QUERY_KEY = ["mfa", "status"] as const;

type Dialogs = "setup" | "disable" | "regenerate" | null;

/**
 * The person's authenticator-app second factor: whether it is on, how many recovery
 * codes are left, and the dialogs that turn it on, off, or replace the codes.
 */
export function TwoFactorSection() {
  const t = useT();
  const [open, setOpen] = useState<Dialogs>(null);
  const statusQuery = useQuery({
    queryKey: MFA_STATUS_QUERY_KEY,
    queryFn: () => mfaService.status(),
  });
  const status = statusQuery.data;

  return (
    <div className="flex flex-col gap-3" data-testid="two-factor-section">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <span className="text-sm font-medium">{t("Authenticator app")}</span>
          {status ? (
            <Badge variant={status.totpEnabled ? "success" : "neutral"}>
              {status.totpEnabled ? t("On") : t("Off")}
            </Badge>
          ) : null}
        </div>
        {status ? <TwoFactorActions status={status} onOpen={setOpen} /> : null}
      </div>
      {status?.totpEnabled ? <TwoFactorSummary status={status} /> : null}
      {statusQuery.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertTriangleIcon />
          <AlertDescription>{t("Two-factor settings could not be loaded.")}</AlertDescription>
        </Alert>
      ) : null}

      <SetupDialog open={open === "setup"} onClose={() => setOpen(null)} />
      <DisableDialog open={open === "disable"} onClose={() => setOpen(null)} />
      <RegenerateDialog open={open === "regenerate"} onClose={() => setOpen(null)} />
    </div>
  );
}

function TwoFactorActions({
  status,
  onOpen,
}: {
  status: MFAStatus;
  onOpen: (dialog: Dialogs) => void;
}) {
  const t = useT();

  if (!status.totpEnabled) {
    return (
      <Button type="button" size="sm" onClick={() => onOpen("setup")}>
        {t("Set up")}
      </Button>
    );
  }

  return (
    <div className="flex shrink-0 gap-2">
      <Button type="button" variant="outline" size="sm" onClick={() => onOpen("regenerate")}>
        {t("New recovery codes")}
      </Button>
      <Button type="button" variant="outline" size="sm" onClick={() => onOpen("disable")}>
        {t("Turn off")}
      </Button>
    </div>
  );
}

function TwoFactorSummary({ status }: { status: MFAStatus }) {
  const t = useT();

  return (
    <div className="flex flex-col gap-2">
      <p className="text-muted-foreground text-xs">
        {status.enabledAt
          ? t("On since {0}.", formatUnixDateMedium(status.enabledAt))
          : t("Sign-in asks for a code from your authenticator app.")}{" "}
        {t(
          "{0, plural, one {# recovery code left.} other {# recovery codes left.}}",
          status.recoveryCodesRemaining,
        )}
      </p>
      {status.recoveryCodesRemaining <= 2 ? (
        <Alert variant="warning" size="sm">
          <AlertTriangleIcon />
          <AlertDescription>
            {t("You are running out of recovery codes. Create new ones and store them safely.")}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}

function useRefreshStatus() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: MFA_STATUS_QUERY_KEY });
}

function RecoveryCodesPanel({ codes }: { codes: string[] }) {
  const t = useT();

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(codes.join("\n"));
      toast.success(t("Recovery codes copied"));
    } catch {
      toast.error(t("Could not copy the recovery codes"));
    }
  };

  const download = () => {
    const blob = new Blob([recoveryCodesFileContents(codes, "Trenova")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "trenova-recovery-codes.txt";
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex flex-col gap-3">
      <Alert variant="warning" size="sm">
        <AlertTriangleIcon />
        <AlertTitle>{t("Save these recovery codes now")}</AlertTitle>
        <AlertDescription>
          {t(
            "Each code signs you in once if you lose your authenticator. They are shown only this time.",
          )}
        </AlertDescription>
      </Alert>
      <ul
        className="bg-muted grid grid-cols-2 gap-x-6 gap-y-1 rounded-md border p-3 font-mono text-sm tabular-nums"
        aria-label={t("Recovery codes")}
      >
        {codes.map((code) => (
          <li key={code}>{code}</li>
        ))}
      </ul>
      <div className="flex gap-2">
        <Button type="button" variant="outline" size="sm" onClick={() => void copy()}>
          <Copy01Icon className="size-3.5" />
          {t("Copy")}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={download}>
          <Download01Icon className="size-3.5" />
          {t("Download")}
        </Button>
      </div>
    </div>
  );
}

type PasswordValues = { password: string };
type CodeValues = { code: string };

function SetupDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();
  const refresh = useRefreshStatus();
  const [enrollment, setEnrollment] = useState<TOTPEnrollment | null>(null);
  const [codes, setCodes] = useState<string[] | null>(null);
  const passwordForm = useForm<PasswordValues>({ defaultValues: { password: "" } });
  const codeForm = useForm<CodeValues>({ defaultValues: { code: "" } });

  const begin = useApiMutation<TOTPEnrollment, PasswordValues, unknown, PasswordValues>({
    mutationFn: (values) => mfaService.beginEnrollment(values.password),
    form: passwordForm,
    resourceName: "Two-factor setup",
    onSuccess: (data) => setEnrollment(data),
  });

  const confirm = useApiMutation<{ recoveryCodes: string[] }, CodeValues, unknown, CodeValues>({
    mutationFn: (values) => mfaService.confirmEnrollment(values.code),
    form: codeForm,
    resourceName: "Two-factor setup",
    onSuccess: async (data) => {
      setCodes(data.recoveryCodes);
      await refresh();
      toast.success(t("Two-factor authentication is on"));
    },
  });

  const close = () => {
    setEnrollment(null);
    setCodes(null);
    passwordForm.reset();
    codeForm.reset();
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Set up two-factor authentication")}</DialogTitle>
          <DialogDescription>
            {t("Sign-in will ask for a code from an authenticator app as well as your password.")}
          </DialogDescription>
        </DialogHeader>

        {codes ? (
          <RecoveryCodesPanel codes={codes} />
        ) : enrollment ? (
          <Form onSubmit={codeForm.handleSubmit((values) => void confirm.mutateAsync(values))}>
            <div className="flex flex-col items-center gap-3">
              <img
                src={enrollment.qrCode}
                alt={t("QR code for your authenticator app")}
                className="size-48 rounded-md border bg-white p-2"
              />
              <p className="text-muted-foreground text-center text-xs">
                {t("Scan the code with your authenticator app, or enter this key by hand:")}
              </p>
              <code className="bg-muted rounded-md border px-2 py-1 font-mono text-xs break-all">
                {enrollment.secret}
              </code>
            </div>
            <FormGroup cols={1} className="mt-3">
              <FormControl>
                <InputField
                  control={codeForm.control}
                  name="code"
                  label={t("Code from the app")}
                  placeholder="123 456"
                  autoComplete="one-time-code"
                  inputMode="numeric"
                  rules={{ required: t("Enter the six-digit code") }}
                />
              </FormControl>
            </FormGroup>
          </Form>
        ) : (
          <Form onSubmit={passwordForm.handleSubmit((values) => void begin.mutateAsync(values))}>
            <FormGroup cols={1}>
              <FormControl>
                <SensitiveField
                  control={passwordForm.control}
                  name="password"
                  label={t("Current password")}
                  placeholder={t("Confirm it is you")}
                  rules={{ required: t("Password is required") }}
                />
              </FormControl>
            </FormGroup>
          </Form>
        )}

        <DialogFooter>
          {codes ? (
            <Button type="button" onClick={close}>
              {t("Done")}
            </Button>
          ) : (
            <>
              <Button type="button" variant="outline" onClick={close}>
                {t("Cancel")}
              </Button>
              {enrollment ? (
                <Button
                  type="button"
                  isLoading={confirm.isPending}
                  loadingText={t("Verifying...")}
                  onClick={codeForm.handleSubmit((values) => void confirm.mutateAsync(values))}
                >
                  {t("Turn on")}
                </Button>
              ) : (
                <Button
                  type="button"
                  isLoading={begin.isPending}
                  loadingText={t("Preparing...")}
                  onClick={passwordForm.handleSubmit((values) => void begin.mutateAsync(values))}
                >
                  {t("Continue")}
                </Button>
              )}
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type DisableValues = { password: string; code: string };

function DisableDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();
  const refresh = useRefreshStatus();
  const form = useForm<DisableValues>({ defaultValues: { password: "", code: "" } });

  const disable = useApiMutation<undefined, DisableValues, unknown, DisableValues>({
    mutationFn: async (values) => {
      await mfaService.disable({ password: values.password, code: values.code });
      return undefined;
    },
    form,
    resourceName: "Two-factor authentication",
    onSuccess: async () => {
      await refresh();
      toast.success(t("Two-factor authentication is off"));
      form.reset();
      onClose();
    },
  });

  const submit = form.handleSubmit((values) => void disable.mutateAsync(values));

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          form.reset();
          onClose();
        }
      }}
    >
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Turn off two-factor authentication")}</DialogTitle>
          <DialogDescription>
            {t("Sign-in will ask only for your password. Your recovery codes stop working.")}
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={submit}>
          <FormGroup cols={1}>
            <FormControl>
              <SensitiveField
                control={form.control}
                name="password"
                label={t("Current password")}
                placeholder={t("Enter your password")}
                rules={{ required: t("Password is required") }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="code"
                label={t("Code from the app")}
                placeholder="123 456"
                autoComplete="one-time-code"
                inputMode="numeric"
                rules={{ required: t("Enter the six-digit code") }}
              />
            </FormControl>
          </FormGroup>
        </Form>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              form.reset();
              onClose();
            }}
          >
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            isLoading={disable.isPending}
            loadingText={t("Turning off...")}
            onClick={submit}
          >
            {t("Turn off")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RegenerateDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();
  const refresh = useRefreshStatus();
  const [codes, setCodes] = useState<string[] | null>(null);
  const form = useForm<CodeValues>({ defaultValues: { code: "" } });

  const regenerate = useApiMutation<{ recoveryCodes: string[] }, CodeValues, unknown, CodeValues>({
    mutationFn: (values) => mfaService.regenerateRecoveryCodes(values.code),
    form,
    resourceName: "Recovery codes",
    onSuccess: async (data) => {
      setCodes(data.recoveryCodes);
      await refresh();
    },
  });

  const close = () => {
    setCodes(null);
    form.reset();
    onClose();
  };
  const submit = form.handleSubmit((values) => void regenerate.mutateAsync(values));

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("New recovery codes")}</DialogTitle>
          <DialogDescription>
            {t("Your current recovery codes stop working as soon as the new ones are made.")}
          </DialogDescription>
        </DialogHeader>
        {codes ? (
          <RecoveryCodesPanel codes={codes} />
        ) : (
          <Form onSubmit={submit}>
            <FormGroup cols={1}>
              <FormControl>
                <InputField
                  control={form.control}
                  name="code"
                  label={t("Code from the app")}
                  placeholder="123 456"
                  autoComplete="one-time-code"
                  inputMode="numeric"
                  rules={{ required: t("Enter the six-digit code") }}
                />
              </FormControl>
            </FormGroup>
          </Form>
        )}
        <DialogFooter>
          {codes ? (
            <Button type="button" onClick={close}>
              {t("Done")}
            </Button>
          ) : (
            <>
              <Button type="button" variant="outline" onClick={close}>
                {t("Cancel")}
              </Button>
              <Button
                type="button"
                isLoading={regenerate.isPending}
                loadingText={t("Creating...")}
                onClick={submit}
              >
                {t("Create codes")}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
