import { SidebarLayout } from "@/components/navigation";
import { usePermissionPolling } from "@/hooks/use-permission-polling";
import { useRealtimeConnection } from "@/hooks/use-realtime-connection";
import { Outlet } from "react-router";

export function AppLayout() {
  usePermissionPolling();
  useRealtimeConnection();

  return (
    <SidebarLayout>
      <Outlet />
    </SidebarLayout>
  );
}
