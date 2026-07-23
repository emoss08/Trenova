import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { api, ApiRequestError } from "@/lib/api";
import { useAuthStore } from "@/stores/auth-store";
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
    retry: false,
  });

  return (
    <div className="flex min-h-dvh flex-col justify-center bg-background px-6 text-foreground">
      <m.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, ease: "easeOut" }}
        className="mx-auto w-full max-w-sm"
      >
        {!token || preview.isError ? (
          <InvalidInvitation />
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

function InvalidInvitation() {
  return (
    <div className="text-center">
      <h1 className="text-2xl font-semibold tracking-tight">This invitation isn&apos;t valid</h1>
      <p className="mt-2 text-sm text-muted-foreground">
        The link may have expired or been revoked. Ask your carrier to send a new invitation to get
        set up on Dash.
      </p>
    </div>
  );
}

function AcceptForm({ token, preview }: { token: string; preview: InvitationPreview }) {
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
          Welcome{preview.workerFirstName ? `, ${preview.workerFirstName}` : ""} 👋
        </p>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">
          {preview.organizationName || "Your carrier"} invited you to Dash
        </h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Choose a password to finish setting up your account for{" "}
          <span className="font-medium text-foreground">{preview.email}</span>.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="dash-new-password">Password</Label>
          <Input
            id="dash-new-password"
            type="password"
            autoComplete="new-password"
            placeholder="At least 8 characters"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="dash-confirm-password">Confirm password</Label>
          <Input
            id="dash-confirm-password"
            type="password"
            autoComplete="new-password"
            placeholder="Repeat your password"
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
