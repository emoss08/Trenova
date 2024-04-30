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
