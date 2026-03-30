import logoRainbow from "@/assets/logo.webp";
import { LazyImage } from "@/components/image";
import { Metadata } from "@/components/metadata";
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

  return (
    <>
      <Metadata title="Sign In" description="Sign in to your Trenova account" />
      <div className="fixed inset-0 h-svh w-full overflow-hidden bg-background">
        <div className="relative flex h-full flex-col items-center justify-center gap-6 p-6 md:p-10">
          <LazyImage
            src={logoRainbow}
            alt="Trenova Logo"
            className="size-14 object-contain drop-shadow-[0_4px_24px_rgba(255,255,255,0.25)]"
          />
          <AuthForm tenantQuery={tenantQuery} organizationSlug={orgSlug} />
        </div>
      </div>
    </>
  );
}
