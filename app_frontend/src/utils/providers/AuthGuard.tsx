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

import React, { FC, useEffect, useRef, useState } from "react";
import { useRouter } from "next/router";
import { WithChildren } from "@/utils/types";
import { AuthModel, UserAuthModel } from "@/models/user";
import * as authHelper from "@/utils/auth";
import { getUserByToken } from "../_requests";
import { LayoutSplashScreen } from "@/components/elements/LayoutSplashScreen";
import { createGlobalStore } from "@/utils/zustand";
import { jobStore } from "@/utils/stores";

type AuthType = { auth?: AuthModel, user?: UserAuthModel };
const store = createGlobalStore<AuthType>({});
export const authStore = store;

export function logout() {
  store.update({ auth: undefined, user: undefined });
  jobStore.update({ job: undefined });
  authHelper.clearAuth();
}

export function saveAuth(auth: AuthModel | undefined) {
  // If the auth token changed, then we'll need to refetch the user
  if (auth?.token !== store.get("auth")?.token) {
    store.set("user", undefined);
  }
  store.set("auth", auth);
  if (auth) {
    authHelper.setAuth(auth);
  } else {
    authHelper.clearAuth();
  }
}

/** List of routes that should be public and accessible without authentication. */
const PUBLIC_PATHS = ["/auth/login"];

const AuthGuard: React.FC<WithChildren> = ({ children }) => {
  const [auth, setAuth] = store.use("auth");
  const router = useRouter();

  const isAuthenticated = !!auth?.token;

  useEffect(() => {
    let isMounted = true;

    if (!isAuthenticated && router.pathname !== "/auth/login") {
      router.push({
          pathname: "/auth/login",
          query: { redirect: router.pathname }
        },
        undefined,
        { shallow: true })
        .catch((e) => {
          if (e.cancelled) {
            throw e;
          }
        });
    } else if (isAuthenticated && router.pathname === "/auth/login") {
      router.replace("/").then(() => {
        if (isMounted) {
          setAuth(auth);
          authHelper.setAuth(auth);
        }
      });
    }

    return () => {
      isMounted = false;
    };
  }, [router, isAuthenticated, setAuth]);

  // Auth check. If trying to navigate to a non-public path, reroute to the login page.
  if (!isAuthenticated && !PUBLIC_PATHS.includes(router.pathname)) {
    return null;
  }

  return <>{children}</>;
};


const AuthInit: FC<WithChildren> = ({ children }) => {
  const [auth] = store.use("auth", authHelper.getAuth());
  const didRequest = useRef(false);
  const [showSplashScreen, setShowSplashScreen] = useState(true);
  const router = useRouter();

  useEffect(() => {
    const requestUser = async (apiToken: string) => {
      try {
        setShowSplashScreen(true);
        if (!didRequest.current) {
          const { data } = await getUserByToken(apiToken);
          console.log("I'm being called", data);
          if (data) {
            store.set("user", data);
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
      return;
    }

    setShowSplashScreen(false);
  }, [auth, router]);

  return showSplashScreen ? <LayoutSplashScreen /> : <>{children}</>;
};

export { AuthInit, AuthGuard };
