import { NavigationProgress } from "@/components/navigation-progress";
import { PlanLimitDialogHost } from "@/components/plan-limit/plan-limit-dialog";
import { OrganizationChangedBanner } from "@/components/session-context/organization-changed-banner";
import { OrganizationChangedDialogHost } from "@/components/session-context/organization-changed-dialog";
import { edition } from "@/lib/edition";
import { Outlet } from "react-router";

const EditionRootHost = edition.slots.RootHost;

export function RootLayout() {
  return (
    <>
      <NavigationProgress />
      <Outlet />
      <PlanLimitDialogHost />
      <OrganizationChangedDialogHost />
      <OrganizationChangedBanner />
      <EditionRootHost />
    </>
  );
}
