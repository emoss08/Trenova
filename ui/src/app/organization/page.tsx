import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { lazy } from "react";

const OrganizationForm = lazy(() => import("./_components/organization-form"));

export function OrganizationSettings() {
  return (
    <>
      <MetaTags title="Organization" description="Organization" />
      <SuspenseLoader>
        <FormSaveProvider>
          <OrganizationForm />
        </FormSaveProvider>
      </SuspenseLoader>
    </>
  );
}
