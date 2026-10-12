import { useT } from "@trenova/shared/i18n/use-t";
import { Metadata } from "@/components/metadata";
import { useCallback, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";
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

  return (
    <>
      <Metadata title={t("Reset password")} description={t("Choose a new Trenova password")} />
      <AuthShell screenKey={isDone ? "done" : "reset"}>
        {isDone ? (
          <ResetPasswordDone onSignIn={goToSignIn} />
        ) : (
          <ResetPasswordForm
            token={token}
            onDone={() => setIsDone(true)}
            onRequestNewLink={goToSignIn}
          />
        )}
      </AuthShell>
    </>
  );
}
