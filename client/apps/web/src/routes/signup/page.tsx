import { Metadata } from "@/components/metadata";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { AuthCard } from "../auth/_components/auth-card";
import type { CredentialReceipt } from "../auth/_components/auth-panel";
import { AuthMobileBrand } from "../auth/_components/auth-primitives";
import { AuthShell } from "../auth/_components/auth-shell";
import { SignupForm } from "./_components/signup-form";
import { SignupSent } from "./_components/signup-sent";

type Submission = {
  emailAddress: string;
  companyName: string;
};

/**
 * Trenova Cloud's self-serve signup. Nothing is created here: the server holds the
 * request until the emailed link is opened, so this page ends at "check your inbox".
 * The route loader keeps it off every install that is not cloud with signup on.
 */
export function SignupPage() {
  const t = useT();
  const { config } = usePublicConfig();
  const [submission, setSubmission] = useState<Submission | null>(null);

  const receipt: CredentialReceipt = {
    issued: false,
    rows: [
      { key: "Identity", value: submission?.emailAddress },
      { key: "Workspace", value: submission?.companyName },
      { key: "Verification", value: submission ? "Link sent" : undefined },
      { key: "Plan", value: "Free demo" },
    ],
  };

  return (
    <>
      <Metadata
        title={t("Create your account")}
        description={t("Start a free Trenova Cloud demo")}
      />
      <AuthShell step={submission ? "done" : "login"} receipt={receipt}>
        <AuthMobileBrand />
        <AuthCard stepKey={submission ? "sent" : "signup"}>
          {submission ? (
            <SignupSent
              emailAddress={submission.emailAddress}
              turnstileSiteKey={config.turnstileSiteKey}
              onStartOver={() => setSubmission(null)}
            />
          ) : (
            <SignupForm turnstileSiteKey={config.turnstileSiteKey} onSubmitted={setSubmission} />
          )}
        </AuthCard>
      </AuthShell>
    </>
  );
}
