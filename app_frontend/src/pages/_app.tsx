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

import "@/styles/SplashScreen.module.css";
import "@/styles/sass/style.scss";
import "@/styles/sass/plugins.scss";
import "@/styles/sass/style.react.scss";
import "nprogress/nprogress.css";
import "react-toastify/dist/ReactToastify.min.css";

import type { AppProps } from "next/app";
import { LayoutProvider } from "@/utils/providers/layout-provider";
import { Poppins } from "next/font/google";
import axios from "axios";
import { setupAxios } from "@/utils/auth";
import { AuthInit, AuthProvider } from "@/utils/providers/AuthProvider";
import { Suspense, useEffect } from "react";
import { LayoutSplashScreen } from "@/components/elements/LayoutSplashScreen";
import NProgress from "nprogress";
import { ToastContainer } from "react-toastify";


const poppins = Poppins({
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  subsets: ["latin"]
});

export default function App({ Component, pageProps, router }: AppProps) {
  setupAxios(axios);

  useEffect(() => {
    const handleRouteStart = () => NProgress.start();
    const handleRouteDone = () => NProgress.done();

    router.events.on("routeChangeStart", handleRouteStart);
    router.events.on("routeChangeComplete", handleRouteDone);
    router.events.on("routeChangeError", handleRouteDone);

    return () => {
      // Make sure to remove the event handler on unmount!
      router.events.off("routeChangeStart", handleRouteStart);
      router.events.off("routeChangeComplete", handleRouteDone);
      router.events.off("routeChangeError", handleRouteDone);
    };
  }, [router.events]);

  return (
    <Suspense fallback={<LayoutSplashScreen />}>
      <AuthInit>
        <LayoutProvider>
          {/*<ThemeProvider>*/}
          <AuthProvider>
            <style jsx global>{`
              html {
                font-family: ${poppins.style.fontFamily};
              }
            `}</style>
            <>
              <Component {...pageProps} />
              <ToastContainer />
            </>
          </AuthProvider>
          {/*</ThemeProvider>*/}
        </LayoutProvider>
      </AuthInit>
    </Suspense>
  );
}
