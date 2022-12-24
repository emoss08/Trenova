import React, { useEffect, useReducer } from 'react';
import { createContext } from 'react';
import axios from 'axios';
import authReducer from '../store/reducers/auth';

import { LOGIN, LOGOUT } from 'store/reducers/actions';
import { AuthProps } from '../types/auth';

export type MontaUserProfile = {
  uid: string;
  title: string;
  firstName: string;
  lastName: string;
  profilePicture?: string;
  bio?: string;
  addressLine1: string;
  addressLine2?: string;
  city?: string;
  state?: string;
  zipCode?: string;
  phone?: string;
};

export type MontaUser = {
  uid: string;
  email?: string;
  username?: string;
  emailVerified?: boolean;
  profile: MontaUserProfile;
};

export type UserContextType = {
  uid?: string;
  isAuthenticated?: boolean;
  token?: string | null;
  user?: MontaUser | null | undefined;
  authenticate: (username: string, password: string) => Promise<({ isAuthenticated: true } & ProvisionResult) | { isAuthenticated: false }>;
  logout: () => void;
};

export type ProvisionResult = {
  token: string;
  user: {
    pk: string;
    username: string;
    profile: {
      pk: string;
      first_name: string;
      last_name: string;
      title: string;
      address_line_1: string;
    };
  };
};

const initialState: AuthProps = {
  isAuthenticated: false,
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
  return { isAuthenticated: false, user: null };
};

export const AuthProvider: React.FC<{ children: React.ReactElement }> = ({ children }) => {
  // const [authState, setAuthState] = useState({
  //   isAuthenticated: false,
  //   user: null
  // });
  const [state, dispatch] = useReducer(authReducer, initialState);

  useEffect(() => {
    const token = localStorage.getItem('token');
    const user = localStorage.getItem('m_user_info');
    if (token && user) {
      dispatch({
        type: LOGIN,
        payload: {
          isAuthenticated: true,
          user: {
            uid: JSON.parse(user).id,
            username: JSON.parse(user).username,
            profile: {
              uid: JSON.parse(user).id,
              firstName: JSON.parse(user).first_name,
              lastName: JSON.parse(user).last_name,
              title: JSON.parse(user).title,
              addressLine1: JSON.parse(user).address_line_1
            }
          }
        }
      });
    } else {
      dispatch({
        type: LOGOUT
      });
    }
  }, [dispatch]);

  return (
    <MontaAuthContext.Provider
      value={{
        ...state,
        authenticate: async (username: string, password: string) => {
          const context = await authenticate(username, password);
          if (!context.isAuthenticated) {
            return context;
          }
          const user = context.user;
          if (!user) {
            return context;
          }
          dispatch({
            type: LOGIN,
            payload: {
              isAuthenticated: true,
              user: {
                uid: user.pk,
                username: user.username,
                profile: {
                  uid: user.profile.pk,
                  firstName: user.profile.first_name,
                  lastName: user.profile.last_name,
                  title: user.profile.title,
                  addressLine1: user.profile.address_line_1
                }
              }
            }
          });
          return context;
        },
        logout
      }}
    >
      {children}
    </MontaAuthContext.Provider>
  );
};
