import { useNowSeconds } from "@/hooks/use-now-seconds";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Lock01Icon, LockUnlocked01Icon, LogOut01Icon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { supportAccess } from "../../lib/queries/support-access";
import {
  SUPPORT_CONSOLE_PATH,
  accessModeLabel,
  formatRemaining,
  secondsUntil,
} from "../../lib/support-access";
import { supportAccessService } from "../../services/support-access";
import type { SessionView } from "../../types/support-access";
import { ElevateDialog } from "./elevate-dialog";

const TICK_MS = 15_000;

function leaveTo(path: string) {
  window.location.assign(path);
}

/**
 * The persistent strip shown above the app for the whole of a Trenova support session:
 * which organization, whether it is read-only or writing, how long is left, and the
 * ways out. It cannot be dismissed.
 */
export function SupportSessionBanner({
  session,
  fetchedAt,
  elevationMinutes,
}: {
  session: SessionView;
  fetchedAt: number;
  elevationMinutes: number;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const now = useNowSeconds(TICK_MS);
  const [elevating, setElevating] = useState(false);
  const [busy, setBusy] = useState(false);

  const writing = session.mode === "read_write";
  const remaining = secondsUntil(session.expiresAt, session.serverTime, fetchedAt, now);
  const writeRemaining = session.elevatedUntil
    ? secondsUntil(session.elevatedUntil, session.serverTime, fetchedAt, now)
    : 0;

  const [failed, setFailed] = useState(false);

  const exit = async () => {
    setBusy(true);
    try {
      await supportAccessService.endSession();
    } finally {
      queryClient.clear();
      leaveTo(SUPPORT_CONSOLE_PATH);
    }
  };

  const dropWrite = async () => {
    setBusy(true);
    setFailed(false);
    try {
      await supportAccessService.dropElevation();
      await queryClient.invalidateQueries({ queryKey: supportAccess.currentSession().queryKey });
      await queryClient.invalidateQueries();
    } catch {
      setFailed(true);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div
      role="status"
      aria-live="polite"
      data-testid="support-session-banner"
      data-mode={session.mode}
      className={cn(
        "flex min-h-9 shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b px-4 py-1.5 text-xs",
        writing
          ? "border-danger-border bg-danger-subtle text-danger-subtle-foreground"
          : "border-warning-border bg-warning-subtle text-warning-subtle-foreground",
      )}
    >
      {writing ? (
        <LockUnlocked01Icon className="size-3.5 shrink-0" aria-hidden="true" />
      ) : (
        <Lock01Icon className="size-3.5 shrink-0" aria-hidden="true" />
      )}
      <span className="font-semibold">{t("Trenova support session")}</span>
      <span className="min-w-0 truncate">{session.organizationName}</span>
      <span className="font-medium">{accessModeLabel(session.mode, t)}</span>
      <span>{t("Expires in {0}", formatRemaining(remaining, t))}</span>
      {writing ? (
        <span>{t("Write access ends in {0}", formatRemaining(writeRemaining, t))}</span>
      ) : null}
      {failed ? <span className="font-medium">{t("Could not return to read-only")}</span> : null}
      <span className="flex-1" />
      <div className="flex shrink-0 items-center gap-2">
        {writing ? (
          <Button type="button" size="xs" variant="outline" disabled={busy} onClick={dropWrite}>
            {t("Return to read-only")}
          </Button>
        ) : session.canElevate ? (
          <Button
            type="button"
            size="xs"
            variant="outline"
            disabled={busy}
            onClick={() => setElevating(true)}
          >
            {t("Elevate to write")}
          </Button>
        ) : null}
        <Button type="button" size="xs" variant="outline" disabled={busy} onClick={exit}>
          <LogOut01Icon className="size-3" aria-hidden="true" />
          {t("Exit")}
        </Button>
      </div>
      <ElevateDialog
        open={elevating}
        onClose={() => setElevating(false)}
        elevationMinutes={elevationMinutes}
      />
    </div>
  );
}

/** Where a person lands when their support session ends, carrying why it ended. */
export function sessionEndedPath(reason: string): string {
  const query = new URLSearchParams({ ended: reason || "ended" });
  return `${SUPPORT_CONSOLE_PATH}?${query.toString()}`;
}
