import { createGlobalStore } from "@/lib/useGlobalStore";

type HeaderStoreProps = {
  menuOpen?: string;
  notificationMenuOpen: boolean;
  searchDialogOpen: boolean;
  asideMenuOpen: boolean;
};

export const useHeaderStore = createGlobalStore<HeaderStoreProps>({
  menuOpen: undefined,
  notificationMenuOpen: false,
  searchDialogOpen: false,
  asideMenuOpen: false,
});
