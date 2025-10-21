import { QueryLazyComponent } from "@/components/error-boundary";
import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";

const OrganizationForm = lazy(() => import("./_components/organization-form"));

export function OrganizationSettings() {
  return (
    <>
      <MetaTags title="Organization" description="Organization" />
      <div className="flex flex-col gap-y-3">
        <Header />
        <QueryLazyComponent queryKey={queries.organization.getOrgById._def}>
          <FormSaveProvider>
            <OrganizationForm />
          </FormSaveProvider>
        </QueryLazyComponent>
      </div>
    </>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">
          Organization Settings
        </h1>
        <p className="text-muted-foreground">
          Configure and manage your organization settings
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
