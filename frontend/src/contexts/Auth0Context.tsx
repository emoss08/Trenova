import React, { createContext, useEffect, useReducer } from 'react';

// third-party
import { Auth0Client } from '@auth0/auth0-spa-js';

// reducer - state management
import { LOGIN, LOGOUT } from 'store/reducers/actions';
import authReducer from 'store/reducers/auth';

// project import
import Loader from 'components/Loader';
import { AUTH0_API } from 'config';
import { KeyedObject } from 'types/root';
import { Auth0ContextType, AuthProps } from 'types/auth';

// constant
let auth0Client: Auth0Client;

const initialState: AuthProps = {
  isLoggedIn: false,
  isInitialized: false,
  user: null
};

// ==============================|| AUTH0 CONTEXT & PROVIDER ||============================== //

const Auth0Context = createContext<Auth0ContextType | null>(null);

export const Auth0Provider = ({ children }: { children: React.ReactElement }) => {
  const [state, dispatch] = useReducer(authReducer, initialState);

  useEffect(() => {
    const init = async () => {
      try {
        auth0Client = new Auth0Client({
          redirect_uri: window.location.origin,
          ...AUTH0_API
        });

        await auth0Client.checkSession();
        const isLoggedIn = await auth0Client.isAuthenticated();

        if (isLoggedIn) {
          const user = await auth0Client.getUser();

          dispatch({
            type: LOGIN,
            payload: {
              isLoggedIn: true,
              user: {
                id: user?.sub,
                email: user?.email
              }
            }
          });
        } else {
          dispatch({
            type: LOGOUT
          });
        }
      } catch (err) {
        dispatch({
          type: LOGOUT
        });
      }
    };

    init();
  }, []);

  const login = async (options?: KeyedObject) => {
    await auth0Client.loginWithPopup(options);
    const isLoggedIn = await auth0Client.isAuthenticated();

    if (isLoggedIn) {
      const user = await auth0Client.getUser();
      dispatch({
        type: LOGIN,
        payload: {
          isLoggedIn: true,
          user: {
            id: user?.sub,
            avatar: user?.picture,
            email: user?.email,
            name: user?.name,
            tier: 'Premium'
          }
        }
      });
    }
  };

  const logout = () => {
    auth0Client.logout();

    dispatch({
      type: LOGOUT
    });
  };

  const resetPassword = async (email: string) => {};

  const updateProfile = () => {};

  if (state.isInitialized !== undefined && !state.isInitialized) {
    return <Loader />;
  }

  return <Auth0Context.Provider value={{ ...state, login, logout, resetPassword, updateProfile }}>{children}</Auth0Context.Provider>;
};

export default Auth0Context;
