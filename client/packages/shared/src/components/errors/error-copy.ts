import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  ErrorDescription,
  ErrorKind,
  ErrorTone,
} from "@trenova/shared/lib/error-presentation";
import {
  CircleAlertIcon,
  CloudOffIcon,
  FileQuestionIcon,
  HourglassIcon,
  KeyRoundIcon,
  LockIcon,
  type LucideIcon,
  RefreshCwIcon,
  ServerCrashIcon,
  TimerIcon,
  TriangleAlertIcon,
  WifiOffIcon,
} from "lucide-react";

export type ErrorCopy = {
  title: string;
  description: string;
};

export const ERROR_ICONS: Record<ErrorKind, LucideIcon> = {
  offline: WifiOffIcon,
  network: CloudOffIcon,
  "stale-build": RefreshCwIcon,
  "session-expired": KeyRoundIcon,
  forbidden: LockIcon,
  "not-found": FileQuestionIcon,
  "rate-limited": TimerIcon,
  timeout: HourglassIcon,
  server: ServerCrashIcon,
  request: CircleAlertIcon,
  unexpected: TriangleAlertIcon,
};

export const ERROR_TONE_WELL: Record<ErrorTone, string> = {
  neutral: "border-neutral-border bg-neutral-subtle text-neutral-subtle-foreground",
  info: "border-info-border bg-info-subtle text-info-subtle-foreground",
  warning: "border-warning-border bg-warning-subtle text-warning-subtle-foreground",
  danger: "border-danger-border bg-danger-subtle text-danger-subtle-foreground",
};

// errorCopy says what happened and what to do about it, in that order, without blaming the
// person or naming the machinery. The server's own message replaces the generic sentence
// only where the server wrote it for people.
export function errorCopy(t: TranslateFn, description: ErrorDescription): ErrorCopy {
  switch (description.kind) {
    case "offline":
      return {
        title: t("You're offline"),
        description: t(
          "Trenova can't reach the network. Check your connection; this view reloads on its own once you're back online.",
        ),
      };
    case "network":
      return {
        title: t("Can't reach Trenova"),
        description: t(
          "The server didn't answer. It may be restarting, or something on your network may be blocking it. Try again in a moment.",
        ),
      };
    case "stale-build":
      return {
        title: t("Trenova has been updated"),
        description: t(
          "A newer version was released while this page was open. Reload to pick it up; nothing you saved is lost.",
        ),
      };
    case "session-expired":
      return {
        title: t("Your session has ended"),
        description: t("Sign in again to carry on where you left off."),
      };
    case "forbidden":
      return {
        title: t("You don't have access to this"),
        description: t(
          "Your roles don't include permission to view this. Ask an administrator if you need it.",
        ),
      };
    case "not-found":
      return {
        title: t("We couldn't find that"),
        description: t(
          "It may have been deleted or moved, or the link that brought you here is out of date.",
        ),
      };
    case "rate-limited":
      return {
        title: t("Too many requests"),
        description: t(
          "You've reached the request limit for now. Wait a few seconds, then try again.",
        ),
      };
    case "timeout":
      return {
        title: t("This is taking too long"),
        description: t(
          "The server took too long to respond. Try again; if it keeps happening, narrow what you're loading.",
        ),
      };
    case "server":
      return {
        title: t("Something went wrong on our side"),
        description: t(
          "The server couldn't complete this request. Try again, and if it keeps failing, send the details to support.",
        ),
      };
    case "request":
      return {
        title: t("This couldn't be loaded"),
        description:
          description.detail ??
          t("The request was rejected. Check what you entered and try again."),
      };
    case "unexpected":
      return {
        title: t("Something went wrong"),
        description: t(
          "This part of the page ran into a problem it couldn't recover from. Try again, or reload the page if it keeps happening.",
        ),
      };
  }
}
