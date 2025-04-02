import { LicenseInformation } from "@/components/license-information";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { licenseDialog } from "@/lib/license";
import { parseAsString, useQueryState } from "nuqs";
import { useState } from "react";
import { Link } from "react-router";
import { CheckEmailForm } from "./check-email-form";
import { LoginForm } from "./login-form";
import { ResetPasswordForm } from "./reset-password-form";

const enum AuthFormType {
  CHECK_EMAIL = "CHECK_EMAIL",
  LOGIN = "LOGIN",
  FORGOT_PASSWORD = "FORGOT_PASSWORD",
}

const searchParams = {
  verifiedEmail: parseAsString,
};

export function AuthForm() {
  const [formType, setFormType] = useState<AuthFormType>(
    AuthFormType.CHECK_EMAIL,
  );

  const [licenseDialogOpen, setLicenseDialogOpen] = useQueryState(
    "licenseDialogOpen",
    licenseDialog.licenseDialogOpen.withOptions({}),
  );

  const [verifiedEmail, setVerifiedEmail] = useQueryState(
    "verifiedEmail",
    searchParams.verifiedEmail.withOptions({}),
  );

  function handleEmailVerified(email: string) {
    setVerifiedEmail(email);
    setFormType(AuthFormType.LOGIN);
  }

  function handleForgotPassword() {
    setFormType(AuthFormType.FORGOT_PASSWORD);
  }

  function handleBackToLogin() {
    setFormType(AuthFormType.LOGIN);
  }

  function renderForm() {
    switch (formType) {
      case AuthFormType.LOGIN:
        return (
          <LoginForm
            email={verifiedEmail ?? ""}
            onForgotPassword={handleForgotPassword}
          />
        );
      case AuthFormType.FORGOT_PASSWORD:
        return (
          <ResetPasswordForm
            onBack={handleBackToLogin}
            email={verifiedEmail ?? ""}
          />
        );
      default:
        return <CheckEmailForm onEmailVerified={handleEmailVerified} />;
    }
  }

  return (
    <>
      <div className="flex flex-col gap-6">
        <Card className="mx-auto w-[400px] border-input">
          <CardHeader className="text-left">
            <CardTitle className="text-xl font-bold">
              {formType === AuthFormType.FORGOT_PASSWORD
                ? "Reset Password"
                : "Welcome back!"}
            </CardTitle>
            <CardDescription className="flex space-x-1 text-sm">
              <span className="text-muted-foreground">
                Don&apos;t have an account yet?
              </span>
              <Link className="text-primary underline" to="#">
                Create an Account
              </Link>
            </CardDescription>
          </CardHeader>
          <CardContent className="px-6 py-4">{renderForm()}</CardContent>
        </Card>
        <div className="text-balance text-center text-xs text-muted-foreground [&_a]:underline [&_a]:underline-offset-4 [&_a]:hover:text-primary">
          By clicking continue, you agree to our{" "}
          <a href="#">Terms of Service</a> and{" "}
          <span
            className="cursor-pointer underline underline-offset-4 hover:text-primary"
            onClick={() => setLicenseDialogOpen(true)}
          >
            License Agreement
          </span>
          .
        </div>
      </div>
      <LicenseInformation />
    </>
  );
}
