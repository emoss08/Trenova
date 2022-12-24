import React, { useEffect } from 'react';
import { createContext } from 'react';
import axios from 'axios';
import create from 'zustand';
import { AuthProps } from 'types/auth';

export type MontaUserProfile = {
  uid: string;
  title: string;
  firstName: string;
  lastName: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  zipCode: string;
  phone: string;
};

export type MontaUser = {
  uid: string;
  organization: string;
  department: string;
  email: string;
  username: string;
  profile: MontaUserProfile;
};

export type UserContextType = {
  uid?: string;
  isAuthenticated: boolean;
  isLoading: boolean;
  token?: string | null;
  user?: MontaUser | null | undefined;
  authenticate: (username: string, password: string) => Promise<({ isAuthenticated: true } & ProvisionResult) | { isAuthenticated: false }>;
  logout: () => void;
};

export type ProvisionResult = {
  token: string;
  user: {
    id: string;
    username: string;
    organization: string;
    department: string;
    email: string;
    profile: {
      id: string;
      first_name: string;
      last_name: string;
      title: string;
      address_line_1: string;
      address_line_2?: string;
      city: string;
      state: string;
      zip_code: string;
      phone: string;
    };
  };
};

const initialState: AuthProps = {
  isAuthenticated: false,
  isInitialized: false,
  isLoading: false,
  user: null
};
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
    localStorage.setItem('token', token);
    localStorage.setItem('m_user_info', JSON.stringify(user));
    return { isAuthenticated: true, token, user };
  } catch (error) {
    console.error(error);
    return { isAuthenticated: false };
  }
};

export const logout = () => {
  localStorage.removeItem('token');
  localStorage.removeItem('m_user_info');
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

/***
 * Thanks to @Acorn1010: https://twitch.tv/Acorn1010 for this code.
 * @param state - The state to be bound to the store.
 * @returns The bound state.
 */
// function createTyped<T>(state: (set: (state: T) => T, get: () => T) => T) {
//   return create(state as any) as UseBoundStore<StoreApi<T>>;
// }

const useAuthStore = create(
  (set) =>
    ({
      authState: initialState,
      authenticate: async (username: string, password: string) => {
        try {
          // Set isLoading to true before making the API request
          set((state: any) => ({
            ...state,
            authState: {
              ...state.authState,
              isLoading: true
            }
          }));
          const authResult = await authenticate(username, password);
          if (authResult.isAuthenticated && authResult.user) {
            set((state: any) => ({
              ...state,
              authState: {
                isAuthenticated: authResult.isAuthenticated,
                isLoading: false, // Set isLoading to false after the API request is complete
                token: authResult.token,
                user: authResult.user
              }
            }));
          }
        } catch (error) {
          console.error(error);
          // Set isLoading to false if the API request fails
          set((state: any) => ({
            ...state,
            authState: {
              ...state.authState,
              isLoading: false
            }
          }));
        }
      },
      setAuthState: (authState: AuthProps) => {
        set((state: any) => ({
          ...state,
          authState
        }));
      },
      logout: () => {
        set((state: AuthProps) => ({
          ...state,
          authState: logout()
        }));
      }
    } as const)
);

export const AuthProvider: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  const { authState, authenticate, logout, setAuthState } = useAuthStore<AuthState>((state: any) => state);
  const { isAuthenticated, user, isLoading } = authState;

  useEffect(() => {
    const token = localStorage.getItem('token');
    const user = localStorage.getItem('m_user_info');
    if (token && user) {
      const parsedUser = JSON.parse(user);
      const newAuthState = {
        isAuthenticated: true,
        isLoading: false,
        user: {
          uid: parsedUser.id,
          username: parsedUser.username,
          email: parsedUser.email,
          organization: parsedUser.organization,
          department: parsedUser.department,
          profile: {
            uid: parsedUser.id,
            title: parsedUser.profile.title,
            firstName: parsedUser.profile.first_name,
            lastName: parsedUser.profile.last_name,
            addressLine1: parsedUser.profile.address_line_1,
            addressLine2: parsedUser.profile.address_line_2,
            city: parsedUser.profile.city,
            state: parsedUser.profile.state,
            zipCode: parsedUser.profile.zip_code,
            phone: parsedUser.profile.phone
          }
        }
      };
      setAuthState(newAuthState);
    }
  }, [isLoading]);

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
