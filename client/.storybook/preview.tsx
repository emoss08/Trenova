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
import React from "react";
import "../src/assets/App.css";
import { ThemeProvider, useTheme } from "../src/components/theme-provider";

const withTheme = (Story, context) => {
  const { setTheme } = useTheme();
  setTheme(context.globals.theme);
  return <Story />;
};


const getTheme = (theme) => {
  switch (theme) {
    case "light":
      return "light";
    case "dark":
      return "dark";
    default:
      return "light";
  }
}

const withThemeProvider = (Story, context) => {
  const theme = getTheme(context.globals.theme);
  return (
    <ThemeProvider defaultTheme="dark">
      <Story />
    </ThemeProvider>
  );
};


const preview: Preview = {
  decorators: [withThemeProvider, withTheme
  ],
  parameters: {
    globalTypes: {
      theme: {
        description: 'Global theme for components',
        defaultValue: 'light',
        toolbar: {
          // The label to show for this toolbar item
          title: 'Theme',
          icon: 'circlehollow',
          // Array of plain string values or MenuItem shape (see below)
          items: ['light', 'dark'],
          // Change title based on selected value
          dynamicTitle: true,
        },
      },
    },

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
