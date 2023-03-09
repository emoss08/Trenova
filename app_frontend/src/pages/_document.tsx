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

import { Html, Head, Main, NextScript } from "next/document";
import Script from "next/script";

const appScript = `
  document.addEventListener("DOMContentLoaded", function() {
    if (document.documentElement) {
      var defaultThemeMode = 'system'

      var hasKTName = document.body.hasAttribute('data-mt-name')
      var lsKey = 'mt_' + (hasKTName ? name + '_' : '') + 'theme_mode_value'
      var themeMode = localStorage.getItem(lsKey)
      if (!themeMode) {
        if (defaultThemeMode === 'system') {
          themeMode = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
        } else {
          themeMode = defaultThemeMode
        }
      }

      document.documentElement.setAttribute('data-bs-theme', themeMode)
    }
  });
`;

export default function Document() {
  return (
    <Html lang="en">
      <Head />
      <body>
      <Script
        id={"theme-mode-script"}
        strategy="beforeInteractive"
        dangerouslySetInnerHTML={{ __html: appScript }}
      />
      <Main />
      <NextScript />

      </body>
    </Html>
  );
}
