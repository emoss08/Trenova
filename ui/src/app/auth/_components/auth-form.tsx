import { LicenseInformation } from "@/components/license-information";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { AnimatePresence, motion } from "motion/react";
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

  const [licenseDialogOpen, setLicenseDialogOpen] = useState(false);

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
      <div className="flex max-w-[400px] flex-col gap-6">
        <Card className="rounded-2xl border-border bg-background backdrop-blur-md">
          <CardHeader className="text-left">
            <motion.div
              key={
                formType === AuthFormType.FORGOT_PASSWORD ? "reset" : "login"
              }
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
            >
              <CardTitle>
                {formType === AuthFormType.FORGOT_PASSWORD
                  ? "Reset Password"
                  : "Welcome back!"}
              </CardTitle>
              <CardDescription className="mt-1 flex space-x-1 text-sm">
                <span className="text-muted-foreground">
                  Don&apos;t have an account yet?
                </span>
                <Link className="text-primary underline" to="#">
                  Create an Account
                </Link>
              </CardDescription>
            </motion.div>
          </CardHeader>
          <CardContent className="px-6 py-4">
            <AnimatePresence mode="wait">
              <motion.div
                key={formType}
                initial={{ opacity: 0, y: 8, scale: 0.98 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -8, scale: 0.98 }}
                transition={{ duration: 0.22, ease: "easeOut" }}
              >
                {renderForm()}
              </motion.div>
            </AnimatePresence>
          </CardContent>
        </Card>
        <div className="text-center text-xs text-balance text-muted-foreground [&_a]:underline [&_a]:underline-offset-4 [&_a]:hover:text-primary">
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
      {licenseDialogOpen && (
        <LicenseInformation
          open={licenseDialogOpen}
          onOpenChange={setLicenseDialogOpen}
        />
      )}
    </>
  );
}
