import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { MFAChallenge } from "@trenova/shared/types/mfa";
import { m } from "motion/react";
import { useState } from "react";
import { useNavigate } from "react-router";

export function DashLoginPage() {
  const t = useT();

  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);
  const verifyMFA = useAuthStore((state) => state.verifyMFA);
  const [emailAddress, setEmailAddress] = useState("");
  const [password, setPassword] = useState("");
  const [challenge, setChallenge] = useState<MFAChallenge | null>(null);
  const [code, setCode] = useState("");
  const [useRecovery, setUseRecovery] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!emailAddress || !password) {
      setError(t("Enter your email and password."));
      return;
    }
    setPending(true);
    setError(null);
    try {
      const outcome = await login({ emailAddress, password });
      if (outcome.status === "mfa_required") {
        setChallenge(outcome.challenge);
        return;
      }
      void navigate("/dash", { replace: true });
    } catch {
      setError(t("We couldn't sign you in. Check your email and password and try again."));
    } finally {
      setPending(false);
    }
  };

  const handleVerify = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!challenge || !code.trim()) {
      setError(t("Enter the code from your authenticator app."));
      return;
    }
    setPending(true);
    setError(null);
    try {
      await verifyMFA({
        challengeToken: challenge.mfaChallengeToken,
        code: useRecovery ? "" : code,
        recoveryCode: useRecovery ? code : "",
      });
      void navigate("/dash", { replace: true });
    } catch {
      setError(t("That code didn't work. Check it and try again, or sign in again."));
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="flex min-h-dvh flex-col justify-center bg-background px-6 text-foreground">
      <m.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, ease: "easeOut" }}
        className="mx-auto w-full max-w-sm"
      >
        <div className="mb-8">
          <h1 className="text-3xl font-semibold tracking-tight">{t("Dash")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("Your loads, settlements, and pay — in one place.")}
          </p>
        </div>

        {challenge ? (
          <form onSubmit={handleVerify} className="flex flex-col gap-4" noValidate>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="dash-code">
                {useRecovery ? t("Recovery code") : t("Authentication code")}
              </Label>
              <Input
                id="dash-code"
                autoComplete="one-time-code"
                inputMode={useRecovery ? "text" : "numeric"}
                placeholder={useRecovery ? "xxxxx-xxxxx" : "123 456"}
                value={code}
                onChange={(event) => setCode(event.target.value)}
              />
            </div>

            {error ? <p className="text-sm text-destructive">{error}</p> : null}

            <Button type="submit" className="mt-2 h-11 w-full" disabled={pending}>
              {pending ? t("Verifying...") : t("Verify")}
            </Button>
            <div className="flex justify-between text-xs text-muted-foreground">
              <button
                type="button"
                className="underline underline-offset-4"
                onClick={() => {
                  setCode("");
                  setUseRecovery((current) => !current);
                }}
              >
                {useRecovery ? t("Use your authenticator app") : t("Use a recovery code")}
              </button>
              <button
                type="button"
                className="underline underline-offset-4"
                onClick={() => {
                  setChallenge(null);
                  setCode("");
                  setPassword("");
                }}
              >
                {t("Start over")}
              </button>
            </div>
          </form>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="dash-email">{t("Email")}</Label>
              <Input
                id="dash-email"
                type="email"
                autoComplete="email"
                inputMode="email"
                placeholder={t("you@example.com")}
                value={emailAddress}
                onChange={(event) => setEmailAddress(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="dash-password">{t("Password")}</Label>
              <Input
                id="dash-password"
                type="password"
                autoComplete="current-password"
                placeholder="••••••••"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>

            {error ? <p className="text-sm text-destructive">{error}</p> : null}

            <Button type="submit" className="mt-2 h-11 w-full" disabled={pending}>
              {pending ? t("Signing in...") : t("Sign in")}
            </Button>
          </form>
        )}

        <p className="mt-6 text-center text-xs text-muted-foreground">
          {t("No account yet? Ask your carrier to send you a Dash invitation.")}
        </p>
        <p className="mt-2 text-center text-xs text-muted-foreground">
          {t("Office or dispatch?")}{" "}
          <a href="/login" className="text-foreground underline underline-offset-4">
            {t("Sign in to Trenova")}
          </a>
        </p>
      </m.div>
    </div>
  );
}
