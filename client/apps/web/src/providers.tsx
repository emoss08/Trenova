// import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { APP_CATALOGS } from "@/lib/app-catalogs";
import { edition } from "@/lib/edition";
import { installPlanLimitHandler } from "@/lib/plan-limit-handler";
import { installSessionContextHandler } from "@/lib/session-context-handler";
import { leaveExpiredSession } from "@/lib/session-exit";
import { queryClient } from "@/lib/query-client";
import { SessionQueryProvider } from "@/components/session-query-provider";
import { retainShellQueries, sessionPersistence, sessionScope } from "@/lib/session-cache";
import { RootErrorBoundary } from "@trenova/shared/components/error-boundary";
import { ThemeProvider } from "@trenova/shared/components/theme-provider";
import { Toaster } from "@trenova/shared/components/ui/toaster";
import { MotionPreferenceProvider } from "@/components/motion-preference-provider";
import { CodeLigaturesPreference } from "@/components/code-ligatures-preference";
import { I18nProvider } from "@trenova/shared/i18n/provider";
import { registerCatalogSource } from "@trenova/shared/i18n/runtime";
import { setPartialErrorReporter, setSessionExpiryHandler } from "@trenova/shared/lib/graphql";
import { currentProfile, useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ReactNode } from "react";
import { installFailureListeners } from "@trenova/shared/lib/failure-reporter";

// The transport cannot reach the router or the auth store, so the expiry path is wired here
// instead; the teardown is the one protectedLoader's 401 branch does.
setSessionExpiryHandler(leaveExpiredSession);

// The shell's queries live on in memory as long as their copy on disk is trusted.
retainShellQueries(queryClient);

// QUOTA_EXCEEDED and PLAN_RESTRICTED from any request open the plan-limit dialog; the
// transports report them here because they cannot reach the dialog themselves.
installPlanLimitHandler();

// A refusal of a tab another tab's sign-in, sign-out or organization switch left behind,
// and the other tabs' news of one, decide what this tab does next.
installSessionContextHandler();

// An edition's strings live in its own catalog; layering it over the app's before the
// first locale loads means its pages render translated like everything else.
void registerCatalogSource(edition.messages);

installFailureListeners();

setPartialErrorReporter((errors, { operationName }) => {
  for (const error of errors) {
    console.warn(
      `[graphql] ${operationName ?? "operation"} returned partial data: ${error.message}`,
      { code: error.code, path: error.path, traceId: error.traceId },
    );
  }
});

// The signed-in user's language is the one their emails and documents already use, so it
// outranks anything the browser reports. Reading it here rather than inside I18nProvider
// keeps the provider free of any dependency on how this app stores its session.
function useUserLocale(): string | null {
  return useAuthStore((state) => currentProfile(state)?.locale ?? null);
}

// The query cache, with the shell's queries kept on disk for the signed-in person in their
// organization (lib/session-cache).
function AppQueryProvider({ children }: { children: ReactNode }) {
  const scope = useAuthStore((state) => sessionScope(currentProfile(state)));
  return (
    <SessionQueryProvider client={queryClient} scope={scope} persistence={sessionPersistence}>
      {children}
    </SessionQueryProvider>
  );
}

function LocalizedApp({ children }: { children: ReactNode }) {
  return (
    <I18nProvider catalogs={APP_CATALOGS} userLocale={useUserLocale()}>
      {children}
    </I18nProvider>
  );
}

export function Providers({ children }: { children: ReactNode }) {
  return (
    <RootErrorBoundary>
      <AppQueryProvider>
        <ThemeProvider defaultTheme="system" storageKey="trenova-ui-theme">
          <MotionPreferenceProvider>
            <LocalizedApp>{children}</LocalizedApp>
            <Toaster position="top-right" />
            <CodeLigaturesPreference />
          </MotionPreferenceProvider>
          {/*<ReactQueryDevtools
              buttonPosition="bottom-left"
              initialIsOpen={false}
            />*/}
        </ThemeProvider>
      </AppQueryProvider>
    </RootErrorBoundary>
  );
}
