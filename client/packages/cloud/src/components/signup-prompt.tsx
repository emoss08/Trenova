import type { LoginPromptProps } from "@trenova/edition";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";
import { SIGNUP_PATH } from "../lib/signup-gate";

/**
 * The line under the sign-in heading. Self-serve signup exists only on Trenova Cloud
 * with signup switched on; anywhere else accounts are made by an administrator, so
 * the host's own line stands and no link leads nowhere.
 */
export function SignupPrompt({ fallback }: LoginPromptProps) {
  const t = useT();
  const { signupAvailable } = usePublicConfig();

  if (!signupAvailable) {
    return <>{fallback}</>;
  }

  return (
    <>
      {t("Don't have an account yet?")}{" "}
      <Link
        to={SIGNUP_PATH}
        className="text-foreground decoration-foreground/35 hover:decoration-foreground underline underline-offset-[3px]"
      >
        {t("Create an account")}
      </Link>
    </>
  );
}
