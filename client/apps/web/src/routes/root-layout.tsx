import { NavigationProgress } from "@/components/navigation-progress";
import { PlanLimitDialogHost } from "@/components/plan-limit/plan-limit-dialog";
import { Outlet } from "react-router";

export function RootLayout() {
  return (
    <>
      <NavigationProgress />
      <Outlet />
      <PlanLimitDialogHost />
    </>
  );
}
