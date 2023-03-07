/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import React, {
  createContext,
  Dispatch, FC,
  SetStateAction,
  useContext, useEffect, useRef,
  useState
} from "react";
import { useRouter } from "next/router";
import { WithChildren } from "@/utils/types";
import { AuthModel, UserAuthModel } from "@/models/user";
import * as authHelper from "@/utils/auth";
import { getUserByToken } from "../_requests";
import { LayoutSplashScreen } from "@/components/elements/LayoutSplashScreen";

export interface AuthContextProps {
  auth: AuthModel | undefined;
  saveAuth: (auth: AuthModel | undefined) => void;
  user: UserAuthModel | undefined;
  setUser: Dispatch<SetStateAction<UserAuthModel | undefined>>;
  logout: () => void;
}

const initAuthContextPropsState: AuthContextProps = {
  auth: authHelper.getAuth(),
  saveAuth: () => {
  },
  user: undefined,
  setUser: () => {
  },
  logout: () => {
  }
};

const AuthContext = createContext<AuthContextProps>(initAuthContextPropsState);

const useAuth = () => {
  return useContext(AuthContext);
};

const AuthProvider: React.FC<WithChildren> = ({ children }) => {
  const [auth, setAuth] = useState<AuthModel | undefined>(authHelper.getAuth());
  const [user, setUser] = useState<UserAuthModel | undefined>(undefined);

  const saveAuth = (auth: AuthModel | undefined) => {
    setAuth(auth);
    if (auth) {
      authHelper.setAuth(auth);
    } else {
      authHelper.clearAuth();
    }
  };

  const logout = () => {
    saveAuth(undefined);
    setUser(undefined);
  };

  return (
    <AuthContext.Provider value={{ auth, saveAuth, user, setUser, logout }}>
      {children}
    </AuthContext.Provider>
  );
};


const AuthInit: FC<WithChildren> = ({ children }) => {
  const { auth, logout, setUser } = useAuth();
  const didRequest = useRef(false);
  const [showSplashScreen, setShowSplashScreen] = useState(true);
  const router = useRouter();
  // We should request user by authToken (IN OUR EXAMPLE IT'S API_TOKEN) before rendering the application
  useEffect(() => {


    const requestUser = async (apiToken: string) => {
      try {
        if (!didRequest.current) {
          const { data } = await getUserByToken(apiToken);
          if (data) {
            setUser(data);
          }
        }
      } catch (error) {
        console.error(error);
        if (!didRequest.current) {
          logout();
        }
      } finally {
        setShowSplashScreen(false);
      }
      return () => (didRequest.current = true);
    };

    if (auth && auth.token) {
      requestUser(auth.token);
    } else {
      router.push("/auth/login").then(
        () => {
          logout();
          setShowSplashScreen(false);
        });
    }
    // eslint-disable-next-line
  }, []);

  return showSplashScreen ? <LayoutSplashScreen /> : <>{children}</>;
};


export { AuthInit, AuthProvider, useAuth };