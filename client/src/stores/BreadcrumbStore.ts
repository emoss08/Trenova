import { createGlobalStore } from "@/lib/useGlobalStore";
import { RouteObjectWithPermission } from "@/routing/AppRoutes";

interface BreadcrumbStoreType {
  currentRoute: RouteObjectWithPermission | null;
  loading: boolean;
}

export const useBreadcrumbStore = createGlobalStore<BreadcrumbStoreType>({
  currentRoute: {
    title: "",
    group: "",
    subMenu: "",
    path: "",
    isPublic: false,
  },
  loading: false,
});
