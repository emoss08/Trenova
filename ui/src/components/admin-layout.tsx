/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { adminLinks } from "@/lib/nav-links";
import { Outlet } from "react-router";
import { LazyComponent } from "./error-boundary";
import { SidebarNav } from "./sidebar-nav";

export function AdminLayout() {
  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[240px_minmax(0,1fr)] lg:gap-10">
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
