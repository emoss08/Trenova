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
import { ThemeModeProvider } from "@/utils/providers/ThemeProvider";
import { MasterInit } from "@/utils/MasterInit";

const poppins = Poppins({
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  subsets: ["latin"]
});

export default function App({ Component, pageProps }: AppProps) {
  setupAxios(axios);
  const [isMounted, setIsMounted] = React.useState(false);

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
              <style jsx global>{`
                html {
                  font-family: ${poppins.style.fontFamily};
                }
              `}</style>
              <>
                <Component {...pageProps} />
                <ToastContainer />
                <MasterInit />
              </>
            </AuthGuard>
          </LayoutProvider>
        </ThemeModeProvider>
      </AuthInit>
    </Suspense>
  );
}
