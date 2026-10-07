import type { AgentVersion } from "@/lib/graphql/agent-builder";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { Ic } from "../../kit/ic";
import { Pop } from "../../kit/pop";
import type { BlockId } from "./block";
import type { CheckStatus } from "./builder-model";

export type RailEntry = { id: BlockId; label: string; summary: string; status: CheckStatus };

type BuilderRailProps = {
  entries: RailEntry[];
  active: BlockId;
  test: { status: CheckStatus; summary: string };
  ready: { done: number; of: number };
  modeNote: string;
  versions: readonly AgentVersion[] | null;
  currentVersion: number;
  restoring: boolean;
  onJump: (id: BlockId) => void;
  onTry: () => void;
  onRestore: (version: number) => void;
};

/** The builder's checklist: each part, how far along it is, and how ready the agent is. */
export function BuilderRail({
  entries,
  active,
  test,
  ready,
  modeNote,
  versions,
  currentVersion,
  restoring,
  onJump,
  onTry,
  onRestore,
}: BuilderRailProps) {
  const t = useT();
  const [history, setHistory] = useState(false);
  const latest = versions?.[0];

  return (
    <nav className="ab-r" aria-label={t("Build")}>
      <div className="ab-rh">{t("Build")}</div>
      {entries.map((entry) => (
        <button
          key={entry.id}
          type="button"
          className={cn("ri", active === entry.id && "on")}
          aria-current={active === entry.id ? "true" : undefined}
          onClick={() => onJump(entry.id)}
        >
          <Status status={entry.status} />
          <span className="ri-t">
            <b>{entry.label}</b>
            <em>{entry.summary}</em>
          </span>
        </button>
      ))}
      <button type="button" className="ri" onClick={onTry}>
        <Status status={test.status} />
        <span className="ri-t">
          <b>{t("Try it")}</b>
          <em>{test.summary}</em>
        </span>
      </button>
      <div className="ab-rf">
        <div className="rdy">
          <Progress value={ready.of ? ready.done / ready.of : 0} />
          <span>
            <b>
              {ready.done === ready.of
                ? t("Ready to go live")
                : ready.of - ready.done === 1
                  ? t("1 left before going live")
                  : t("{0} left before going live", ready.of - ready.done)}
            </b>
            <em>{modeNote}</em>
          </span>
        </div>
        {latest && (
          <div className="rel">
            <button
              type="button"
              className="ab-ver"
              aria-expanded={history}
              onClick={() => setHistory((open) => !open)}
            >
              <Ic n="timeline" s={12} />
              <span>
                {[
                  `v${latest.version}`,
                  formatUnixDateMedium(latest.createdAt),
                  latest.author?.name ?? t("Trenova"),
                ].join(" · ")}
              </span>
            </button>
            {history && versions && (
              <Pop className="hist" label={t("Versions")} onClose={() => setHistory(false)}>
                <div className="mn-h">{t("Versions")}</div>
                {versions.map((version) => {
                  const current = version.version === currentVersion;
                  return (
                    <div key={version.id} className={cn("vh-r", current && "cur")}>
                      <span className="vh-d" />
                      <div>
                        <b>{version.summary}</b>
                        <span>
                          {`${version.author?.name ?? t("Trenova")} · ${formatUnixDateTimeShort(version.createdAt)}`}
                        </span>
                      </div>
                      {current ? (
                        <span className="tg">{t("Current")}</span>
                      ) : (
                        <button
                          type="button"
                          className="btn sm"
                          disabled={restoring}
                          onClick={() => {
                            setHistory(false);
                            onRestore(version.version);
                          }}
                        >
                          {t("Restore")}
                        </button>
                      )}
                    </div>
                  );
                })}
              </Pop>
            )}
          </div>
        )}
      </div>
    </nav>
  );
}

function Status({ status }: { status: CheckStatus }) {
  return (
    <span className={cn("ri-s", status)}>
      {status === "ok" ? <Ic n="check" s={10} w={3} /> : status === "warn" ? "!" : null}
    </span>
  );
}

function Progress({ value, size = 30 }: { value: number; size?: number }) {
  const radius = size / 2 - 3;
  const circumference = 2 * Math.PI * radius;
  const half = size / 2;
  return (
    <svg width={size} height={size} className="prog" aria-hidden>
      <circle cx={half} cy={half} r={radius} />
      <circle
        cx={half}
        cy={half}
        r={radius}
        className="f"
        strokeDasharray={circumference}
        strokeDashoffset={circumference * (1 - value)}
        transform={`rotate(-90 ${half} ${half})`}
      />
    </svg>
  );
}
