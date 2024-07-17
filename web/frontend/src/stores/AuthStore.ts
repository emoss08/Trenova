/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { createGlobalStore } from "@/lib/useGlobalStore";
import { User } from "@/types/accounts";
import { create, SetState, StateCreator } from "zustand";
import { persist } from "zustand/middleware";

type AuthState = {
  isAuthenticated: boolean;
  setIsAuthenticated: (isAuthenticated: boolean) => void;
  loading: boolean;
  setLoading: (loading: boolean) => void;
  reset: () => void;
};

const createStore = (set: SetState<AuthState>) => ({
  isAuthenticated: false,
  setIsAuthenticated: (isAuthenticated: boolean) => set({ isAuthenticated }),
  loading: false,
  setLoading: (loading: boolean) => set({ loading }),
  reset: () => set({ isAuthenticated: false }),
});

export const useAuthStore = create<AuthState>(
  persist(createStore, {
    name: "Trenova-auth-storage",
  }) as StateCreator<AuthState>,
);

type UserStoreState = {
  user: User;
};

type CookieState = {
  isCookieConsentGiven: boolean;
  setIsCookieConsentGiven: (isCookieConsentGiven: boolean) => void;
  essentialCookies: boolean;
  setEssentialCookies: (essentialCookies: boolean) => void;
  functionalCookies: boolean;
  setFunctionalCookies: (functionalCookies: boolean) => void;
  performanceCookies: boolean;
  setPerformanceCookies: (performanceCookies: boolean) => void;
};

export const createCookieStore = (set: SetState<CookieState>) => ({
  isCookieConsentGiven: false,
  setIsCookieConsentGiven: (isCookieConsentGiven: boolean) =>
    set({ isCookieConsentGiven }),
  essentialCookies: false,
  setEssentialCookies: (essentialCookies: boolean) => set({ essentialCookies }),
  functionalCookies: false,
  setFunctionalCookies: (functionalCookies: boolean) =>
    set({ functionalCookies }),
  performanceCookies: false,
  setPerformanceCookies: (performanceCookies: boolean) =>
    set({ performanceCookies }),
});

export const useCookieStore = create<CookieState>(
  persist(createCookieStore, {
    name: "Trenova-cookie-storage",
  }) as StateCreator<CookieState>,
);

export const useUserStore = createGlobalStore<UserStoreState>({
  user: {
    id: "",
    username: "",
    organizationId: "",
    email: "",
    isAdmin: false,
    status: "I",
    timezone: "AmericaNewYork",
    name: "",
    profilePicUrl: "",
    isSuperAdmin: false,
    version: 0,
    createdAt: "",
    updatedAt: "",
    edges: {
      roles: [],
    },
  },
});

type TimezoneState = {
  timezone: string;
  setTimezone: (timezone: string) => void;
};

// Store timezone information in cookie
export const createTimezoneStore = (set: SetState<TimezoneState>) => ({
  timezone: "AmericaNewYork",
  setTimezone: (timezone: string) => set({ timezone }),
});

export const useTimezoneStore = create<TimezoneState>(
  persist(createTimezoneStore, {
    name: "Trenova-timezone-storage",
  }) as StateCreator<TimezoneState>,
);
