import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { getNameInitials } from "@trenova/shared/lib/utils";
import type { EditConflict } from "@trenova/shared/types/errors";
import { useState } from "react";
import { Button } from "@trenova/shared/components/ui/button";

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
    <div className="es-cx" role="alert">
      <div className="es-cxr">
        <span className="me sm">{getNameInitials(conflict.updatedByName, "?")}</span>
        <span>{t("{0} saved changes {1} while you were editing.", who, when)}</span>
        <span className="sp" />
        {conflict.changes.length > 0 && (
          <Button type="button" variant="outline" size="sm" onClick={() => setOpen((shown) => !shown)}>
            {open ? t("Hide") : t("See theirs")}
          </Button>
        )}
        {onLoadTheirs && (
          <Button type="button" variant="outline" size="sm" onClick={onLoadTheirs}>
            {t("Load theirs")}
          </Button>
        )}
        <Button type="button" variant="default" size="sm" onClick={onKeepMine}>
          {t("Keep mine")}
        </Button>
      </div>
      {open && (
        <ul className="es-cxl">
          {conflict.changes.map((change) => (
            <li key={change.field}>{change.label}</li>
          ))}
        </ul>
      )}
    </div>
  );
}
