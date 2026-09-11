import { useT } from "@trenova/shared/i18n/use-t";
import logoRainbow from "@/assets/logo.webp";
import { Metadata } from "@/components/metadata";
import { PRIVACY_URL, TERMS_URL } from "@trenova/shared/lib/constants";
import { useCallback, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { AuthCard } from "./_components/auth-card";
import type { CredentialReceipt } from "./_components/auth-panel";
import { AuthShell } from "./_components/auth-shell";
import { ResetPasswordDone, ResetPasswordForm } from "./_components/reset-password-form";

/**
 * The page the emailed link opens. It is reached without a session and knows nothing
 * about the account behind the token — deliberately, since anything it displayed about
 * the user would be readable by whoever holds the link.
 */
export function ResetPasswordPage() {
  const t = useT();

  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const [isDone, setIsDone] = useState(false);

  const goToSignIn = useCallback(() => {
    void navigate("/login", { replace: true });
  }, [navigate]);

  // The receipt has no identity to show here, so it tracks the recovery itself: the
  // link arrived, then the password was set.
  const receipt: CredentialReceipt = {
    issued: isDone,
    rows: [
      { key: "Link", value: token ? "Received" : undefined },
      { key: "Password", value: isDone ? "Updated" : undefined },
    ],
  };

  return (
    <>
      <Metadata title={t("Reset password")} description={t("Choose a new Trenova password")} />
      <AuthShell step={isDone ? "done" : "login"} receipt={receipt}>
        <div className="mb-1 flex items-center justify-center gap-2.5 min-[900px]:hidden">
          <img src={logoRainbow} alt="" className="size-6 object-contain" />
          <span className="text-[14px] font-semibold tracking-[-0.02em]">{t("Trenova")}</span>
        </div>

        <AuthCard stepKey={isDone ? "done" : "reset"}>
          {isDone ? (
            <ResetPasswordDone onSignIn={goToSignIn} />
          ) : (
            <ResetPasswordForm
              token={token}
              onDone={() => setIsDone(true)}
              onRequestNewLink={goToSignIn}
            />
          )}
        </AuthCard>

        <p className="text-subtle-foreground m-0 text-center text-[11.5px] text-balance">
          {t("By continuing you agree to our")}{" "}
          <a
            href={TERMS_URL}
            target="_blank"
            rel="noreferrer"
            className="text-muted-foreground hover:text-foreground underline underline-offset-[3px]"
          >
            {t("Terms of Service")}
          </a>{" "}
          and{" "}
          <a
            href={PRIVACY_URL}
            target="_blank"
            rel="noreferrer"
            className="text-muted-foreground hover:text-foreground underline underline-offset-[3px]"
          >
            {t("Privacy Policy")}
          </a>
          .
        </p>
      </AuthShell>
    </>
  );
}
