import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { Outlet } from "react-router";

export function AdminLayout() {
  return (
    <div className="flex-1">
      <div className="mx-auto min-w-0">
        <LazyComponent>
          <Outlet />
        </LazyComponent>
      </div>
    </div>
  );
}
