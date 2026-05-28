import { Metadata } from "@/components/metadata";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { PRIVACY_URL, TERMS_URL } from "@/lib/constants";
import type { TenantLoginMetadata, UserOrganization } from "@/types/organization";
import type { UseQueryResult } from "@tanstack/react-query";
import { AnimatePresence, m } from "motion/react";
import { useState } from "react";
import { Link } from "react-router";
import { LoginForm } from "./login-form";
import { OrganizationSelection } from "./organization-selection";

type AuthFormType = "LOGIN" | "FORGOT_PASSWORD";
type AuthStep = "LOGIN" | "ORGANIZATION";

function renderForm({
  authStep,
  formType,
  organizationSlug,
  tenantMetadata,
  selectableOrganizations,
  onOrganizationSelectionRequired,
}: {
  authStep: AuthStep;
  formType: AuthFormType;
  organizationSlug?: string;
  tenantMetadata?: TenantLoginMetadata;
  selectableOrganizations: UserOrganization[];
  onOrganizationSelectionRequired: (organizations: UserOrganization[]) => void;
}) {
  if (authStep === "ORGANIZATION") {
    return <OrganizationSelection organizations={selectableOrganizations} />;
  }

  switch (formType) {
    case "LOGIN":
      return (
        <LoginForm
          organizationSlug={organizationSlug}
          tenantMetadata={tenantMetadata}
          onOrganizationSelectionRequired={onOrganizationSelectionRequired}
        />
      );
    case "FORGOT_PASSWORD":
      return <div>Coming soon</div>;
    default:
      return (
        <LoginForm
          organizationSlug={organizationSlug}
          tenantMetadata={tenantMetadata}
          onOrganizationSelectionRequired={onOrganizationSelectionRequired}
        />
      );
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
  const [authStep, setAuthStep] = useState<AuthStep>("LOGIN");
  const [selectableOrganizations, setSelectableOrganizations] = useState<UserOrganization[]>([]);
  const tenantMetadata = tenantQuery?.data;
  const isOrganizationStep = authStep === "ORGANIZATION";
  const title = isOrganizationStep
    ? "Select organization"
    : formType === "FORGOT_PASSWORD"
      ? "Reset Password"
      : tenantMetadata
        ? tenantMetadata.organizationName
        : "Welcome back!";
  const subtitle = isOrganizationStep
    ? "Choose the workspace for this session."
    : tenantMetadata
      ? `Sign in to ${tenantMetadata.organizationName}`
      : "Don't have an account yet?";

  const handleOrganizationSelectionRequired = (organizations: UserOrganization[]) => {
    setSelectableOrganizations(organizations);
    setAuthStep("ORGANIZATION");
  };

  return (
    <>
      <Metadata title="Sign In" description="Sign in to your Trenova account" />
      <div className="flex max-w-[400px] flex-col gap-6">
        <Card className="rounded-2xl border-border bg-background backdrop-blur-md">
          <CardHeader className="text-left">
            <m.div
              key={`${authStep}-${formType}`}
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
            >
              <CardTitle>{title}</CardTitle>
              <CardDescription className="mt-1 flex space-x-1 text-sm">
                <span className="text-muted-foreground">{subtitle}</span>
                {!tenantMetadata && !isOrganizationStep && (
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
                  key={`${authStep}-${formType}`}
                  initial={{ opacity: 0, y: 8, scale: 0.98 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: -8, scale: 0.98 }}
                  transition={{ duration: 0.22, ease: "easeOut" }}
                >
                  {renderForm({
                    authStep,
                    formType,
                    organizationSlug,
                    tenantMetadata,
                    selectableOrganizations,
                    onOrganizationSelectionRequired: handleOrganizationSelectionRequired,
                  })}
                </m.div>
              </AnimatePresence>
            )}
          </CardContent>
        </Card>
        <div className="text-center text-xs text-balance text-muted-foreground [&_a]:underline [&_a]:underline-offset-4 [&_a]:hover:text-primary">
          By clicking continue, you agree to our{" "}
          <a href={TERMS_URL} target="_blank" rel="noreferrer">
            Terms of Service
          </a>{" "}
          and{" "}
          <a href={PRIVACY_URL} target="_blank" rel="noreferrer">
            Privacy Policy
          </a>
          .
        </div>
      </div>
    </>
  );
}
