import React, { useEffect } from 'react';
import { createContext } from 'react';
import axios from 'axios';
import { AuthProps } from 'types/auth';
import { ProvisionResult, UserContextType } from 'types/user';
import { useAuthStore } from 'store/auth';
import LocalStorageService from '../services/LocalStorageService';

export const MontaAuthContext = createContext({} as UserContextType);

export const authenticate = async (
  username: string,
  password: string
): Promise<({ isAuthenticated: true; isInitialized: true } & ProvisionResult) | { isAuthenticated: false; isInitialized: false }> => {
  try {
    const response = await axios.post('http://localhost:8000/api/token/provision/', {
      username,
      password
    });
    const { token, user } = response.data as ProvisionResult;
    LocalStorageService.setToken(token);
    LocalStorageService.setUser(user);
    return { isAuthenticated: true, isInitialized: true, token, user };
  } catch (error) {
    console.error(error);
    return { isAuthenticated: false, isInitialized: false };
  }
};

export const logout = () => {
  LocalStorageService.clearRelatedUser();
  return { isAuthenticated: false, isInitialized: false, user: null };
};

interface AuthState {
  authState: AuthProps;
  authenticate: (username: string, password: string) => Promise<({ isAuthenticated: true } & ProvisionResult) | { isAuthenticated: false }>;

  logout: () => void;
  setAuthState: (authState: {
    isAuthenticated: boolean;
    isInitialized: boolean;
    user: {
      uid: any;
      organization: any;
      profile: {
        uid: any;
        firstName: string;
        lastName: string;
        zipCode: string;
        city: any;
        phone: any;
        addressLine1: string;
        addressLine2: string | undefined;
        state: any;
        title: any;
      };
      department: any;
      email: any;
      username: any;
    };
  }) => void;
}

export const AuthProvider: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const { authState, authenticate, logout, setAuthState } = useAuthStore<AuthState>((state: any) => state);
  const { isAuthenticated, user, isLoading } = authState;

  useEffect(() => {
    // Remove the authentication logic from the useEffect hook
  }, [isLoading, setAuthState, isAuthenticated]);

  // Create a new function to handle authentication

  return (
    <MontaAuthContext.Provider
      value={{
        uid: user?.uid,
        isAuthenticated,
        token: authState.token,
        user,
        authenticate,
        logout,
        isLoading
      }}
    >
      {children}
    </MontaAuthContext.Provider>
  );
};
