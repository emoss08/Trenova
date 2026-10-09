import { Button } from "@trenova/shared/components/ui/button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { artifactCsvUrl } from "@/services/assistant";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, downloadFromUrl } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
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
import { useOutsideDismiss } from "@/components/desk-chat/use-outside-dismiss";
import { tableViewFrom } from "./artifact-payloads";
import { ArtIcon, DeskArtKindIcon, deskArtKind, deskArtKindName } from "./desk-art-kinds";
import { DeskArtifactBrowser } from "./desk-artifact-browser";
import { DeskArtifactsEmpty } from "./desk-artifacts-empty";
import {
  DeskDecisionBody,
  DeskDiffBody,
  DeskEmailBody,
  DeskPlanBody,
  DeskRateBody,
  DeskRecordBody,
  DeskReportRunBody,
  DeskViewBody,
} from "./desk-bodies";
import { DeskDocBody } from "./desk-doc-body";
import { VERSION_OPTION_CLASS, versionTriggerClass } from "./desk-version-classes";
import { DeskExtractBody } from "./desk-extract-body";
import { groupLineages, lineageContaining, type ArtifactLineage } from "./desk-lineage";
import { olderNote, pushRecent, stackLineages, stepLineage } from "./desk-workspace-state";
import { NO_PENDING_LOOKUPS, withoutPendingLookups } from "./pending-lookups";
import {
  DeskReportBars,
  DeskTableBody,
  barsOf,
  changedCells,
  gridOf,
  versionNote,
} from "./desk-table-body";

/** How long a just-arrived artifact keeps its "New" mark. */
const NEW_MS = 6000;

function shortTime(at: number): string {
  return new Date(at * 1000).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

/** Where an artifact opens: its conversation, with the artifact named by its slug. */
export function artifactLink(
  artifact: Pick<AssistantArtifact, "threadId" | "slug" | "id" | "lineageId">,
): string {
  return `/desk/c/${artifact.threadId}/a/${artifact.slug || artifact.lineageId || artifact.id}`;
}

/** The artifact on a page of its own, the whole of it and nothing else. */
export function artifactPageLink(
  artifact: Pick<AssistantArtifact, "threadId" | "slug" | "id" | "lineageId">,
): string {
  return `${artifactLink(artifact)}/page`;
}

/** The session's recently opened lineages per conversation; gone with the tab. */
const recentByThread = new Map<string, string[]>();

/**
 * The artifacts as a stack of cards: the open one in front, the rest peeking
 * behind it. A click on the front card fans them out to pick from, with the
 * way to everything at the bottom; a click outside or Esc folds them back.
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
  const rootRef = useRef<HTMLDivElement>(null);
  const order = [active, ...lineages.filter((lineage) => lineage.id !== active.id)];
  const count = order.length;
  const gap = Math.max(34, Math.min(54, (height - 40) / (count + 1)));
  const open = () => {
    setFan(true);
    onFan(true);
  };
  const close = useCallback(() => {
    setFan(false);
    onFan(false);
  }, [onFan]);
  // Esc folds the fan and goes no further: the pane stays open.
  useOutsideDismiss(rootRef, fan, close);

  return (
    <div
      ref={rootRef}
      className={cn("dk-ax-stack", fan && "dk-fan")}
      style={{ "--dk-fanh": `${count * gap + 52}px` } as CSSProperties}
    >
      {order.map((lineage, index) => {
        const artifact = lineage.latest;
        const kind = deskArtKind(artifact);
        const offset = fan ? index * gap : Math.min(index, 2) * 6;
        const scale = fan ? 1 : 1 - Math.min(index, 2) * 0.035;
        return (
          <Button
            key={lineage.id}
            variant="bare"
            size="bare"
            className={cn(
              "dk-ax-card",
              index === 0 && "dk-front",
              lineage.id === newest && "dk-nw",
            )}
            onClick={() => {
              if (index === 0 && !fan) {
                open();
                return;
              }
              onPick(lineage.id);
              close();
            }}
            title={index === 0 && !fan ? t("Switch artifact") : undefined}
            aria-expanded={index === 0 ? fan : undefined}
            aria-hidden={index === 0 || fan ? undefined : true}
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
                {deskArtKindName(kind, t)} · {shortTime(artifact.createdAt)}
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
          </Button>
        );
      })}
      <Button
        variant="bare"
        size="bare"
        className="dk-ax-card border border-dashed border-dsk-b-strong bg-dsk-sunken! ring-1! ring-dsk-b-sub hover:bg-dsk-hover!"
        onClick={() => {
          onAll();
          close();
        }}
        tabIndex={fan ? 0 : -1}
        aria-hidden={fan ? undefined : true}
        aria-keyshortcuts="Meta+J"
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
        <span className="dk-kbd" aria-hidden>
          ⌘J
        </span>
      </Button>
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
  const buttonRef = useRef<HTMLButtonElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  // Closing hands focus back to the button when it was inside the list, so
  // it is not dropped on the page when the list goes.
  const close = useCallback(() => {
    if (rootRef.current?.contains(document.activeElement)) {
      buttonRef.current?.focus();
    }
    setOpen(false);
  }, []);
  useOutsideDismiss(rootRef, open, close);
  const last = lineage.versions.length - 1;
  useEffect(() => {
    if (open) {
      listRef.current?.querySelector<HTMLElement>("[aria-selected='true']")?.focus();
    }
  }, [open]);
  const onListKey = (event: React.KeyboardEvent<HTMLDivElement>) => {
    const options = [...(listRef.current?.querySelectorAll<HTMLElement>("[role='option']") ?? [])];
    const at = options.indexOf(document.activeElement as HTMLElement);
    const next =
      event.key === "ArrowDown"
        ? Math.min(options.length - 1, at + 1)
        : event.key === "ArrowUp"
          ? Math.max(0, at - 1)
          : event.key === "Home"
            ? 0
            : event.key === "End"
              ? options.length - 1
              : null;
    if (next !== null) {
      event.preventDefault();
      options[next]?.focus();
    } else if (event.key === "Tab") {
      close();
    }
  };

  return (
    <span className="dk-axv" ref={rootRef}>
      <Button
        variant="bare"
        size="bare"
        className={versionTriggerClass(index !== last)}
        ref={buttonRef}
        title={t("Versions")}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        v{index + 1}
        <ArtIcon name="down" size={9} stroke={2.6} />
      </Button>
      {open && (
        <div
          className="dk-axv-pop"
          role="listbox"
          aria-label={t("Versions")}
          ref={listRef}
          onKeyDown={onListKey}
        >
          {[...lineage.versions].reverse().map((version) => {
            const at = lineage.versions.indexOf(version);
            return (
              <Button
                key={version.id}
                variant="bare"
                size="bare"
                role="option"
                aria-selected={at === index}
                tabIndex={at === index ? 0 : -1}
                className={VERSION_OPTION_CLASS}
                onClick={() => {
                  onChange(at);
                  close();
                }}
              >
                <b>v{at + 1}</b>
                <span>
                  <em>{versionNote(version, lineage.versions[at - 1] ?? null, t)}</em>
                  <i>
                    {shortTime(version.createdAt)}
                    {at === last ? ` · ${t("Latest")}` : ""}
                  </i>
                </span>
                {at === index && <ArtIcon name="check" size={12} stroke={2.4} />}
              </Button>
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
  const tool = typeof artifact.payload.tool === "string" ? artifact.payload.tool : "";
  const tabular = artifact.kind === "table_view" || artifact.kind === "report_preview";
  const grid = tabular && !("path" in artifact.payload) ? gridOf(artifact) : null;
  const calls =
    artifact.kind === "table_view" && !("path" in artifact.payload)
      ? tableViewFrom(artifact).calls
      : 0;
  const changes = grid && previous ? changedCells(grid, gridOf(previous)).size : 0;

  if (tool === "" && !grid && calls === 0) {
    return null;
  }

  return (
    <div className="dk-ax-prov dk-sm">
      {tool !== "" && <code>{tool}</code>}
      {grid && <span>{t("{0, plural, one {# row} other {# rows}}", grid.rowCount)}</span>}
      {calls > 0 && <span>{t("{0, plural, one {# call} other {# calls}}", calls)}</span>}
      {changes > 0 && <span className="dk-ax-chg">{t("{0} changed", changes)}</span>}
    </div>
  );
}

export function ArtifactBody({
  artifact,
  previous,
  versions,
  lineage,
  onOpenArtifact,
}: {
  artifact: AssistantArtifact;
  previous: AssistantArtifact | null;
  versions: React.ReactNode;
  /** Every version, for a document's own version list. */
  lineage: ArtifactLineage;
  /** Opens another of the conversation's artifacts, from a citation or a row. */
  onOpenArtifact: (id: string) => void;
}) {
  const t = useT();
  switch (artifact.kind) {
    case "table_view":
      return "path" in artifact.payload ? (
        <DeskViewBody artifact={artifact} />
      ) : (
        <DeskTableBody artifact={artifact} previous={previous} versions={versions} />
      );
    case "report_preview": {
      const bars = barsOf(gridOf(artifact));
      return bars ? (
        <>
          {versions && <div className="dk-ax-tools dk-end">{versions}</div>}
          <DeskReportBars artifact={artifact} bars={bars} />
        </>
      ) : (
        <DeskTableBody artifact={artifact} previous={previous} versions={versions} />
      );
    }
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
      return (
        <DeskDocBody
          key={lineage.id}
          lineage={lineage}
          shown={artifact}
          onOpenArtifact={onOpenArtifact}
        />
      );
    case "extraction":
      return <DeskExtractBody artifact={artifact} />;
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

/** Reads the lineage the store remembers when the first page does not hold it. */
function useRememberedLineage(
  threadId: string,
  lineages: readonly ArtifactLineage[],
  remembered: string | undefined,
  loaded: boolean,
): ArtifactLineage | null {
  const inPage = lineageContaining(lineages, remembered);
  const lineageQuery = useQuery({
    ...queries.assistant.artifactLineage(threadId, remembered ?? ""),
    enabled: loaded && remembered !== undefined && inPage === null,
    retry: false,
  });
  if (inPage) {
    return inPage;
  }
  const versions = lineageQuery.data?.results ?? [];
  return versions.length > 0 ? (groupLineages(versions)[0] ?? null) : null;
}

/**
 * Everything a conversation made, beside it.
 *
 * The open artifact sits in front of a small stack of the others: the newest,
 * the pinned and the ones opened lately. A click on it fans them out, and ⌘J
 * opens all of them to search. Under the stack is where the artifact came
 * from, then the artifact itself drawn for its kind, then a line to link,
 * pin, export or open it on its own page. A table read again later in the
 * conversation is one artifact with versions: the latest shows, the earlier
 * ones are a click away, and the cells that changed since the version before
 * are marked.
 */
export function DeskWorkspace({
  threadId,
  liveArtifacts,
  pendingLookups = NO_PENDING_LOOKUPS,
  onClose,
}: {
  threadId: string;
  liveArtifacts: LiveArtifacts;
  /** The running turn's lookups, left out until the reply says which it keeps. */
  pendingLookups?: ReadonlySet<string>;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const artifactsQuery = useQuery(queries.assistant.artifacts(threadId));
  const page = useMemo(
    () =>
      withoutPendingLookups(
        artifactsQuery.data?.results ?? [],
        artifactsQuery.data?.counts,
        pendingLookups,
      ),
    [artifactsQuery.data, pendingLookups],
  );
  const lineages = useMemo(() => groupLineages(page.results), [page.results]);
  const total = Math.max(page.counts?.all ?? 0, lineages.length);
  const remembered = useDeskStore((state) => state.activeArtifactByThread[threadId]);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const browsing = useDeskStore((state) => state.browsing);
  const setBrowsing = useDeskStore((state) => state.setBrowsing);
  const rememberedLineage = useRememberedLineage(
    threadId,
    lineages,
    remembered,
    artifactsQuery.isSuccess,
  );
  const active = rememberedLineage ?? lineages[0] ?? null;

  const [versionByLineage, setVersionByLineage] = useState<Record<string, number>>({});
  const [fanning, setFanning] = useState(false);
  const [recent, setRecent] = useState<string[]>(() => recentByThread.get(threadId) ?? []);
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

  // A revision the stack has not settled on yet is one that just landed; the
  // revision the workspace opened with is already settled, so reopening a
  // conversation mid-turn does not flag anything as new.
  const [settledRevision, setSettledRevision] = useState(liveRevision);
  const arrived = liveRevision !== 0 && liveRevision !== settledRevision;
  const newest = arrived ? (lineageContaining(lineages, newestLive)?.id ?? null) : null;
  useEffect(() => {
    if (!arrived) {
      return;
    }
    const timer = window.setTimeout(() => setSettledRevision(liveRevision), NEW_MS);
    return () => window.clearTimeout(timer);
  }, [arrived, liveRevision]);

  const paneRef = useRef<HTMLDivElement>(null);
  // Leaving the list of every artifact takes away whatever had focus in it;
  // focus lands on the open artifact's card rather than on the page.
  const browsedRef = useRef(browsing);
  useEffect(() => {
    const left = browsedRef.current && !browsing;
    browsedRef.current = browsing;
    if (!left) {
      return;
    }
    const active = document.activeElement;
    if (active === null || active === document.body || !active.isConnected) {
      paneRef.current?.querySelector<HTMLElement>(".dk-ax-card.dk-front")?.focus();
    }
  }, [browsing]);
  const [height, setHeight] = useState(700);
  const hasArtifacts = active !== null;
  useLayoutEffect(() => {
    const element = paneRef.current;
    if (!element) {
      return;
    }
    const observer = new ResizeObserver(() => setHeight(element.clientHeight));
    observer.observe(element);
    return () => observer.disconnect();
  }, [hasArtifacts]);

  const pinMutation = useApiMutation({
    mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) =>
      apiService.assistantService.pinArtifact(threadId, id, pinned),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queries.assistant.artifacts(threadId).queryKey }),
    resourceName: "Artifact",
  });

  // Opening one remembers it for the stack, with the one it replaces, for
  // as long as the tab is open.
  const activeId = active?.id ?? null;
  const open = useCallback(
    (id: string) => {
      setActiveArtifact(threadId, id);
      setRecent((current) => {
        const next = pushRecent(activeId ? pushRecent(current, activeId) : current, id);
        recentByThread.set(threadId, next);
        return next;
      });
    },
    [activeId, setActiveArtifact, threadId],
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
            <Button
              variant="quiet"
              size="sm"
              className="mt-3.5"
              onClick={() => void artifactsQuery.refetch()}
            >
              {t("Try again")}
            </Button>
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
  const go = (step: number) => {
    const next = stepLineage(
      lineages.some((lineage) => lineage.id === active.id) ? lineages : [active, ...lineages],
      active.id,
      step,
    );
    if (next) open(next);
  };
  const isDocument = artifact.kind === "document" || artifact.kind === "briefing";
  const versions =
    active.versions.length > 1 && !isDocument ? (
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
  const older = olderNote(artifact.createdAt, artifact.turn, new Date());
  const pinned = active.versions.some((version) => version.pinned);

  return (
    <div className={cn("dk-apx", fanning && "dk-fanning", browsing && "dk-browsing")} ref={paneRef}>
      {browsing ? (
        <DeskArtifactBrowser
          threadId={threadId}
          total={total}
          pendingLookups={pendingLookups}
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
              lineages={stackLineages(lineages, active, recent)}
              active={active}
              newest={newest}
              total={total}
              height={height}
              onPick={open}
              onFan={setFanning}
              onAll={() => setBrowsing(true)}
            />
            <div className="dk-ax-nav">
              <Button
                variant="quiet"
                size="icon-sm"
                className="text-dsk-subtle"
                title={t("Previous")}
                aria-label={t("Previous")}
                onClick={() => go(-1)}
              >
                <ArtIcon name="up" size={13} stroke={2.2} />
              </Button>
              <Button
                variant="quiet"
                size="icon-sm"
                className="text-dsk-subtle"
                title={t("Next")}
                aria-label={t("Next")}
                onClick={() => go(1)}
              >
                <ArtIcon name="down" size={13} stroke={2.2} />
              </Button>
              <Button
                variant="quiet"
                size="icon-sm"
                className="text-dsk-subtle"
                title={t("Close")}
                aria-label={t("Hide artifacts")}
                onClick={onClose}
              >
                <ArtIcon name="x" size={13} stroke={2.2} />
              </Button>
            </div>
          </div>
          <div className="dk-ax-body" key={isDocument ? active.id : artifact.id}>
            {older && (
              <div className="dk-ax-oldnote">
                {older.turn
                  ? older.day === "yesterday"
                    ? t("From yesterday · {0}", older.turn)
                    : t("From {0} · {1}", older.day, older.turn)
                  : older.day === "yesterday"
                    ? t("From yesterday")
                    : t("From {0}", older.day)}
              </div>
            )}
            <Provenance artifact={artifact} previous={previous} />
            <ArtifactBody
              artifact={artifact}
              previous={previous}
              versions={versions}
              lineage={active}
              onOpenArtifact={open}
            />
          </div>
          <div className="dk-ax-foot">
            <Button
              variant="bare"
              size="bare"
              className="h-7 min-w-0 gap-1.75 rounded-lg px-2.25 font-plex-mono text-xs text-dsk-subtle transition-colors duration-120 hover:bg-dsk-hover hover:text-dsk-fg [&_b]:font-medium [&_b]:text-dsk-fg2 [&_span]:truncate"
              onClick={copyLink}
              title={t("Copy link")}
            >
              <ArtIcon name="copy" size={12} />
              <span>
                desk/c/{threadId.slice(-4).toLowerCase()}/a/<b>{artifact.slug || artifact.id}</b>
              </span>
            </Button>
            <span className="flex-1" />
            <Button
              variant="quiet"
              size="icon-sm"
              className={cn(
                "text-dsk-subtle",
                pinned && "text-dsk-fg [&_svg_path]:fill-current",
              )}
              title={pinned ? t("Unpin") : t("Pin to conversation")}
              aria-label={t("Pin to conversation")}
              aria-pressed={pinned}
              onClick={() => pinMutation.mutate({ id: artifact.id, pinned: !pinned })}
            >
              <ArtIcon name="pin" size={14} />
            </Button>
            {tabular && (
              <Button
                variant="quiet"
                size="icon-sm"
                className="text-dsk-subtle"
                title={t("Export CSV")}
                aria-label={t("Export CSV")}
                onClick={() => downloadFromUrl(artifactCsvUrl(threadId, artifact.id))}
              >
                <ArtIcon name="dl" size={14} />
              </Button>
            )}
            <a
              className={cn(
                buttonVariants({ variant: "quiet", size: "icon-sm" }),
                "text-dsk-subtle",
              )}
              href={artifactPageLink(artifact)}
              target="_blank"
              rel="noreferrer"
              title={t("Open on its own page")}
              aria-label={t("Open on its own page")}
            >
              <ArtIcon name="ext" size={14} />
            </a>
          </div>
        </>
      )}
    </div>
  );
}
