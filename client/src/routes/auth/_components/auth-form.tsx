import { Metadata } from "@/components/metadata";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { TenantLoginMetadata } from "@/types/organization";
import type { UseQueryResult } from "@tanstack/react-query";
import { AnimatePresence, m } from "motion/react";
import { useState } from "react";
import { Link } from "react-router";
import { LoginForm } from "./login-form";

type AuthFormType = "LOGIN" | "FORGOT_PASSWORD";

function renderForm({
  formType,
  organizationSlug,
  tenantMetadata,
}: {
  formType: AuthFormType;
  organizationSlug?: string;
  tenantMetadata?: TenantLoginMetadata;
}) {
  switch (formType) {
    case "LOGIN":
      return <LoginForm organizationSlug={organizationSlug} tenantMetadata={tenantMetadata} />;
    case "FORGOT_PASSWORD":
      return <div>Coming soon</div>;
    default:
      return <LoginForm organizationSlug={organizationSlug} tenantMetadata={tenantMetadata} />;
  }
}

export function AuthForm({
  tenantQuery,
  organizationSlug,
}: {
  tenantQuery?: UseQueryResult<TenantLoginMetadata>;
  organizationSlug?: string;
}) {
  const [formType] = useState<AuthFormType>("LOGIN");
  const tenantMetadata = tenantQuery?.data;
  const subtitle = tenantMetadata
    ? `Sign in to ${tenantMetadata.organizationName}`
    : "Don't have an account yet?";

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
                  : tenantMetadata
                    ? tenantMetadata.organizationName
                    : "Welcome back!"}
              </CardTitle>
              <CardDescription className="mt-1 flex space-x-1 text-sm">
                <span className="text-muted-foreground">{subtitle}</span>
                {!tenantMetadata && (
                  <Link className="text-primary underline" to="#">
                    Create an Account
                  </Link>
                )}
              </CardDescription>
            </m.div>
          </CardHeader>
          <CardContent>
            {tenantQuery?.isLoading ? (
              <div className="text-sm text-muted-foreground">Loading organization sign-in...</div>
            ) : tenantQuery?.isError ? (
              <div className="text-sm text-destructive">
                We couldn&apos;t load this tenant login page.
              </div>
            ) : (
              <AnimatePresence mode="wait">
                <m.div
                  key={formType}
                  initial={{ opacity: 0, y: 8, scale: 0.98 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: -8, scale: 0.98 }}
                  transition={{ duration: 0.22, ease: "easeOut" }}
                >
                  {renderForm({ formType, organizationSlug, tenantMetadata })}
                </m.div>
              </AnimatePresence>
            )}
          </CardContent>
        </Card>
        <div className="text-center text-xs text-balance text-muted-foreground [&_a]:underline [&_a]:underline-offset-4 [&_a]:hover:text-primary">
          By clicking continue, you agree to our <a href="#">Terms of Service</a> and{" "}
          <span className="cursor-pointer underline underline-offset-4 hover:text-primary">
            License Agreement
          </span>
          .
        </div>
      </div>
    </>
  );
}
