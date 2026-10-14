import { organization } from "@/lib/queries/organization";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";
import { AuthForm } from "./_components/auth-form";

export function AuthPage() {
  const { orgSlug } = useParams();
  const tenantQuery = useQuery({
    ...organization.tenantLogin(orgSlug ?? ""),
    enabled: Boolean(orgSlug),
  });

  return <AuthForm tenantQuery={tenantQuery} organizationSlug={orgSlug} />;
}
