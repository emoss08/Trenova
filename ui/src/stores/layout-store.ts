/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
