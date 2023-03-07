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

import { Poppins } from "next/font/google";
import "@/styles/SplashScreen.module.css";
import "@/styles/sass/style.scss";
import "@/styles/sass/plugins.scss";
import "@/styles/sass/style.react.scss";
import type { AppProps } from "next/app";
import { LayoutProvider } from "@/utils/providers/layout-provider";
import { AuthProvider, ProtectRoute } from "@/utils/providers/AuthProvider";
import { ThemeProvider } from "next-themes";
import NextNProgress from "nextjs-progressbar";
import axios from "axios";
import { setupAxios } from "@/utils/auth";

const poppins = Poppins({
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  subsets: ["latin"]
});

export default function App({ Component, pageProps }: AppProps) {
  setupAxios(axios);

  return (
    <LayoutProvider>
      <ThemeProvider>
        <AuthProvider>
          <ProtectRoute>
            <style jsx global>{`
              html {
                font-family: ${poppins.style.fontFamily};
              }
            `}</style>
            <>
              <NextNProgress nonce="my-nonce" />
              <Component {...pageProps} />
            </>
          </ProtectRoute>

        </AuthProvider>
      </ThemeProvider>
    </LayoutProvider>

  );
}
