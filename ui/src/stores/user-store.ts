import { type User } from "@/types/user";
import { create } from "zustand";
import { useShallow } from "zustand/react/shallow";

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

// Optimized selectors to prevent unnecessary re-renders
export const useUser = () => useAuthStore((state) => state.user);

export const useIsAuthenticated = () =>
  useAuthStore((state) => state.isAuthenticated);

export const useIsInitialized = () =>
  useAuthStore((state) => state.isInitialized);

// Use the useShallow hook to memoize the selector
export const useAuthActions = () =>
  useAuthStore(
    useShallow((state) => ({
      setUser: state.setUser,
      setInitialized: state.setInitialized,
      clearAuth: state.clearAuth,
    })),
  );
