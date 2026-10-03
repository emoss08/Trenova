import { kfmt } from "@/components/assistant/compaction";
import type { AssistantMessage } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { DeskIcon, type DeskIconName } from "../desk-icons";

/** The summary's points, one per bullet the model wrote. */
export function summaryPoints(content: string): string[] {
  return content
    .split("\n")
    .map((line) => line.trim().replace(/^[-*•]\s+/, ""))
    .filter((line) => line !== "");
}

/** A point with the names and figures the model set in bold kept bold. */
function Point({ text }: { text: string }) {
  const parts = text.split(/\*\*(.+?)\*\*/g);
  return <>{parts.map((part, index) => (index % 2 === 1 ? <b key={index}>{part}</b> : part))}</>;
}

/**
 * Where the conversation was compacted: a rule across the thread with a pill
 * saying how many earlier messages were summarized and how far the context
 * fell. Opening it shows what the agent carries forward and what stayed in
 * full. Everything above it is dimmed, because the agent now reads it only
 * through the summary.
 */
export function DeskCompactMark({ message }: { message: AssistantMessage }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const record = message.compaction;
  const points = summaryPoints(message.content);

  const kept: { key: string; icon: DeskIconName; label: string }[] = [];
  if (record?.kept.includes("recent")) {
    kept.push({ key: "recent", icon: "chat", label: t("Latest turns") });
  }
  kept.push({ key: "facts", icon: "pin", label: t("Pinned facts") });
  if (record?.kept.includes("approvals")) {
    kept.push({ key: "approvals", icon: "inbox", label: t("Pending approvals") });
  }

  return (
    <div className={cn("dk-cmk", open && "dk-open")}>
      <div className="dk-cmk-l">
        <span className="dk-cmk-rule" />
        <button
          type="button"
          className="dk-cmk-p"
          aria-expanded={open}
          onClick={() => setOpen((value) => !value)}
        >
          <span className="dk-cmk-ic">
            <DeskIcon name="compact" size={12} stroke={2} />
          </span>
          <span>
            {record?.auto ? t("Auto-compacted") : t("Compacted")} ·{" "}
            {t(
              "{0, plural, one {# earlier message summarized} other {# earlier messages summarized}}",
              record?.summarized ?? 0,
            )}
          </span>
          {record && record.before > 0 && (
            <span className="dk-cmk-n">
              {kfmt(record.before)} → {kfmt(record.after)}
            </span>
          )}
          <span className="dk-cmk-cv">
            <DeskIcon name="chevR" size={11} stroke={2} />
          </span>
        </button>
        <span className="dk-cmk-rule" />
      </div>
      {open && (
        <div className="dk-cmk-sum">
          <div className="dk-cmk-h">{t("What the agent carries forward")}</div>
          <ul>
            {points.map((point, index) => (
              <li key={index}>
                <Point text={point} />
              </li>
            ))}
          </ul>
          <div className="dk-cmk-h">{t("Kept in full")}</div>
          <div className="dk-cmk-keep">
            {kept.map((item) => (
              <span key={item.key}>
                <DeskIcon name={item.icon} size={11} />
                {item.label}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
