/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import "@/assets/App.css";
import "@/assets/Datepicker.css";
import LoadingSkeleton from "@/components/layout/loading-skeleton";
import { ThemeProvider } from "@/components/ui/theme-provider";
import { UserPermissionsProvider } from "@/context/user-permissions";
import { useVerifyToken } from "@/hooks/useVerifyToken";
import { THEME_KEY } from "@/lib/constants";
import { ProtectedRoutes } from "@/routing/ProtectedRoutes";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import "non.geist";
import { Suspense, memo } from "react";
import "react-datepicker/dist/react-datepicker.css";
import { BrowserRouter } from "react-router-dom";
import { Toaster } from "./components/ui/sonner";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      staleTime: Infinity,
      refetchOnWindowFocus: false,
    },
  },
});

export default function App() {
  const { isVerifying, isInitializationComplete } = useVerifyToken();
  const isLoading = isVerifying || !isInitializationComplete;

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  return (
    <ThemeProvider defaultTheme="dark" storageKey={THEME_KEY}>
      <UserPermissionsProvider>
        <AppImpl />
      </UserPermissionsProvider>
      <Toaster richColors closeButton />
    </ThemeProvider>
  );
}

const AppImpl = memo(() => (
  <QueryClientProvider client={queryClient}>
    <BrowserRouter>
      <Suspense fallback={<LoadingSkeleton />}>
        <ProtectedRoutes />
      </Suspense>
    </BrowserRouter>
    <ReactQueryDevtools buttonPosition="bottom-right" initialIsOpen={false} />
  </QueryClientProvider>
));
