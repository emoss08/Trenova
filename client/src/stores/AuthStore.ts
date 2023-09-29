/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { create, SetState, StateCreator } from "zustand";
import { persist } from "zustand/middleware";
import { createGlobalStore } from "@/lib/useGlobalStore";

type AuthState = {
  isAuthenticated: boolean;
  setIsAuthenticated: (isAuthenticated: boolean) => void;
  loading: boolean;
  setLoading: (loading: boolean) => void;
  initialLoading: boolean;
  setInitialLoading: (initialLoading: boolean) => void;
  reset: () => void;
};

const createStore = (set: SetState<AuthState>) => ({
  isAuthenticated: false,
  setIsAuthenticated: (isAuthenticated: boolean) => set({ isAuthenticated }),
  loading: false,
  setLoading: (loading: boolean) => set({ loading }),
  initialLoading: true,
  setInitialLoading: (initialLoading: boolean) => set({ initialLoading }),
  reset: () => set({ isAuthenticated: false }),
});

// TODO(WOLFRED): Switch this to createGlobalStore once we have a way to persist global stores
export const useAuthStore = create<AuthState>(
  persist(createStore, {
    name: "auth-storage",
  }) as StateCreator<AuthState>,
);

type UserState = {
  userId: string;
  organizationId: string;
  userPermissions: string[];
  userGroups: string[];
  userIsStaff: boolean;
};

type UserStoreState = {
  user: UserState;
};

export const useUserStore = createGlobalStore<UserStoreState>({
  user: {
    userId: "",
    organizationId: "",
    userPermissions: [],
    userGroups: [],
    userIsStaff: false,
  },
});
