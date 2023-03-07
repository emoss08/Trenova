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
  Dispatch, FC, ReactNode,
  SetStateAction,
  useContext,
  useEffect,
  useState
} from "react";

import { WithChildren } from "@/utils/types";
import { AuthModel, JobTitleModel, UserAuthModel } from "@/models/user";
import * as authHelper from "@/utils/auth";
import { getJobTitle, getUserByToken } from "@/utils/_requests";
import { useRouter } from "next/router";
import { LayoutSplashScreen } from "@/components/elements/LayoutSplashScreen";
import { set } from "immutable";

export interface AuthContextType {
  isAuthenticated: boolean;
  user: UserAuthModel | undefined;
  loading: boolean;
  setLoading: Dispatch<SetStateAction<boolean>>;
  logout: () => void


}

const InitalAuthContext: AuthContextType = {
  setLoading: () => {},
  loading: false,
  isAuthenticated: false,
  user: undefined,
  logout: () => {},
}

const AuthContext = createContext<AuthContextType>(InitalAuthContext);

export const AuthProvider: React.FC<WithChildren> = ({ children }) => {
  const [user, setCurrentUser] = useState<UserAuthModel | undefined>();
  const [loading, setLoading] = useState<boolean>(true);
  const router = useRouter();

  useEffect(() => {
    async function loadUser() {
      const auth = authHelper.getAuth();
      if (auth) {
        console.log("Got Auth", auth);
        const { data: user } = await getUserByToken(auth.token);
        if (user) setCurrentUser(user);
      }

    }

    loadUser();
  }, []);

  const logout = () => {
    authHelper.clearAuth();
    setCurrentUser(undefined);
    router.push("/login");
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated: !user, user, logout, loading, setLoading }}>
      {children}
    </AuthContext.Provider>
  );
};
export const useAuth = () => useContext(AuthContext);

export const ProtectRoute: ({ children }: { children: any }) => (JSX.Element) = ({ children }) => {
  const router = useRouter();
  const { isAuthenticated, loading, setLoading } = useAuth();
  useEffect(() => {
    console.log("useEffect executed");
    console.log((router.pathname))
    console.log("Loading??", loading)
    console.log("Authenticated ",isAuthenticated)

    if (!loading && !isAuthenticated && router.pathname !== "/auth/login") {
      router.push("/auth/login");
      console.log((router.pathname))
      if (router.pathname === "/auth/login") {
        setLoading(false);
        console.log(loading);
      }
    }
  }, [loading, isAuthenticated, router, setLoading]);

  return loading ? <LayoutSplashScreen /> : children;
};
