/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy, memo } from "react";

const EmailProfileTable = lazy(
  () => import("./_components/email-profile-table"),
);

export function EmailProfiles() {
  return (
    <>
      <MetaTags title="Email Profiles" description="Email Profiles" />
      <Header />
      <QueryLazyComponent queryKey={["email-profile-list"]}>
        <EmailProfileTable />
      </QueryLazyComponent>
    </>
  );
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Email Profiles</h1>
        <p className="text-muted-foreground">
          Manage and track all email profiles in your system
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
