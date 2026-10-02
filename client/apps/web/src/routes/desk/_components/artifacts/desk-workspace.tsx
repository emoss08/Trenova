import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, downloadTextFile, slugify } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import { toast } from "sonner";
import type { LiveArtifacts } from "../desk-layout";
import { useOutsideDismiss } from "../use-outside-dismiss";
import { tableViewCsv } from "./artifact-export";
import { tableViewFrom } from "./artifact-payloads";
import {
  ArtIcon,
  DeskArtKindIcon,
  deskArtKind,
  deskArtKindLabel,
} from "./desk-art-kinds";
import { DeskArtifactBrowser } from "./desk-artifact-browser";
import { DeskArtifactsEmpty } from "./desk-artifacts-empty";
import {
  DeskDecisionBody,
  DeskDiffBody,
  DeskDocBody,
  DeskEmailBody,
  DeskPlanBody,
  DeskRateBody,
  DeskRecordBody,
  DeskReportRunBody,
  DeskViewBody,
} from "./desk-bodies";
import { groupLineages, lineageContaining, type ArtifactLineage } from "./desk-lineage";
import { DeskTableBody, changedCells, gridOf } from "./desk-table-body";

/** How long a just-arrived artifact keeps its "New" mark. */
const NEW_MS = 6000;

function shortTime(at: number): string {
  return new Date(at * 1000).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

/** Where an artifact opens: its conversation, with the artifact named. */
export function artifactLink(artifact: AssistantArtifact): string {
  return `/desk/t/${artifact.threadId}?a=${artifact.lineageId || artifact.id}`;
}

/**
 * The artifacts as a stack of cards: the open one in front, the rest peeking
 * behind it. Pointing at the stack fans them out to pick from, with the way
 * to everything at the bottom.
 */
function ArtStack({
  lineages,
  active,
  newest,
  total,
  height,
  onPick,
  onFan,
  onAll,
}: {
  lineages: ArtifactLineage[];
  active: ArtifactLineage;
  newest: string | null;
  total: number;
  height: number;
  onPick: (id: string) => void;
  onFan: (fanned: boolean) => void;
  onAll: () => void;
}) {
  const t = useT();
  const [fan, setFan] = useState(false);
  const timer = useRef<number | undefined>(undefined);
  const order = [active, ...lineages.filter((lineage) => lineage.id !== active.id)];
  const count = order.length;
  const gap = Math.max(34, Math.min(54, (height - 40) / (count + 1)));
  const open = () => {
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      setFan(true);
      onFan(true);
    }, 110);
  };
  const close = () => {
    window.clearTimeout(timer.current);
    setFan(false);
    onFan(false);
  };
  useEffect(() => () => window.clearTimeout(timer.current), []);

  return (
    <div
      className={cn("dk-ax-stack", fan && "dk-fan")}
      onMouseEnter={open}
      onMouseLeave={close}
      onFocus={open}
      onBlur={(event) => !event.currentTarget.contains(event.relatedTarget) && close()}
      style={{ "--dk-fanh": `${count * gap + 52}px` } as CSSProperties}
    >
      {order.map((lineage, index) => {
        const artifact = lineage.latest;
        const kind = deskArtKind(artifact);
        const offset = fan ? index * gap : Math.min(index, 2) * 6;
        const scale = fan ? 1 : 1 - Math.min(index, 2) * 0.035;
        return (
          <button
            key={lineage.id}
            type="button"
            className={cn("dk-ax-card", index === 0 && "dk-front", lineage.id === newest && "dk-nw")}
            onClick={() => {
              onPick(lineage.id);
              close();
            }}
            tabIndex={index === 0 || fan ? 0 : -1}
            style={{
              zIndex: count - index,
              transform: `translateY(${offset}px) scale(${scale})`,
              opacity: !fan && index > 2 ? 0 : 1,
              transitionDelay: `${fan ? index * 16 : 0}ms`,
            }}
          >
            <span className={cn("dk-ax-ki", `dk-k-${kind}`)}>
              <DeskArtKindIcon kind={kind} />
            </span>
            <span className="dk-ax-ct">
              <b>{artifact.title}</b>
              <span>
                {t(deskArtKindLabel(kind))} · {shortTime(artifact.createdAt)}
              </span>
            </span>
            {lineage.id === newest && <span className="dk-ax-new">{t("New")}</span>}
            {lineage.versions.length > 1 && (
              <span className="dk-ax-ver">v{lineage.versions.length}</span>
            )}
            {index === 0 && !fan && (
              <span className="dk-ax-cnt">
                {total}
                <ArtIcon name="down" size={11} stroke={2.2} />
              </span>
            )}
          </button>
        );
      })}
      <button
        type="button"
        className="dk-ax-card dk-ax-all"
        onClick={() => {
          onAll();
          close();
        }}
        tabIndex={fan ? 0 : -1}
        style={{
          zIndex: 0,
          transform: `translateY(${fan ? count * gap : 12}px) scale(${fan ? 1 : 0.93})`,
          opacity: fan ? 1 : 0,
          transitionDelay: `${fan ? count * 16 : 0}ms`,
        }}
      >
        <span className="dk-ax-ki">
          <ArtIcon name="table" size={15} />
        </span>
        <span className="dk-ax-ct">
          <b>{t("All {0} artifacts", total)}</b>
          <span>{t("Search everything this conversation made")}</span>
        </span>
        <span className="dk-kbd">⌘J</span>
      </button>
    </div>
  );
}

/** Which version of an artifact is showing, and the others to go back to. */
function VersionPicker({
  lineage,
  index,
  onChange,
}: {
  lineage: ArtifactLineage;
  index: number;
  onChange: (index: number) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLSpanElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutsideDismiss(rootRef, open, close);
  const last = lineage.versions.length - 1;

  return (
    <span className="dk-axv" ref={rootRef}>
      <button
        type="button"
        className={cn("dk-axv-b", open && "dk-on", index !== last && "dk-old")}
        title={t("Versions")}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        v{index + 1}
        <ArtIcon name="down" size={9} stroke={2.6} />
      </button>
      {open && (
        <div className="dk-axv-pop" role="listbox">
          {[...lineage.versions].reverse().map((version) => {
            const at = lineage.versions.indexOf(version);
            return (
              <button
                key={version.id}
                type="button"
                role="option"
                aria-selected={at === index}
                className={cn("dk-axv-r", at === index && "dk-on")}
                onClick={() => {
                  onChange(at);
                  setOpen(false);
                }}
              >
                <b>v{at + 1}</b>
                <span>
                  <em>{at === 0 ? t("First read") : t("Read again")}</em>
                  <i>
                    {shortTime(version.createdAt)}
                    {at === last ? ` · ${t("Latest")}` : ""}
                  </i>
                </span>
                {at === index && <ArtIcon name="check" size={12} stroke={2.4} />}
              </button>
            );
          })}
        </div>
      )}
    </span>
  );
}

/** Where an artifact came from: the tool, how many rows, how many calls, what changed. */
function Provenance({
  artifact,
  previous,
}: {
  artifact: AssistantArtifact;
  previous: AssistantArtifact | null;
}) {
  const t = useT();
  const tool =
    typeof artifact.payload.tool === "string" && artifact.payload.tool !== ""
      ? artifact.payload.tool
      : artifact.kind;
  const tabular = artifact.kind === "table_view" || artifact.kind === "report_preview";
  const grid = tabular && !("path" in artifact.payload) ? gridOf(artifact) : null;
  const calls =
    artifact.kind === "table_view" && !("path" in artifact.payload)
      ? tableViewFrom(artifact).calls
      : 0;
  const changes =
    grid && previous ? changedCells(grid, gridOf(previous)).size : 0;

  return (
    <div className="dk-ax-prov dk-sm">
      <code>{tool}</code>
      {grid && <span>{t("{0, plural, one {# row} other {# rows}}", grid.rowCount)}</span>}
      {calls > 0 && <span>{t("{0, plural, one {# call} other {# calls}}", calls)}</span>}
      {changes > 0 && <span className="dk-ax-chg">{t("{0} changed", changes)}</span>}
    </div>
  );
}

function ArtifactBody({
  artifact,
  previous,
  versions,
}: {
  artifact: AssistantArtifact;
  previous: AssistantArtifact | null;
  versions: React.ReactNode;
}) {
  const t = useT();
  switch (artifact.kind) {
    case "table_view":
      return "path" in artifact.payload ? (
        <DeskViewBody artifact={artifact} />
      ) : (
        <DeskTableBody artifact={artifact} previous={previous} versions={versions} />
      );
    case "report_preview":
      return <DeskTableBody artifact={artifact} previous={previous} versions={versions} />;
    case "report_run":
      return <DeskReportRunBody artifact={artifact} />;
    case "entity_card":
      return <DeskRecordBody artifact={artifact} />;
    case "rate_explanation":
      return <DeskRateBody artifact={artifact} />;
    case "email_draft":
      return <DeskEmailBody artifact={artifact} />;
    case "plan":
      return <DeskPlanBody artifact={artifact} />;
    case "run_diff":
      return <DeskDiffBody artifact={artifact} />;
    case "document":
    case "briefing":
      return <DeskDocBody artifact={artifact} />;
    case "navigation":
    case "dashboard_ref":
      return <DeskViewBody artifact={artifact} />;
    case "decision_request":
      return <DeskDecisionBody artifact={artifact} />;
    default:
      return (
        <div className="dk-ax-pad dk-ax-oldnote">
          {t("This kind of artifact cannot be shown here yet.")}
        </div>
      );
  }
}

/**
 * Everything a conversation made, beside it.
 *
 * The open artifact sits in front of a small stack of the others; pointing at
 * the stack fans them out, and ⌘J opens all of them to search. Under the
 * stack is where the artifact came from, then the artifact itself drawn for
 * its kind, then a line to link, pin, export or open it on its own page. A
 * table read again later in the conversation is one artifact with versions:
 * the latest shows, the earlier ones are a click away, and the cells that
 * changed since the version before are marked.
 */
export function DeskWorkspace({
  threadId,
  liveArtifacts,
  onClose,
}: {
  threadId: string;
  liveArtifacts: LiveArtifacts;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const artifactsQuery = useQuery(queries.assistant.artifacts(threadId));
  const lineages = useMemo(
    () => groupLineages(artifactsQuery.data?.results ?? []),
    [artifactsQuery.data],
  );
  const remembered = useDeskStore((state) => state.activeArtifactByThread[threadId]);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const active = lineageContaining(lineages, remembered) ?? lineages[0] ?? null;

  const [versionByLineage, setVersionByLineage] = useState<Record<string, number>>({});
  const [fanning, setFanning] = useState(false);
  const [browsing, setBrowsing] = useState(false);
  const [newest, setNewest] = useState<string | null>(null);

  const newestLive = liveArtifacts.ids.at(-1);
  const liveRevision = liveArtifacts.revision;
  useEffect(() => {
    if (liveRevision === 0) {
      return;
    }
    void queryClient.invalidateQueries({
      queryKey: queries.assistant.artifacts(threadId).queryKey,
    });
    if (newestLive) {
      setActiveArtifact(threadId, newestLive);
    }
  }, [liveRevision, newestLive, queryClient, setActiveArtifact, threadId]);

  const newestLineage = lineageContaining(lineages, newestLive)?.id ?? null;
  useEffect(() => {
    if (!newestLineage || liveRevision === 0) {
      return;
    }
    setNewest(newestLineage);
    const timer = window.setTimeout(() => setNewest(null), NEW_MS);
    return () => window.clearTimeout(timer);
  }, [liveRevision, newestLineage]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "j") {
        event.preventDefault();
        setBrowsing((value) => !value);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const paneRef = useRef<HTMLDivElement>(null);
  const [height, setHeight] = useState(700);
  useLayoutEffect(() => {
    const element = paneRef.current;
    if (!element) {
      return;
    }
    const observer = new ResizeObserver(() => setHeight(element.clientHeight));
    observer.observe(element);
    return () => observer.disconnect();
  }, [active === null]);

  const pinMutation = useApiMutation({
    mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) =>
      apiService.assistantService.pinArtifact(threadId, id, pinned),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queries.assistant.artifacts(threadId).queryKey }),
    resourceName: "Artifact",
  });

  const open = useCallback(
    (id: string) => setActiveArtifact(threadId, id),
    [setActiveArtifact, threadId],
  );

  if (artifactsQuery.isLoading) {
    return (
      <div className="dk-apx" ref={paneRef}>
        <div className="dk-ax-top">
          <div className="dk-ax-stack">
            <div className="dk-ax-card dk-front dk-sk-card" />
          </div>
        </div>
        <div className="dk-ax-body" />
        <div className="dk-ax-foot" />
      </div>
    );
  }

  if (artifactsQuery.isError) {
    return (
      <div className="dk-apx dk-apx-solo">
      <div className="dk-axe">
        <div className="dk-axe-in">
          <b className="dk-axe-t">{t("This conversation's artifacts could not be loaded.")}</b>
          <button
            type="button"
            className="dk-ec-btn"
            style={{ marginTop: 14 }}
            onClick={() => void artifactsQuery.refetch()}
          >
            {t("Try again")}
          </button>
        </div>
      </div>
      </div>
    );
  }

  if (active === null) {
    return (
      <div className="dk-apx dk-apx-solo">
        <DeskArtifactsEmpty onClose={onClose} />
      </div>
    );
  }

  const last = active.versions.length - 1;
  const index = Math.min(versionByLineage[active.id] ?? last, last);
  const artifact = active.versions[index];
  const previous = index > 0 ? active.versions[index - 1] : null;
  const position = lineages.findIndex((lineage) => lineage.id === active.id);
  const go = (step: number) => {
    const next = lineages[(position + step + lineages.length) % lineages.length];
    open(next.id);
  };
  const versions =
    active.versions.length > 1 ? (
      <VersionPicker
        lineage={active}
        index={index}
        onChange={(at) => setVersionByLineage((current) => ({ ...current, [active.id]: at }))}
      />
    ) : null;
  const tabular =
    (artifact.kind === "table_view" && !("path" in artifact.payload)) ||
    artifact.kind === "report_preview";
  const link = artifactLink(artifact);
  const copyLink = () => {
    void navigator.clipboard?.writeText(window.location.origin + link);
    toast.success(t("Link copied"));
  };
  const exportCsv = () => {
    if (artifact.kind === "table_view") {
      downloadTextFile(
        tableViewCsv(tableViewFrom(artifact), t),
        `${slugify(artifact.title) || "artifact"}.csv`,
        "text/csv",
      );
    }
  };

  return (
    <div className={cn("dk-apx", fanning && "dk-fanning", browsing && "dk-browsing")} ref={paneRef}>
      {browsing ? (
        <DeskArtifactBrowser
          lineages={lineages}
          activeId={active.id}
          onPick={(id) => {
            open(id);
            setBrowsing(false);
          }}
          onBack={() => setBrowsing(false)}
          onClose={() => {
            setBrowsing(false);
            onClose();
          }}
        />
      ) : (
        <>
          <div className="dk-ax-top">
            <ArtStack
              lineages={lineages.slice(0, 8)}
              active={active}
              newest={newest}
              total={lineages.length}
              height={height}
              onPick={open}
              onFan={setFanning}
              onAll={() => setBrowsing(true)}
            />
            <div className="dk-ax-nav">
              <button type="button" className="dk-ax-ib" title={t("Previous")} aria-label={t("Previous")} onClick={() => go(-1)}>
                <ArtIcon name="up" size={13} stroke={2.2} />
              </button>
              <button type="button" className="dk-ax-ib" title={t("Next")} aria-label={t("Next")} onClick={() => go(1)}>
                <ArtIcon name="down" size={13} stroke={2.2} />
              </button>
              <button type="button" className="dk-ax-ib" title={t("Close")} aria-label={t("Hide artifacts")} onClick={onClose}>
                <ArtIcon name="x" size={13} stroke={2.2} />
              </button>
            </div>
          </div>
          <div className="dk-ax-body" key={artifact.id}>
            {index !== last && (
              <div className="dk-ax-oldnote">
                {t("Version {0} of {1}, read {2}", index + 1, last + 1, shortTime(artifact.createdAt))}
              </div>
            )}
            <Provenance artifact={artifact} previous={previous} />
            <ArtifactBody artifact={artifact} previous={previous} versions={versions} />
          </div>
          <div className="dk-ax-foot">
            <button type="button" className="dk-ax-link" onClick={copyLink} title={t("Copy link")}>
              <ArtIcon name="copy" size={12} />
              <span>
                desk/t/…/a/<b>{slugify(artifact.title) || artifact.id}</b>
              </span>
            </button>
            <span className="flex-1" />
            <button
              type="button"
              className={cn("dk-ax-ib", artifact.pinned && "dk-on")}
              title={artifact.pinned ? t("Unpin") : t("Pin to conversation")}
              aria-pressed={artifact.pinned}
              onClick={() => pinMutation.mutate({ id: artifact.id, pinned: !artifact.pinned })}
            >
              <ArtIcon name="pin" size={14} />
            </button>
            {tabular && artifact.kind === "table_view" && (
              <button type="button" className="dk-ax-ib" title={t("Export CSV")} onClick={exportCsv}>
                <ArtIcon name="dl" size={14} />
              </button>
            )}
            <a className="dk-ax-ib" href={link} target="_blank" rel="noreferrer" title={t("Open on its own page")}>
              <ArtIcon name="ext" size={14} />
            </a>
          </div>
        </>
      )}
    </div>
  );
}
