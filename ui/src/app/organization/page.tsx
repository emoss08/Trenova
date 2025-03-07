import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { lazy, memo } from "react";

const OrganizationForm = lazy(() => import("./_components/organization-form"));

export function OrganizationSettings() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags title="Organization" description="Organization" />
      <Header />
      <SuspenseLoader>
        <FormSaveProvider>
          <OrganizationForm />
        </FormSaveProvider>
      </SuspenseLoader>
    </div>
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
