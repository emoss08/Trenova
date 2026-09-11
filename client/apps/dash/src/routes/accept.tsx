import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { api, ApiRequestError } from "@trenova/shared/lib/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";
import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

type InvitationPreview = {
  organizationName: string;
  workerFirstName: string;
  email: string;
  expiresAt: number;
};

type AcceptResult = {
  emailAddress: string;
  organizationName: string;
};

export function DashAcceptPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token") ?? "";

  const preview = useQuery({
    queryKey: ["dash-invitation-preview", token],
    queryFn: () =>
      api.get<InvitationPreview>(`/portal/invitations/preview?token=${encodeURIComponent(token)}`),
    enabled: token.length > 0,
    retry: (failureCount, error) => !isRejectedInvitation(error) && failureCount < 2,
  });

  const rejected = !token || (preview.isError && isRejectedInvitation(preview.error));

  return (
    <div className="flex min-h-dvh flex-col justify-center bg-background px-6 text-foreground">
      <m.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, ease: "easeOut" }}
        className="mx-auto w-full max-w-sm"
      >
        {rejected ? (
          <InvalidInvitation />
        ) : preview.isError ? (
          <UnreachableInvitation
            onRetry={() => void preview.refetch()}
            retrying={preview.isFetching}
          />
        ) : preview.isPending ? (
          <div className="flex justify-center py-16">
            <Spinner className="size-6" />
          </div>
        ) : (
          <AcceptForm token={token} preview={preview.data} />
        )}
      </m.div>
    </div>
  );
}

// The server answers a bad, revoked, or expired token with a 4xx problem
// document. Anything else - the API being down, a network drop, a blocked
// cross-origin request - is a transport failure, and telling the driver their
// invitation is dead would be wrong.
const REJECTED_INVITATION_STATUSES = new Set([400, 404, 410, 422]);

function isRejectedInvitation(error: unknown): boolean {
  return error instanceof ApiRequestError && REJECTED_INVITATION_STATUSES.has(error.status);
}

function UnreachableInvitation({ onRetry, retrying }: { onRetry: () => void; retrying: boolean }) {
  const t = useT();

  return (
    <div className="text-center">
      <h1 className="text-2xl font-semibold tracking-tight">
        {t("We couldn't load your invitation")}
      </h1>
      <p className="mt-2 text-sm text-muted-foreground">
        {t("Your invitation link looks fine, but we couldn't reach Dash just now. Check your connection and try again.")}
      </p>
      <Button className="mt-6 h-11 w-full" onClick={onRetry} disabled={retrying}>
        {retrying ? "Retrying..." : "Try again"}
      </Button>
    </div>
  );
}

function InvalidInvitation() {
  const t = useT();

  return (
    <div className="text-center">
      <h1 className="text-2xl font-semibold tracking-tight">{t("This invitation isn't valid")}</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        {t("The link may have expired or been revoked. Ask your carrier to send a new invitation to get set up on Dash.")}
      </p>
    </div>
  );
}

function AcceptForm({ token, preview }: { token: string; preview: InvitationPreview }) {
  const t = useT();

  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (password.length < 8) {
      setError("Your password must be at least 8 characters.");
      return;
    }
    if (password !== confirmPassword) {
      setError("Passwords don't match.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      const result = await api.post<AcceptResult>("/portal/invitations/accept", {
        token,
        password,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      });
      await login({ emailAddress: result.emailAddress, password });
      void navigate("/dash", { replace: true });
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(err.message);
      } else {
        setError("Something went wrong setting up your account. Try again.");
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <div>
      <div className="mb-8">
        <p className="text-sm text-muted-foreground">
          {t("Welcome{0} 👋", preview.workerFirstName ? `, ${preview.workerFirstName}` : "")}
        </p>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">
          {t("{0} invited you to Dash", preview.organizationName || "Your carrier")}
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          {t("Choose a password to finish setting up your account for")}{" "}
          <span className="font-medium text-foreground">{preview.email}</span>.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="dash-new-password">{t("Password")}</Label>
          <Input
            id="dash-new-password"
            type="password"
            autoComplete="new-password"
            placeholder={t("At least 8 characters")}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="dash-confirm-password">{t("Confirm password")}</Label>
          <Input
            id="dash-confirm-password"
            type="password"
            autoComplete="new-password"
            placeholder={t("Repeat your password")}
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
          />
        </div>

        {error ? <p className="text-sm text-destructive">{error}</p> : null}

        <Button type="submit" className="mt-2 h-11 w-full" disabled={pending}>
          {pending ? "Setting up..." : "Create account & sign in"}
        </Button>
      </form>
    </div>
  );
}
