import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { getNameInitials } from "@trenova/shared/lib/utils";
import type { EditConflict } from "@trenova/shared/types/errors";
import { useState } from "react";

type ConflictBarProps = {
  conflict: EditConflict;
  /** Absent when the editor cannot read the other save, so only "Keep mine" is offered. */
  onLoadTheirs?: () => void;
  onKeepMine: () => void;
};

/**
 * Someone else saved while this person was editing. Says who, when and what they changed,
 * and lets the person load theirs or keep their own over it.
 */
export function ConflictBar({ conflict, onLoadTheirs, onKeepMine }: ConflictBarProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [seenAt] = useState(() => Date.now() / 1000);
  const who = conflict.updatedByName || t("Someone");
  const when = formatSecondsAgo(seenAt - conflict.updatedAt);

  return (
    <div className="border-b border-border bg-warning-subtle px-4 py-2.5" role="alert">
      <div className="flex flex-wrap items-center gap-2">
        <Avatar size="sm">
          <AvatarFallback>{getNameInitials(conflict.updatedByName, "?")}</AvatarFallback>
        </Avatar>
        <span className="min-w-0 flex-1">
          {t("{0} saved changes {1} while you were editing.", who, when)}
        </span>
        {conflict.changes.length > 0 && (
          <Button size="sm" variant="ghost" onClick={() => setOpen((shown) => !shown)}>
            {open ? t("Hide") : t("See theirs")}
          </Button>
        )}
        {onLoadTheirs && (
          <Button size="sm" variant="outline" onClick={onLoadTheirs}>
            {t("Load theirs")}
          </Button>
        )}
        <Button size="sm" onClick={onKeepMine}>
          {t("Keep mine")}
        </Button>
      </div>
      {open && (
        <ul className="mt-2 list-disc pl-10 text-xs text-muted-foreground">
          {conflict.changes.map((change) => (
            <li key={change.field}>{change.label}</li>
          ))}
        </ul>
      )}
    </div>
  );
}
