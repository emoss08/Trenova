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

import React, {
  memo,
  PropsWithChildren,
  Suspense,
  useEffect,
  useMemo,
} from "react";
import { BrowserRouter } from "react-router-dom";
import "./assets/styles/App.css";
import {
  ColorScheme,
  ColorSchemeProvider,
  MantineProvider,
} from "@mantine/core";
import { QueryClient, QueryClientProvider } from "react-query";
import { Notifications } from "@mantine/notifications";
import { ReactQueryDevtools } from "react-query/devtools";
import { ContextMenuProvider } from "mantine-contextmenu";
import { ModalsProvider } from "@mantine/modals";
import { useHotkeys, useLocalStorage } from "@mantine/hooks";
import { useAuthStore } from "@/stores/AuthStore";
import { LoadingScreen } from "@/components/common/LoadingScreen";
import { ProtectedRoutes } from "@/routing/ProtectedRoutes";
import { useVerifyToken } from "@/hooks/useVerifyToken";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});
export default function App() {
  const { isVerifying, isInitializationComplete } = useVerifyToken();

  const initialLoading = useAuthStore(
    (state: { initialLoading: boolean }) => state.initialLoading,
  );

  const isLoading = isVerifying || initialLoading || !isInitializationComplete;

  if (isLoading) {
    return <LoadingScreen />;
  }

  return (
    <MantineColorProvider>
      <AppImpl />
    </MantineColorProvider>
  );
}

function MantineColorProvider({
  children,
}: PropsWithChildren<NonNullable<unknown>>) {
  const [colorScheme, setColorScheme] = useLocalStorage<ColorScheme>({
    key: "mt-color-scheme",
    defaultValue: "light",
    getInitialValueInEffect: true,
  });
  useHotkeys([["mod+J", () => toggleColorScheme()]]);

  useEffect(() => {
    document.body.className =
      colorScheme === "dark" ? "dark-theme" : "light-theme";
  }, [colorScheme]);

  const toggleColorScheme = useMemo(
    () => (value?: ColorScheme) => {
      setColorScheme(value || (colorScheme === "dark" ? "light" : "dark"));
    },
    [colorScheme, setColorScheme],
  );

  return (
    <ColorSchemeProvider
      colorScheme={colorScheme}
      toggleColorScheme={toggleColorScheme}
    >
      <MantineProvider
        theme={{
          colorScheme,
          fontFamily: "Inter, sans-serif",
        }}
        withGlobalStyles
        withNormalizeCSS
        withCSSVariables
      >
        {children}
      </MantineProvider>
    </ColorSchemeProvider>
  );
}

const AppImpl = memo(() => {
  return (
    <ModalsProvider>
      <ContextMenuProvider zIndex={1000} shadow="md" borderRadius="md">
        <Notifications limit={3} position="top-right" zIndex={2077} />
        <QueryClientProvider client={queryClient}>
          <BrowserRouter>
            <Suspense fallback={<LoadingScreen />}>
              <ProtectedRoutes />
            </Suspense>
          </BrowserRouter>
          <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
      </ContextMenuProvider>
    </ModalsProvider>
  );
});
