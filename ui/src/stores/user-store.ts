import { type User } from "@/types/user";
import { create } from "zustand";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isInitialized: boolean;
  setUser: (user: User | null) => void;
  setInitialized: (initialized: boolean) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isInitialized: false,
  setUser: (user) =>
    set({
      user,
      isAuthenticated: !!user,
    }),
  setInitialized: (initialized) =>
    set({
      isInitialized: initialized,
    }),
  clearAuth: () =>
    set({
      user: null,
      isAuthenticated: false,
    }),
}));
