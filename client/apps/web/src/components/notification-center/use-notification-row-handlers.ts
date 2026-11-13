import { handleMutationError, useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { user as userQueries } from "@/lib/queries/user";
import { toast } from "@trenova/shared/components/ui/toast";
import {
  useNotificationAction,
  useSnoozeNotifications,
} from "@trenova/shared/hooks/use-notifications";
import { useReducedMotion } from "@trenova/shared/hooks/use-reduced-motion";
import { useT } from "@trenova/shared/i18n/use-t";
import { notificationPreferencesService } from "@trenova/shared/services/notification-preferences";
import type { Notification } from "@trenova/shared/types/notification";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { runNotificationAction, type NotificationWriteAction } from "./notification-actions";
import { notificationPreferenceKey } from "./notification-registry";

/** How long a row takes to fold away before the list closes over it. */
export const ROW_LEAVE_MS = 180;

const NO_ROWS: ReadonlySet<string> = new Set();

/**
 * Rows on their way out of the list. A row folds shut first and is taken out of the list
 * after, so the rows under it slide up instead of jumping. `commit` is the write that
 * removes it; the row stays folded until that settles, and opens again if it failed and
 * the list still holds it. A row already leaving refuses a second commit.
 */
function useLeavingRows() {
  const reducedMotion = useReducedMotion();
  const [leaving, setLeaving] = useState(NO_ROWS);
  const pending = useRef(new Map<string, { timer: number; commit: () => Promise<unknown> }>());

  // Closing the panel mid-fold must not lose the write the person asked for.
  useEffect(() => {
    const waiting = pending.current;

    return () => {
      for (const { timer, commit } of waiting.values()) {
        window.clearTimeout(timer);
        void commit().catch(() => undefined);
      }
      waiting.clear();
    };
  }, []);

  const settle = (id: string) =>
    setLeaving((current) => {
      const next = new Set(current);
      next.delete(id);
      return next;
    });

  const leave = (id: string, commit: () => Promise<unknown>) => {
    if (pending.current.has(id) || leaving.has(id)) return;
    if (reducedMotion) {
      void commit().catch(() => undefined);
      return;
    }

    setLeaving((current) => new Set(current).add(id));
    const timer = window.setTimeout(() => {
      pending.current.delete(id);
      void commit()
        .catch(() => undefined)
        .finally(() => settle(id));
    }, ROW_LEAVE_MS);
    pending.current.set(id, { timer, commit });
  };

  return { leaving, leave };
}

export interface NotificationRowHandlers {
  /** Goes to `href` and closes the panel; the row is read, or done when opening settles it. */
  open: (notification: Notification, href: string, settles?: boolean) => void;
  read: (id: string) => void;
  unread: (id: string) => void;
  done: (id: string) => void;
  /** Sets the row aside until `until` (Unix seconds). */
  later: (id: string, until: number) => void;
  /** Brings a row set aside back now. */
  wake: (id: string) => void;
  /** Moves a finished row back to where it was. */
  restore: (id: string) => void;
  /**
   * Carries out a row's own action; the row is done once it succeeds. A refusal is reported
   * here, so the promise only tells the button when to let go.
   */
  run: (notification: Notification, action: NotificationWriteAction) => Promise<void>;
  /** Stops this kind of notification reaching the person, with a way back. */
  turnOff: (notification: Notification) => void;
  copyLink: (href: string) => void;
}

/**
 * Everything a row can do, wired once for the whole list: the writes, the navigation and
 * the fold a row makes on its way out.
 */
export function useNotificationRowHandlers(onClose: () => void): {
  handlers: NotificationRowHandlers;
  leaving: ReadonlySet<string>;
} {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { copy } = useCopyToClipboard();
  const { leaving, leave } = useLeavingRows();

  const markRead = useNotificationAction("read");
  const markUnread = useNotificationAction("unread");
  const dismiss = useNotificationAction("dismiss");
  const restore = useNotificationAction("restore");
  const wake = useNotificationAction("wake");
  const snooze = useSnoozeNotifications();

  const preferencesKey = userQueries.myNotificationPreferences().queryKey;
  const setKind = useApiMutation({
    mutationFn: async ({ key, off }: { key: string; off: boolean }) => {
      const preferences = await queryClient.fetchQuery(userQueries.myNotificationPreferences());
      const disabled = new Set(preferences.disabledEventTypes);
      if (disabled.has(key) === off) return preferences;
      if (off) disabled.add(key);
      else disabled.delete(key);

      return notificationPreferencesService.update({
        ...preferences,
        disabledEventTypes: [...disabled],
      });
    },
    onSuccess: (saved, { key, off }) => {
      queryClient.setQueryData(preferencesKey, saved);
      if (!off) return;

      toast.success(t("Turned off"), {
        description: t("Notifications like this one will no longer reach you."),
        action: { label: t("Undo"), onClick: () => setKind.mutate({ key, off: false }) },
      });
    },
  });

  // Two presses can land before the button shows it is working, so a row's own action
  // refuses a second run while the first is in flight.
  const running = useRef(new Set<string>());

  const handlers: NotificationRowHandlers = {
    open: (notification, href, settles = false) => {
      if (settles) {
        dismiss.mutate([notification.id]);
      } else if (notification.readAt === null) {
        markRead.mutate([notification.id]);
      }
      onClose();
      void navigate(href);
    },
    read: (id) => markRead.mutate([id]),
    unread: (id) => markUnread.mutate([id]),
    done: (id) => leave(id, () => dismiss.mutateAsync([id])),
    later: (id, until) => leave(id, () => snooze.mutateAsync({ ids: [id], until })),
    wake: (id) => leave(id, () => wake.mutateAsync([id])),
    restore: (id) => leave(id, () => restore.mutateAsync([id])),
    run: async (notification, action) => {
      if (running.current.has(notification.id)) return;
      running.current.add(notification.id);
      try {
        await runNotificationAction(action);
        leave(notification.id, () => dismiss.mutateAsync([notification.id]));
      } catch (error) {
        handleMutationError({ error });
      } finally {
        running.current.delete(notification.id);
      }
    },
    turnOff: (notification) => {
      const key = notificationPreferenceKey(notification);
      if (key !== null) setKind.mutate({ key, off: true });
    },
    copyLink: (href) => {
      void copy(new URL(href, window.location.origin).toString(), { withToast: true });
    },
  };

  return { handlers, leaving };
}
