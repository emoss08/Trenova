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
import type { Preview } from "@storybook/react";
import "../src/assets/App.css";
import { ThemeProvider, useTheme } from "../src/components/theme-provider";

const withTheme = (Story, context) => {
  const { setTheme } = useTheme();
  setTheme(context.globals.theme);
  return <Story />;
};

const preview: Preview = {
  parameters: {
    globalTypes: {
      locale: {
        description: 'Internationalization locale',
        defaultValue: 'en',
        toolbar: {
          icon: 'globe',
          items: [
            { value: 'en', right: '🇺🇸', title: 'English' },
            { value: 'fr', right: '🇫🇷', title: 'Français' },
            { value: 'es', right: '🇪🇸', title: 'Español' },
            { value: 'zh', right: '🇨🇳', title: '中文' },
            { value: 'kr', right: '🇰🇷', title: '한국어' },
          ],
        },
      },
      theme: {
        name: 'Theme',
        description: 'Global theme for components',
        defaultValue: 'light',
        toolbar: {
          icon: 'circlehollow',
          items: ['light', 'dark', 'system'],
          showName: true,
        },
      },
    },
    decorators: [
      (Story) => (
        <ThemeProvider defaultTheme="dark">
          <Story />
        </ThemeProvider>
      ),
      withTheme,
    ],
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/,
      },
    },
  },
};

export default preview;
