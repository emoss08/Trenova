import { AuthProps } from '../types/auth';
import { authenticate, logout } from '../contexts/MontaContext';
import createTyped from '../utils/zustandUtils';

export const initialState: AuthProps = {
  isAuthenticated: false,
  isInitialized: false,
  isLoading: false,
  user: null
};

export const useAuthStore = createTyped(
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
          console.log(authResult);
          if (authResult.isAuthenticated && authResult.user) {
            set((state: any) => ({
              ...state,
              authState: {
                isAuthenticated: authResult.isAuthenticated,
                isInitialized: authResult.isInitialized,
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
