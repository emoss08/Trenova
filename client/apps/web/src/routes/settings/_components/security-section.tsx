import { FieldWrapper } from "@/components/fields/field-components";
import { SensitiveField } from "@/components/fields/sensitive-field";
import {
  TwoFactorDialogs,
  useMFAStatus,
  type TwoFactorDialog,
} from "@/components/navigation/two-factor-section";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { user as userQueries } from "@/lib/queries/user";
import {
  daysSince,
  describeSignIn,
  passwordChecks,
  passwordScore,
  type PasswordCheck,
  type SignInKind,
} from "@/lib/user-settings";
import { apiService } from "@/services/api";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  CheckIcon,
  Key01Icon,
  Laptop01Icon,
  LogOut01Icon,
  Monitor01Icon,
  Phone01Icon,
  ShieldTickIcon,
  Tablet01Icon,
  XCloseIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone, resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { DeviceKind, SessionSummary, SignInSummary } from "@trenova/shared/types/user";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { useSettingsFeedback } from "./settings-feedback";
import { SettingsCard, SettingsRow } from "./settings-layout";

const LEAVE_MS = 300;

type PasswordValues = {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
};

const METER_COLOR = ["", "bg-danger", "bg-warning", "bg-warning", "bg-success"];

function PasswordRow() {
  const t = useT();
  const feedback = useSettingsFeedback();
  const user = useAuthStore((state) => state.user);
  const now = useNow("minute");
  const [open, setOpen] = useState(false);
  const form = useForm<PasswordValues>({
    defaultValues: { currentPassword: "", newPassword: "", confirmPassword: "" },
  });
  const [currentPassword, newPassword, confirmPassword] = useWatch({
    control: form.control,
    name: ["currentPassword", "newPassword", "confirmPassword"],
  });
  const checks = passwordChecks(newPassword);
  const score = passwordScore(newPassword);
  const matches = confirmPassword !== "" && confirmPassword === newPassword;
  const ready = currentPassword !== "" && score === 4 && matches;
  const requirementLabels: Record<PasswordCheck, string> = {
    length: t("12+ characters"),
    number: t("A number"),
    case: t("Upper and lower case"),
    symbol: t("A symbol"),
  };

  const changedAt = user?.passwordChangedAt;
  const description = changedAt
    ? daysSince(changedAt, new Date(now * 1000)) === 0
      ? t("Changed today.")
      : t(
          "{0, plural, one {Last changed # day ago.} other {Last changed # days ago.}}",
          daysSince(changedAt, new Date(now * 1000)),
        )
    : t("Use a password you don't use anywhere else.");

  const close = () => {
    form.reset();
    setOpen(false);
  };

  const submit = form.handleSubmit(async (values) => {
    try {
      const updated = await apiService.userService.changeMyPassword(values);
      useAuthStore.getState().setUser(updated);
      close();
      feedback.message(t("Password updated"));
    } catch (error) {
      handleMutationError({ error, form, resourceName: "Change Password" });
    }
  });

  return (
    <SettingsRow id="password">
      <FieldWrapper layout="inline" label={t("Password")} description={description}>
        {!open ? (
          <div>
            <Button type="button" variant="outline" size="sm" onClick={() => setOpen(true)}>
              <Key01Icon className="size-3.5" />
              {t("Change password")}
            </Button>
          </div>
        ) : (
          <form
            className="animate-in fade-in-0 slide-in-from-top-1 flex flex-col gap-2.5 duration-300"
            onSubmit={(event) => {
              event.preventDefault();
              if (ready) void submit();
            }}
          >
            <SensitiveField
              control={form.control}
              name="currentPassword"
              label={t("Current password")}
              autoComplete="current-password"
              autoFocus
            />
            <SensitiveField
              control={form.control}
              name="newPassword"
              label={t("New password")}
              autoComplete="new-password"
            />
            <div className="grid grid-cols-4 gap-1" aria-hidden>
              {[1, 2, 3, 4].map((step) => (
                <i
                  key={step}
                  className={cn(
                    "bg-border h-1 rounded-full transition-colors duration-300",
                    step <= score && METER_COLOR[score],
                  )}
                />
              ))}
            </div>
            <ul
              className="flex flex-wrap gap-x-3.5 gap-y-1.5 text-xs"
              aria-label={t("Requirements")}
            >
              {(Object.keys(requirementLabels) as PasswordCheck[]).map((check) => (
                <li
                  key={check}
                  className={cn(
                    "flex items-center gap-1",
                    checks[check] ? "text-success-foreground" : "text-foreground-subtle",
                  )}
                >
                  {checks[check] ? (
                    <CheckIcon className="size-3" />
                  ) : (
                    <XCloseIcon className="size-3" />
                  )}
                  {requirementLabels[check]}
                </li>
              ))}
            </ul>
            <SensitiveField
              control={form.control}
              name="confirmPassword"
              label={t("Confirm new password")}
              autoComplete="new-password"
              description={
                confirmPassword !== "" && !matches
                  ? t("The two new passwords don't match.")
                  : undefined
              }
            />
            <div className="mt-1 flex justify-end gap-2">
              <Button type="button" variant="ghost" size="sm" onClick={close}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={!ready}
                isLoading={form.formState.isSubmitting}
                loadingText={t("Updating...")}
              >
                {t("Update password")}
              </Button>
            </div>
          </form>
        )}
      </FieldWrapper>
    </SettingsRow>
  );
}

function TwoFactorRow() {
  const t = useT();
  const { data: status, isError } = useMFAStatus();
  const [dialog, setDialog] = useState<TwoFactorDialog>(null);
  const [showCodes, setShowCodes] = useState(false);
  const enabled = status?.totpEnabled ?? false;

  return (
    <SettingsRow id="two-factor">
      <FieldWrapper
        layout="inline"
        label={t("Two-factor authentication")}
        description={
          isError
            ? t("Two-factor settings could not be loaded.")
            : enabled
              ? t("Authenticator app. Asks for a code when you sign in.")
              : t("Ask for a code from an authenticator app when you sign in.")
        }
      >
        {!status ? (
          <Skeleton className="h-7 w-48 rounded-md" />
        ) : (
          <div className="flex flex-col gap-2.5">
            <div className="flex flex-wrap items-center gap-2">
              {enabled ? (
                <>
                  <Badge variant="success">{t("On")}</Badge>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    aria-expanded={showCodes}
                    onClick={() => setShowCodes((value) => !value)}
                  >
                    {t("Recovery codes")}
                  </Button>
                  <Button
                    type="button"
                    variant="quiet"
                    size="sm"
                    onClick={() => setDialog("disable")}
                  >
                    {t("Turn off")}
                  </Button>
                </>
              ) : (
                <Button type="button" size="sm" onClick={() => setDialog("setup")}>
                  <ShieldTickIcon className="size-3.5" />
                  {t("Set up")}
                </Button>
              )}
            </div>
            {enabled && showCodes ? (
              <div className="animate-in fade-in-0 slide-in-from-top-1 flex flex-col gap-2 duration-300">
                <p className="text-2xs text-foreground/70">
                  {t(
                    "{0, plural, one {# recovery code left.} other {# recovery codes left.}} Each works once if you lose your phone. They are shown only when they are made, so keep them somewhere safe.",
                    status.recoveryCodesRemaining,
                  )}
                </p>
                <div>
                  <Button
                    type="button"
                    variant="outline"
                    size="xs"
                    onClick={() => setDialog("regenerate")}
                  >
                    {t("Generate new codes")}
                  </Button>
                </div>
              </div>
            ) : null}
          </div>
        )}
      </FieldWrapper>
      <TwoFactorDialogs open={dialog} onClose={() => setDialog(null)} />
    </SettingsRow>
  );
}

const DEVICE_ICON: Record<DeviceKind, IconComponent> = {
  desktop: Laptop01Icon,
  mobile: Phone01Icon,
  tablet: Tablet01Icon,
  unknown: Monitor01Icon,
};

function lastActive(session: SessionSummary, now: number, t: ReturnType<typeof useT>): string {
  if (session.current) return t("Active now");
  return formatRelativeTime(session.lastAccessedAt - now);
}

function SessionsCard() {
  const t = useT();
  const feedback = useSettingsFeedback();
  const queryClient = useQueryClient();
  const now = useNow("minute");
  const { data: sessions, isError, refetch } = useQuery(userQueries.mySessions());
  const [leaving, setLeaving] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const others = sessions?.filter((session) => !session.current) ?? [];

  const removeLater = (ids: string[]) => {
    setLeaving((current) => [...current, ...ids]);
    window.setTimeout(() => {
      queryClient.setQueryData<SessionSummary[]>(
        userQueries.mySessions().queryKey,
        (current) => current?.filter((session) => !ids.includes(session.id)) ?? [],
      );
      setLeaving((current) => current.filter((id) => !ids.includes(id)));
    }, LEAVE_MS);
  };

  const signOut = async (id: string) => {
    setBusy(true);
    try {
      await apiService.userService.revokeMySession(id);
      removeLater([id]);
      feedback.message(t("Session signed out"));
    } catch (error) {
      handleMutationError({ error, resourceName: "Sessions" });
    } finally {
      setBusy(false);
    }
  };

  const signOutOthers = async () => {
    setBusy(true);
    try {
      const { revoked } = await apiService.userService.revokeMyOtherSessions();
      removeLater(others.map((session) => session.id));
      feedback.message(
        t("{0, plural, one {Signed out of # session} other {Signed out of # sessions}}", revoked),
      );
    } catch (error) {
      handleMutationError({ error, resourceName: "Sessions" });
    } finally {
      setBusy(false);
    }
  };

  return (
    <SettingsCard
      id="sessions"
      title={t("Where you're signed in")}
      description={t("Every browser and device with a session open, most recent first.")}
      actions={
        others.length > 0 ? (
          <Button
            type="button"
            variant="quiet"
            size="xs"
            disabled={busy}
            onClick={() => void signOutOthers()}
          >
            <LogOut01Icon className="size-3" />
            {t("Sign out everywhere else")}
          </Button>
        ) : undefined
      }
    >
      {isError ? (
        <div className="flex items-center justify-between gap-3 px-4.5 py-3 text-sm">
          <span>{t("Sessions could not be loaded.")}</span>
          <Button type="button" variant="outline" size="xs" onClick={() => void refetch()}>
            {t("Try again")}
          </Button>
        </div>
      ) : !sessions ? (
        <div className="flex flex-col gap-2 px-4.5 py-3">
          <Skeleton className="h-9 w-full rounded-md" />
          <Skeleton className="h-9 w-full rounded-md" />
        </div>
      ) : (
        sessions.map((session) => {
          const Icon = DEVICE_ICON[session.device];
          return (
            <div
              key={session.id}
              className={cn(
                "flex items-center gap-3.5 px-4.5 py-3 transition-[opacity,translate] duration-300 ease-settle",
                leaving.includes(session.id) && "translate-x-3 opacity-0",
              )}
            >
              <span className="bg-sunken text-muted-foreground grid size-9 shrink-0 place-items-center rounded-lg">
                <Icon className="size-4" />
              </span>
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="flex items-center gap-2 text-sm font-medium">
                  {session.label || t("Unknown browser")}
                  {session.current ? <Badge variant="brand">{t("This device")}</Badge> : null}
                </span>
                <span className="text-foreground-subtle truncate text-xs">
                  {session.ipAddress ? (
                    <>
                      <span className="font-mono">{session.ipAddress}</span>
                      {" · "}
                    </>
                  ) : null}
                  {lastActive(session, now, t)}
                </span>
              </div>
              {!session.current ? (
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  disabled={busy || leaving.includes(session.id)}
                  onClick={() => void signOut(session.id)}
                >
                  {t("Sign out")}
                </Button>
              ) : null}
            </div>
          );
        })
      )}
    </SettingsCard>
  );
}

function signInText(kind: SignInKind, t: ReturnType<typeof useT>): string {
  switch (kind) {
    case "password-and-code":
      return t("Signed in with password + code");
    case "recovery-code":
      return t("Signed in with a recovery code");
    case "password":
      return t("Signed in with password");
    case "code-requested":
      return t("Password accepted, code requested");
    case "single-sign-on":
      return t("Signed in with single sign-on");
    case "wrong-code":
      return t("Wrong verification code");
    case "wrong-password":
      return t("Wrong password");
    case "refused":
      return t("Sign-in refused");
    case "throttled":
      return t("Too many attempts, sign-in paused");
  }
}

function SignInsCard() {
  const t = useT();
  const user = useAuthStore((state) => state.user);
  const timezone = resolveUserTimezone(user?.timezone);
  const { data: signIns, isError } = useQuery(userQueries.mySignIns());

  const where = (signIn: SignInSummary) =>
    [signIn.label, signIn.ipAddress].filter(Boolean).join(" · ");

  return (
    <SettingsCard id="sign-ins" title={t("Recent sign-ins")}>
      {isError ? (
        <p className="px-4.5 py-3 text-sm">{t("Sign-ins could not be loaded.")}</p>
      ) : !signIns ? (
        <div className="flex flex-col gap-2 px-4.5 py-3">
          <Skeleton className="h-4 w-full rounded-md" />
          <Skeleton className="h-4 w-full rounded-md" />
        </div>
      ) : signIns.length === 0 ? (
        <p className="text-foreground-subtle px-4.5 py-3 text-sm">
          {t("No sign-ins recorded yet.")}
        </p>
      ) : (
        <ul className="grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 px-4.5 pt-3 pb-4">
          {signIns.map((signIn) => {
            const { kind, tone } = describeSignIn(signIn);
            return (
              <li key={signIn.id} className="contents">
                <span
                  aria-hidden
                  className={cn(
                    "size-1.75 justify-self-center rounded-full",
                    tone === "ok" && "bg-success",
                    tone === "pending" && "bg-warning",
                    tone === "bad" && "bg-danger",
                  )}
                />
                <span className="min-w-0 truncate text-sm">
                  {signInText(kind, t)}
                  {where(signIn) ? (
                    <span className="text-foreground-subtle ml-2 text-xs">{where(signIn)}</span>
                  ) : null}
                </span>
                <time
                  dateTime={new Date(signIn.occurredAt * 1000).toISOString()}
                  className="text-foreground-subtle font-mono text-xs"
                >
                  {formatUnixInUserTimezone(signIn.occurredAt, {
                    timezone,
                    timeFormat: user?.timeFormat,
                    month: "short",
                    day: "numeric",
                    hour: "numeric",
                    minute: "2-digit",
                  })}
                </time>
              </li>
            );
          })}
        </ul>
      )}
    </SettingsCard>
  );
}

export function SecuritySection() {
  const t = useT();

  return (
    <>
      <SettingsCard title={t("Signing in")}>
        <PasswordRow />
        <TwoFactorRow />
      </SettingsCard>
      <SessionsCard />
      <SignInsCard />
    </>
  );
}
