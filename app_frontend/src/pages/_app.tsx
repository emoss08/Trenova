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
// f1416c


import "nouislider/dist/nouislider.css";
import "../styles/assets/sass/style.scss";
import "../styles/assets/sass/plugins.scss";
import "../styles/assets/sass/style.react.scss";
import "../../public/splash-screen.css";
import "nprogress/nprogress.css";
import "react-toastify/dist/ReactToastify.min.css";

import type { AppProps } from "next/app";
import { LayoutProvider } from "@/utils/layout/LayoutProvider";
import { Poppins } from "next/font/google";
import axios from "axios";
import { setupAxios } from "@/utils/auth";
import { AuthInit, AuthGuard } from "@/utils/providers/AuthGuard";
import React, { Suspense, useEffect } from "react";
import { LayoutSplashScreen } from "@/components/elements/LayoutSplashScreen";
import { ToastContainer } from "react-toastify";
import { ThemeModeProvider, useThemeMode } from "@/utils/providers/ThemeProvider";
import { MasterInit } from "@/utils/MasterInit";
import KeepAliveConnection from "@/utils/components/KeepAliveConnection";


const poppins = Poppins({
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  subsets: ["latin"]
});

export default function App({ Component, pageProps }: AppProps) {
  setupAxios(axios);
  const [isMounted, setIsMounted] = React.useState(false);
  const { mode } = useThemeMode();
  const themeString = mode === "light" ? "light" : "dark";

  useEffect(() => {
    setIsMounted(true);
  }, []);

  if (!isMounted) {
    return <LayoutSplashScreen />;
  }

  return (
    <Suspense fallback={<LayoutSplashScreen />}>
      <AuthInit>
        <ThemeModeProvider>
          <LayoutProvider>
            <AuthGuard>
              <>
                <main className={poppins.className}>
                  <Component {...pageProps} />
                  <ToastContainer theme={themeString} />
                  <MasterInit />
                </main>
              </>
            </AuthGuard>
          </LayoutProvider>
        </ThemeModeProvider>
      </AuthInit>
    </Suspense>
  );
}
