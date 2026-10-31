import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { onSignedInElsewhere } from "@trenova/shared/lib/session-context";
import { useEffect } from "react";

/**
 * While a sign-in page is mounted, calls `follow` when the person finishes signing in in
 * another tab, so this one goes into the app with them instead of asking them to sign in a
 * second time. A tab in the middle of its own sign-in already holds a session, and is never
 * called.
 */
export function useFollowSignInElsewhere(follow: () => void): void {
  const latestFollow = useLatestCallback(follow);

  useEffect(() => onSignedInElsewhere(latestFollow), [latestFollow]);
}
