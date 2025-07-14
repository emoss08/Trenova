import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

const EmailProfileTable = lazy(
  () => import("./_components/email-profile-table"),
);

export function EmailProfiles() {
  return (
    <>
      <MetaTags title="Email Profiles" description="Email Profiles" />
      <QueryLazyComponent queryKey={["email-profile-list"]}>
        <EmailProfileTable />
      </QueryLazyComponent>
    </>
  );
}
