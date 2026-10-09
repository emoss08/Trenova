import type { WatchedRecordChange } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { DeskIcon } from "../desk-icons";

/**
 * Records the agent was working with that changed elsewhere while it worked,
 * as it was told: which record, what happened to it, who did it and which
 * fields moved. The agent reads them again before it relies on them.
 */
export function DeskWorldChange({ changes }: { changes: readonly WatchedRecordChange[] }) {
  const t = useT();
  const me = useAuthStore((state) => state.user?.id);
  if (changes.length === 0) {
    return null;
  }

  return (
    <div className="dk-world" role="note">
      <span className="dk-world-hd">
        <DeskIcon name="alert" size={12} />
        {t("Changed while the agent was working")}
      </span>
      <ul>
        {changes.map((change) => (
          <li key={change.recordId}>
            <b>{change.label !== "" ? change.label : change.recordId}</b> {whatChanged(change, t)} ·{" "}
            {changedBy(change, me, t)}
          </li>
        ))}
      </ul>
      <span>{t("The agent was told, and reads them again before relying on them.")}</span>
    </div>
  );
}

function whatChanged(change: WatchedRecordChange, t: TranslateFn): string {
  const action = change.action.replace(/^bulk_/u, "");
  if (action === "deleted") {
    return t("Deleted");
  }
  if (action === "created") {
    return t("Created");
  }
  if (change.fields.length > 0) {
    return t("Changed: {0}", change.fields.join(", "));
  }

  return t("Changed");
}

function changedBy(change: WatchedRecordChange, me: string | undefined, t: TranslateFn): string {
  if (me !== undefined && change.actorUserId === me && change.actorType !== "agent") {
    return t("By you, elsewhere");
  }
  switch (change.actorType) {
    case "agent":
      return t("By an agent");
    case "system":
      return t("By the system");
    case "api_key":
      return t("Through the API");
    case "session_user":
      return t("By someone else");
    default:
      return t("Elsewhere");
  }
}
