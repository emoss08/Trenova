import { NavigationProgress } from "@/components/navigation-progress";
import { Outlet } from "react-router";

export function RootLayout() {
  return (
    <>
      <NavigationProgress />
      <Outlet />
    </>
  );
}
