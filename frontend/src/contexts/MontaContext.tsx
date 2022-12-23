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
  profilePicture: string;
  bio?: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  zipCode: string;
  phone: string;
};

export type MontaUser = {
  uid?: string;
  email?: string;
  username?: string;
  emailVerified?: boolean;
  profile?: MontaUserProfile;
};

export type UserContextType = {
  uid?: string;
  isAuthenticated?: boolean;

  token?: string | null;
  user?: MontaUser | null | undefined;
  authenticate: (username: string, password: string) => Promise<{ isAuthenticated: boolean; user: UserContextType }>;
  logout: () => void;
};

const initialState: AuthProps = {
  isAuthenticated: false,
  user: null
};

export const MontaAuthContext = createContext({} as UserContextType);

export const authenticate = async (username: string, password: string): Promise<{ isAuthenticated: boolean; user: UserContextType }> => {
  try {
    const response = await axios.post('http://localhost:8000/api/token/provision/', {
      username,
      password
    });
    const { token, user: userData } = response.data;
    console.log(response.data);
    localStorage.setItem('token', token);
    localStorage.setItem('m_user_info', JSON.stringify(userData));
    return { isAuthenticated: true, user: { ...userData, token } };
  } catch (error) {
    console.error(error);
    return { isAuthenticated: false, user: {} as UserContextType };
  }
};

export const logout = () => {
  localStorage.removeItem('token');
  console.log({ authenticate });
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
            username: JSON.parse(user).username
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
        authenticate,
        logout
      }}
    >
      {children}
    </MontaAuthContext.Provider>
  );
};
