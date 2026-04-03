import { LazyComponent } from "@/components/error-boundary";
import { SidebarNav } from "@/components/sidebar-nav";
import { adminLinks } from "@/config/navigation.config";
import { Outlet } from "react-router";

export function AdminLayout() {
  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[180px_minmax(0,1fr)] lg:grid-cols-[220px_minmax(0,1fr)]">
      <SidebarNav links={adminLinks} />
      <div className="relative lg:gap-10">
        <div className="mx-auto min-w-0">
          <LazyComponent>
            <Outlet />
          </LazyComponent>
        </div>
      </div>
    </div>
  );
}
