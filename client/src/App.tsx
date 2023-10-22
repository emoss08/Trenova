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

import "@/assets/App.css";
import "@/assets/Datepicker.css";
import LoadingSkeleton from "@/components/layout/loading-skeleton";
import { ThemeProvider } from "@/components/ui/theme-provider";
import { useVerifyToken } from "@/hooks/useVerifyToken";
import { ProtectedRoutes } from "@/routing/ProtectedRoutes";
import { useAuthStore } from "@/stores/AuthStore";
import "@fontsource-variable/inter";
import { memo, Suspense } from "react";
import "react-datepicker/dist/react-datepicker.css";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";
import { THEME_KEY } from "@/lib/constants";
import { UserPermissionsProvider } from "@/context/user-permissions";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";

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
    return <LoadingSkeleton />;
  }
  return (
    <ThemeProvider defaultTheme="dark" storageKey={THEME_KEY}>
      <UserPermissionsProvider>
        <AppImpl />
      </UserPermissionsProvider>
    </ThemeProvider>
  );
}

const AppImpl = memo(() => {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true }}>
        <Suspense fallback={<LoadingSkeleton />}>
          <ProtectedRoutes />
        </Suspense>
      </BrowserRouter>
      <ReactQueryDevtools buttonPosition="bottom-left" initialIsOpen={false} />
    </QueryClientProvider>
  );
});
