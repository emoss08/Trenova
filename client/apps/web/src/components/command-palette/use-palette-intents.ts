import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSignOut } from "@/hooks/use-sign-out";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { documentContentUrl } from "@/services/document";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useAssistantStore } from "@/stores/assistant-store";
import { useNavigationStore } from "@/stores/navigation-store";
import type { AssistantEntityRef } from "@/types/assistant";
import { useTheme } from "@trenova/shared/components/theme-provider";
import {
  useMarkAllNotificationsRead,
  useNotificationAction,
} from "@trenova/shared/hooks/use-notifications";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import type { PaletteIntent } from "./palette-model";

interface IntentRunnerOptions {
  /** Closes the palette and forgets what was typed. */
  close: () => void;
  /** Turns the palette into a question about one record. */
  askAbout: (subject: AssistantEntityRef) => void;
}

function absoluteUrl(href: string): string {
  return new URL(href, window.location.origin).toString();
}

function openInNewTab(url: string): void {
  window.open(url, "_blank", "noopener,noreferrer");
}

/**
 * Carries out whatever a row, an action or a command asks for. Everything
 * that changes the screen closes the palette first, so a dialog it opens is
 * not stacked under a dialog that is still leaving.
 */
export function usePaletteIntentRunner({ close, askAbout }: IntentRunnerOptions) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const signOut = useSignOut();
  const { setTheme } = useTheme();
  const { copy } = useCopyToClipboard();
  const openDialog = useAppDialogsStore((state) => state.openDialog);
  const toggleSidebar = useNavigationStore((state) => state.toggleSidebar);
  const markAllRead = useMarkAllNotificationsRead();
  const markRead = useNotificationAction("read");

  const { mutate: togglePin } = useApiMutation({
    mutationFn: (values: { pageUrl: string; pageTitle: string }) =>
      apiService.pageFavoriteService.togglePageFavorite(values),
    resourceName: "Page Favorite",
    onSuccess: (result) => {
      toast.success(result.favorited ? t("Pinned") : t("Unpinned"));
      void queryClient.invalidateQueries({ queryKey: queries.pageFavorite._def });
    },
  });

  const copyText = useCallback(
    async (text: string) => {
      const copied = await copy(text, { withToast: false });
      if (copied) {
        toast.success(t("Copied to clipboard"));
      } else {
        toast.error(t("Couldn't copy to the clipboard"));
      }
    },
    [copy, t],
  );

  return useCallback(
    (intent: PaletteIntent) => {
      switch (intent.type) {
        case "navigate":
          close();
          void navigate(intent.href);
          return;
        case "new-tab":
          openInNewTab(absoluteUrl(intent.href));
          close();
          return;
        case "copy":
          void copyText(intent.text);
          return;
        case "copy-link":
          void copyText(absoluteUrl(intent.href));
          return;
        case "toggle-pin":
          togglePin({ pageUrl: intent.pageUrl, pageTitle: intent.pageTitle });
          return;
        case "ask-about":
          askAbout(intent.subject);
          return;
        case "set-theme":
          setTheme(intent.theme);
          close();
          return;
        case "sign-out":
          close();
          void signOut();
          return;
        case "open-dialog":
          close();
          openDialog(intent.dialog);
          return;
        case "assistant": {
          close();
          const assistant = useAssistantStore.getState();
          if (intent.mode === "new-chat") {
            assistant.setActiveThreadId(null);
          }
          assistant.openWidget();
          return;
        }
        case "toggle-sidebar":
          close();
          toggleSidebar();
          return;
        case "mark-notification-read":
          markRead.mutate([intent.notificationId]);
          return;
        case "mark-all-notifications-read":
          markAllRead.mutate();
          close();
          return;
        case "open-document":
          openInNewTab(documentContentUrl(intent.documentId, intent.disposition));
          close();
          return;
      }
    },
    [
      askAbout,
      close,
      copyText,
      markAllRead,
      markRead,
      navigate,
      openDialog,
      setTheme,
      signOut,
      togglePin,
      toggleSidebar,
    ],
  );
}
