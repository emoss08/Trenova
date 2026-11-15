import { useRouter, useRouterState } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";

/**
 * Back and forward through the app's own history. The router numbers every entry it writes,
 * so there is somewhere to go back to while the entry is not the first, and somewhere to go
 * forward to while it sits below the furthest entry pushed since. A push drops the entries
 * after it, as the browser does.
 */
export function useHistoryNavigation() {
  const router = useRouter();
  const index = useRouterState({ select: (state) => state.location.state.__TSR_index ?? 0 });
  const [pushedTo, setPushedTo] = useState(index);

  useEffect(
    () =>
      router.history.subscribe(({ location, action }) => {
        if (action.type === "PUSH") {
          setPushedTo(location.state.__TSR_index ?? 0);
        }
      }),
    [router],
  );

  const canGoBack = index > 0;
  const canGoForward = index < Math.max(pushedTo, index);

  const goBack = useCallback(() => {
    if (canGoBack) router.history.back();
  }, [canGoBack, router]);

  const goForward = useCallback(() => {
    if (canGoForward) router.history.forward();
  }, [canGoForward, router]);

  return {
    canGoBack,
    canGoForward,
    goBack,
    goForward,
  };
}
