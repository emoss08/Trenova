/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import React, { FC, PropsWithChildren, useEffect, useRef } from "react";
import { useRouter } from "next/router";
import { AuthModel, UserAuthModel } from "@/models/user";
import * as authHelper from "@/utils/auth";
import { getUserByToken } from "../_requests";
import { createGlobalStore } from "@/utils/zustand";
import { jobStore } from "@/utils/stores";

type AuthType = { auth?: AuthModel; user?: UserAuthModel };
const store = createGlobalStore<AuthType>({});
export const authStore = store;

export function logout() {
  store.update({ auth: undefined, user: undefined });
  jobStore.update({ job: undefined });
  authHelper.clearAuth();
}

export function saveAuth(auth: AuthModel | undefined) {
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

const PUBLIC_PATHS = ["/auth/login"];

const AuthGuard: FC<PropsWithChildren> = ({ children }) => {
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
  }, [router, isAuthenticated, setAuth, auth]);

  if (!isAuthenticated && !PUBLIC_PATHS.includes(router.pathname)) {
    return null;
  }

  return <>{children}</>;
};

const AuthInit: FC<PropsWithChildren> = ({ children }) => {
  const [auth] = store.use("auth", authHelper.getAuth());
  const didRequest = useRef(false);
  const router = useRouter();

  useEffect(() => {
    const requestUser = async (apiToken: string) => {
      try {
        if (!didRequest.current) {
          const { data } = await getUserByToken(apiToken);
          if (data) {
            store.set("user", data);
          }
        }
      } catch (error) {
        console.error(error);
        if (!didRequest.current) {
          logout();
        }
      }
      return () => (didRequest.current = true);
    };

    if (auth && auth.token) {
      requestUser(auth.token).then(() => {
      });
      return;
    }

  }, [auth, router]);

  return <>{children}</>;
};

export { AuthInit, AuthGuard };
