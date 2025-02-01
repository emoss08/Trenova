import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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

export function AuthForm() {
  const [formType, setFormType] = useState<AuthFormType>(
    AuthFormType.CHECK_EMAIL,
  );
  const [verifiedEmail, setVerifiedEmail] = useState<string>("");

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
            email={verifiedEmail}
            onForgotPassword={handleForgotPassword}
          />
        );
      case AuthFormType.FORGOT_PASSWORD:
        return (
          <ResetPasswordForm onBack={handleBackToLogin} email={verifiedEmail} />
        );
      default:
        return <CheckEmailForm onEmailVerified={handleEmailVerified} />;
    }
  }

  return (
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
      <CardContent>{renderForm()}</CardContent>
    </Card>
  );
}
