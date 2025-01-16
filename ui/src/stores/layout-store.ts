import { createGlobalStore } from "@/hooks/use-global-store";

type LayoutStoreProps = {
  notificationMenuOpen: boolean;
  searchDialogOpen: boolean;
  signOutDialogOpen: boolean;
};

export const useLayoutStore = createGlobalStore<LayoutStoreProps>({
  notificationMenuOpen: false,
  searchDialogOpen: false,
  signOutDialogOpen: false,
});
