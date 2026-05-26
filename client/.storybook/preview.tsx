import "@/styles/app.css";

import type { Decorator, Preview } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { useMemo } from "react";

import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/toaster";

const withAppProviders: Decorator = (Story, context) => {
  const theme = context.globals.theme === "dark" ? "dark" : "light";

  function StorybookProviders() {
    const queryClient = useMemo(
      () =>
        new QueryClient({
          defaultOptions: {
            queries: {
              retry: false,
              refetchOnWindowFocus: false,
              staleTime: 5 * 60 * 1000,
              gcTime: 10 * 60 * 1000,
            },
          },
        }),
      [],
    );

    return (
      <QueryClientProvider client={queryClient}>
        <NuqsTestingAdapter hasMemory>
          <ThemeProvider
            key={theme}
            defaultTheme={theme}
            storageKey={`trenova-storybook-theme-${theme}`}
          >
            <div className="min-h-screen bg-background p-6 text-foreground">
              <Story />
            </div>
            <Toaster position="top-center" />
          </ThemeProvider>
        </NuqsTestingAdapter>
      </QueryClientProvider>
    );
  }

  return <StorybookProviders />;
};

const preview: Preview = {
  decorators: [withAppProviders],
  globalTypes: {
    theme: {
      description: "Theme",
      toolbar: {
        icon: "circlehollow",
        items: [
          { value: "light", title: "Light" },
          { value: "dark", title: "Dark" },
        ],
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: {
    theme: "light",
  },
  parameters: {
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    layout: "fullscreen",
  },
  tags: ["autodocs", "test"],
};

export default preview;
