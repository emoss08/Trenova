import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";
import { AuthForm } from "./_components/auth-form";

export function AuthPage() {
  const { orgSlug } = useParams();
  const tenantQuery = useQuery({
    ...queries.organization.tenantLogin(orgSlug ?? ""),
    enabled: Boolean(orgSlug),
  });

  return <AuthForm tenantQuery={tenantQuery} organizationSlug={orgSlug} />;
}
