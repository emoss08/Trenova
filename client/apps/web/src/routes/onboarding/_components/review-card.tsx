import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { Kbd } from "./keyboard-hint";

export type ReviewRow = {
  label: string;
  value: string;
  step: number;
  mono?: boolean;
};

export type ReviewGroup = {
  title: string;
  rows: readonly ReviewRow[];
};

/**
 * Everything the conversation collected, printed as one card. Every line is a button
 * that goes back to the turn that asked it; ⌘/Ctrl+Enter or the footer button finishes.
 */
export function ReviewCard({
  groups,
  onEdit,
  onFinish,
}: {
  groups: readonly ReviewGroup[];
  onEdit: (step: number) => void;
  onFinish: () => void;
}) {
  const t = useT();
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        onFinish();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const offsets = groups.map((_, groupIndex) =>
    groups.slice(0, groupIndex).reduce((count, group) => count + group.rows.length, 0),
  );

  return (
    <div className="nv-rv">
      {groups.map((group, groupIndex) => (
        <div key={group.title} className="nv-rv-g" role="group" aria-label={group.title}>
          <h3 className="nv-rv-h">{group.title}</h3>
          {group.rows.map((row, rowIndex) => {
            const delay = 120 + (offsets[groupIndex] + rowIndex) * 45;
            return (
              <button
                key={row.label}
                type="button"
                className="nv-rv-r"
                style={{ animationDelay: `${delay}ms` }}
                onClick={() => onEdit(row.step)}
              >
                <span className="nv-k">{row.label}</span>
                <span
                  className={row.mono && row.value ? "nv-v nv-mono" : "nv-v"}
                  data-empty={!row.value}
                >
                  {row.value || "—"}
                </span>
              </button>
            );
          })}
        </div>
      ))}
      <div className="nv-rv-f">
        <span className="nv-n">{t("You can change all of this in Organization settings.")}</span>
        <span className="nv-sp" />
        <button type="button" className="nv-bt" data-ink="true" onClick={onFinish}>
          {t("Looks good, finish setup")} <Kbd>{"⌘↵"}</Kbd>
        </button>
      </div>
    </div>
  );
}
