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
): Promise<({ isAuthenticated: true } & ProvisionResult) | { isAuthenticated: false }> => {
  try {
    const response = await axios.post('http://localhost:8000/api/token/provision/', {
      username,
      password
    });
    const { token, user } = response.data as ProvisionResult;
    LocalStorageService.setToken(token);
    LocalStorageService.setUser(user);
    return { isAuthenticated: true, token, user };
  } catch (error) {
    console.error(error);
    return { isAuthenticated: false };
  }
};

export const logout = () => {
  LocalStorageService.clearRelatedUser();
  return { isAuthenticated: false, user: null };
};

interface AuthState {
  authState: AuthProps;
  authenticate: (username: string, password: string) => Promise<({ isAuthenticated: true } & ProvisionResult) | { isAuthenticated: false }>;

  logout: () => void;
  setAuthState: (authState: {
    isAuthenticated: boolean;
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
    const token = LocalStorageService.getToken();
    const user = LocalStorageService.getUser();
    if (token && user) {
      const newAuthState = {
        isAuthenticated: true,
        isLoading: false,
        user: {
          uid: user.id,
          username: user.username,
          email: user.email,
          organization: user.organization,
          department: user.department,
          profile: {
            uid: user.id,
            title: user.profile.title,
            firstName: user.profile.first_name,
            lastName: user.profile.last_name,
            addressLine1: user.profile.address_line_1,
            addressLine2: user.profile.address_line_2,
            city: user.profile.city,
            state: user.profile.state,
            zipCode: user.profile.zip_code,
            phone: user.profile.phone
          }
        }
      };
      setAuthState(newAuthState);
    }
  }, [isLoading, setAuthState]);

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
