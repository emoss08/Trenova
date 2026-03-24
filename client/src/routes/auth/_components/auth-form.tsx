import { Metadata } from "@/components/metadata";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { AnimatePresence, m } from "motion/react";
import { useState } from "react";
import { Link } from "react-router";
import { LoginForm } from "./login-form";

type AuthFormType = "LOGIN" | "FORGOT_PASSWORD";

function renderForm({ formType }: { formType: AuthFormType }) {
  switch (formType) {
    case "LOGIN":
      return <LoginForm />;
    case "FORGOT_PASSWORD":
      return <div>Coming soon</div>;
    default:
      return <LoginForm />;
  }
}

export function AuthForm() {
  const [formType] = useState<AuthFormType>("LOGIN");

  return (
    <>
      <Metadata title="Sign In" description="Sign in to your Trenova account" />
      <div className="flex max-w-[400px] flex-col gap-6">
        <Card className="rounded-2xl border-border bg-background backdrop-blur-md">
          <CardHeader className="text-left">
            <m.div
              key={formType === "FORGOT_PASSWORD" ? "reset" : "login"}
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
            >
              <CardTitle>
                {formType === "FORGOT_PASSWORD"
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
            </m.div>
          </CardHeader>
          <CardContent>
            <AnimatePresence mode="wait">
              <m.div
                key={formType}
                initial={{ opacity: 0, y: 8, scale: 0.98 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -8, scale: 0.98 }}
                transition={{ duration: 0.22, ease: "easeOut" }}
              >
                {renderForm({ formType })}
              </m.div>
            </AnimatePresence>
          </CardContent>
        </Card>
        <div className="text-center text-xs text-balance text-muted-foreground [&_a]:underline [&_a]:underline-offset-4 [&_a]:hover:text-primary">
          By clicking continue, you agree to our{" "}
          <a href="#">Terms of Service</a> and{" "}
          <span className="cursor-pointer underline underline-offset-4 hover:text-primary">
            License Agreement
          </span>
          .
        </div>
      </div>
    </>
  );
}
