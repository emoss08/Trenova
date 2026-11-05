import { disablePush, getPushSubscription } from "@trenova/shared/lib/push";

const PUSH_OWNER_KEY = "trenova-dash-push-owner";

/** Records whose alerts this browser's push subscription carries. */
export function rememberPushOwner(userId: string): void {
  window.localStorage.setItem(PUSH_OWNER_KEY, userId);
}

/** Stops this browser's alerts for the driver leaving it. */
export async function releasePush(): Promise<void> {
  window.localStorage.removeItem(PUSH_OWNER_KEY);
  await disablePush();
}

/**
 * An idle timeout keeps a driver's alerts, so a phone left alone still hears about a new load.
 * When someone else signs in on that browser, the alerts stop rather than reach them.
 */
export async function dropPushOfAnotherDriver(userId: string): Promise<void> {
  if (window.localStorage.getItem(PUSH_OWNER_KEY) === userId) {
    return;
  }
  if (await getPushSubscription()) {
    await releasePush();
  }
}
