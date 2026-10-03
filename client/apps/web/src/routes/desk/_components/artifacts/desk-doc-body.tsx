import { useApiMutation } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { apiService } from "@/services/api";
import { artifactDocumentUrl } from "@/services/assistant";
import type { AssistantArtifact } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, downloadFromUrl } from "@trenova/shared/lib/utils";
import { Fragment, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useOutsideDismiss } from "../use-outside-dismiss";
import { documentFrom, type DocumentSource } from "./artifact-payloads";
import { ArtIcon } from "./desk-art-kinds";
import type { ArtifactLineage } from "./desk-lineage";
import {
  editedBlockMarkdown,
  inlineParts,
  joinDocBlocks,
  parseDocBlocks,
  plainText,
  readMinutes,
  replaceDocBlock,
  wordCount,
  type DocBlock,
} from "./doc-blocks";

type RewriteMode = "shorter" | "plain" | "ask";
type Selection = { blockId: string; x: number; y: number };
type Suggestion = { blockId: string; mode: RewriteMode; ask: string; text: string };

const TOAST_MS = 1600;

function shortTime(at: number): string {
  return new Date(at * 1000).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

/** A citation mark: the source's number, and what it is when pointed at. */
function Cite({
  n,
  source,
  onOpen,
}: {
  n: number;
  source: DocumentSource | undefined;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const timer = useRef<number | undefined>(undefined);
  const show = () => {
    window.clearTimeout(timer.current);
    setOpen(true);
  };
  const hide = () => {
    timer.current = window.setTimeout(() => setOpen(false), 120);
  };
  useEffect(() => () => window.clearTimeout(timer.current), []);

  return (
    <span
      className="dk-dx-cw"
      onMouseEnter={show}
      onMouseLeave={hide}
      contentEditable={false}
      data-cite={n}
    >
      <button
        type="button"
        className="dk-dx-c"
        onClick={() => source?.artifactId && onOpen(source.artifactId)}
        aria-label={source ? source.label : t("Source {0}", n)}
      >
        {n}
      </button>
      {open && source && (
        <span className="dk-dx-cp" onMouseEnter={show} onMouseLeave={hide}>
          {source.tool !== "" && <code>{source.tool}</code>}
          <b>{source.label}</b>
          {source.detail !== "" && <em>{source.detail}</em>}
          {source.artifactId !== "" && (
            <button type="button" onClick={() => onOpen(source.artifactId)}>
              <ArtIcon name="ext" size={11} />
              {t("Open artifact")}
            </button>
          )}
        </span>
      )}
    </span>
  );
}

/** A block's text with its emphasis, links and citation marks. */
function Rich({
  text,
  sources,
  onOpen,
}: {
  text: string;
  sources: Map<number, DocumentSource>;
  onOpen: (id: string) => void;
}) {
  return (
    <>
      {inlineParts(text).map((part, index) => {
        switch (part.kind) {
          case "cite":
            return <Cite key={index} n={part.n} source={sources.get(part.n)} onOpen={onOpen} />;
          case "bold":
            return <b key={index}>{part.text}</b>;
          case "italic":
            return <i key={index}>{part.text}</i>;
          case "code":
            return <code key={index}>{part.text}</code>;
          case "link":
            return (
              <a key={index} href={part.href} target="_blank" rel="noreferrer">
                {part.text}
              </a>
            );
          default:
            return <Fragment key={index}>{part.text}</Fragment>;
        }
      })}
    </>
  );
}

/** A menu under a toolbar button, closed by a click outside it. */
function DocMenu({
  label,
  icon,
  children,
}: {
  label: string;
  icon: ReactNode;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLSpanElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutsideDismiss(rootRef, open, close);

  return (
    <span className="dk-dx-m" ref={rootRef}>
      <button
        type="button"
        className={cn("dk-dx-tb", open && "dk-on")}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        {icon}
        {label}
        <ArtIcon name="down" size={10} stroke={2.4} />
      </button>
      {open && (
        // oxlint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions -- the menu's own buttons carry the keys; the wrapper only folds it after a choice
        <div className="dk-dx-mp dk-right" onClick={close}>
          {children}
        </div>
      )}
    </span>
  );
}

/**
 * A write-up the agent published, read as a page: a serif reading column with
 * its kind, length and author above it, citation marks that say what each
 * claim rests on, and the sources at the end. Every edit is a version: a
 * person can rewrite a passage they select, edit any paragraph in place, go
 * back to an earlier version and restore it, and take it away as a PDF, a
 * Word file or text.
 */
export function DeskDocBody({
  lineage,
  shown,
  onOpenArtifact,
}: {
  lineage: ArtifactLineage;
  /** The version the workspace was asked to show. */
  shown: AssistantArtifact;
  onOpenArtifact: (id: string) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const versions = lineage.versions;
  const latestIndex = versions.length - 1;
  const shownIndex = versions.findIndex((version) => version.id === shown.id);
  const [view, setView] = useState<number | null>(
    shownIndex >= 0 && shownIndex !== latestIndex ? shownIndex : null,
  );
  const index = view === null ? latestIndex : Math.min(view, latestIndex);
  const current = versions[index];
  const doc = useMemo(() => documentFrom(current), [current]);
  const blocks = useMemo(() => parseDocBlocks(doc.body), [doc.body]);
  const sources = useMemo(() => new Map(doc.sources.map((source) => [source.n, source])), [doc]);
  const old = index !== latestIndex;
  const threadId = current.threadId;

  const now = useNowSeconds(30_000);
  const [mode, setMode] = useState<"read" | "edit">("read");
  const [dirty, setDirty] = useState(false);
  const [editRevision, setEditRevision] = useState(0);
  const [selection, setSelection] = useState<Selection | null>(null);
  const [ask, setAsk] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [suggestion, setSuggestion] = useState<Suggestion | null>(null);
  const [versionsOpen, setVersionsOpen] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const paperRef = useRef<HTMLElement>(null);
  const versionsRef = useRef<HTMLSpanElement>(null);
  const closeVersions = useCallback(() => setVersionsOpen(false), []);
  useOutsideDismiss(versionsRef, versionsOpen, closeVersions);

  useEffect(() => {
    if (notice === null) return;
    const timer = window.setTimeout(() => setNotice(null), TOAST_MS);
    return () => window.clearTimeout(timer);
  }, [notice]);

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: ["assistant-artifacts", threadId] });

  const saveMutation = useApiMutation({
    mutationFn: ({ body, note }: { body: string; note: string }) =>
      apiService.assistantService.saveDocumentVersion(threadId, current.id, { body, note }),
    onSuccess: async (saved) => {
      await refresh();
      setView(null);
      setNotice(t("Saved as v{0}", saved.lineageSeq));
    },
    resourceName: "Document",
  });
  const restoreMutation = useApiMutation({
    mutationFn: (id: string) => apiService.assistantService.restoreDocumentVersion(threadId, id),
    onSuccess: async (saved) => {
      await refresh();
      setView(null);
      setNotice(t("Restored as v{0}", saved.lineageSeq));
    },
    resourceName: "Document",
  });
  const rewriteMutation = useApiMutation({
    mutationFn: (input: { block: DocBlock; mode: RewriteMode; ask: string }) =>
      apiService.assistantService.rewriteDocument(threadId, current.id, {
        text: input.block.raw,
        mode: input.mode,
        prompt: input.ask,
      }),
    onSuccess: (result, input) => {
      setBusy(null);
      setSuggestion({
        blockId: input.block.id,
        mode: input.mode,
        ask: input.ask,
        text: result.text,
      });
    },
    onError: () => setBusy(null),
    resourceName: "Document",
  });

  const words = wordCount(doc.body);
  const headings = blocks.filter((block) => block.type === "h");
  const editing = mode === "edit" && !old;

  const scrollTo = (id: string) => {
    const element = paperRef.current?.querySelector<HTMLElement>(`[data-blk="${id}"]`);
    if (!element) return;
    let scroller = element.parentElement;
    while (
      scroller &&
      scroller !== document.body &&
      !(
        scroller.scrollHeight > scroller.clientHeight &&
        /auto|scroll/u.test(getComputedStyle(scroller).overflowY)
      )
    ) {
      scroller = scroller.parentElement;
    }
    scroller?.scrollTo({
      top:
        scroller.scrollTop +
        element.getBoundingClientRect().top -
        scroller.getBoundingClientRect().top -
        12,
      behavior: "smooth",
    });
    element.classList.remove("dk-dx-hit");
    void element.offsetWidth;
    element.classList.add("dk-dx-hit");
  };

  const onMouseUp = () => {
    if (mode !== "read" || old || busy || suggestion) return;
    const picked = window.getSelection();
    if (!picked || picked.isCollapsed || !picked.toString().trim()) {
      setSelection(null);
      return;
    }
    const anchor = picked.anchorNode;
    const node = anchor && (anchor.nodeType === 1 ? (anchor as Element) : anchor.parentElement);
    const element = node?.closest<HTMLElement>("[data-blk]");
    const paper = paperRef.current;
    if (!element || !paper || !paper.contains(element)) return;
    const block = blocks.find((candidate) => candidate.id === element.dataset.blk);
    if (!block || (block.type !== "p" && block.type !== "sum")) {
      setSelection(null);
      return;
    }
    const range = picked.getRangeAt(0).getBoundingClientRect();
    const frame = paper.getBoundingClientRect();
    setSelection({
      blockId: block.id,
      x: Math.max(8, Math.min(frame.width - 250, range.left - frame.left + range.width / 2 - 125)),
      y: range.top - frame.top - 44,
    });
    setAsk("");
  };

  useEffect(() => {
    const onDown = (event: MouseEvent) => {
      if (event.target instanceof Element && event.target.closest(".dk-dx-sel")) return;
      window.setTimeout(() => {
        const picked = window.getSelection();
        if (!picked || picked.isCollapsed) setSelection(null);
      }, 0);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, []);

  const rewrite = (how: RewriteMode, prompt = "") => {
    const block = blocks.find((candidate) => candidate.id === selection?.blockId);
    if (!block) return;
    setSelection(null);
    window.getSelection()?.removeAllRanges();
    setBusy(block.id);
    rewriteMutation.mutate({ block, mode: how, ask: prompt });
  };

  const accept = () => {
    if (!suggestion) return;
    const note =
      suggestion.mode === "shorter"
        ? t("Shortened a paragraph")
        : suggestion.mode === "plain"
          ? t("Simplified a paragraph")
          : t("Rewrote: “{0}”", suggestion.ask || t("paragraph"));
    saveMutation.mutate({
      body: replaceDocBlock(doc.body, suggestion.blockId, suggestion.text),
      note,
    });
    setSuggestion(null);
  };

  const save = () => {
    const paper = paperRef.current;
    if (!paper) return;
    const edited = blocks.map((block) => {
      const element = paper.querySelector<HTMLElement>(`[data-blk="${block.id}"]`);
      return element ? { ...block, raw: editedBlockMarkdown(block, element) } : block;
    });
    saveMutation.mutate({ body: joinDocBlocks(edited), note: t("Your edits") });
    setDirty(false);
    setMode("read");
  };

  const discard = () => {
    setDirty(false);
    setEditRevision((value) => value + 1);
    setMode("read");
  };

  const switchMode = (next: "read" | "edit") => {
    if (next === mode) return;
    if (mode === "edit" && dirty) {
      discard();
      return;
    }
    setSelection(null);
    setSuggestion(null);
    setMode(next);
  };

  const copy = async (text: string, done: string) => {
    await navigator.clipboard?.writeText(text);
    setNotice(done);
  };

  const versionAuthor = (version: AssistantArtifact) => {
    const versionDoc = documentFrom(version);
    return versionDoc.editedBy === "person" ? t("You") : versionDoc.author || t("Agent");
  };

  const renderBlock = (block: DocBlock) => {
    const key = `${block.id}:${current.id}:${mode}:${editRevision}`;
    const editable =
      editing && block.type !== "table" && block.type !== "code" && block.type !== "rule";
    const common = {
      "data-blk": block.id,
      contentEditable: editable ? true : undefined,
      suppressContentEditableWarning: true,
      onInput: () => setDirty(true),
      className: cn("dk-dx-b", `dk-dx-${block.type}`, busy === block.id && "dk-busy"),
    };

    if (suggestion?.blockId === block.id) {
      const before = wordCount(block.raw);
      const after = wordCount(suggestion.text);
      return (
        <div key={key} data-blk={block.id} className="dk-dx-b dk-dx-sgw">
          <div className="dk-dx-sgh">
            <ArtIcon name="diff" size={12} />
            {suggestion.mode === "shorter"
              ? t("Shorter")
              : suggestion.mode === "plain"
                ? t("Plainer")
                : `“${suggestion.ask}”`}
            <span>{t("{0} → {1} words", before, after)}</span>
          </div>
          <p className="dk-dx-old">{plainText(block.raw)}</p>
          <p className="dk-dx-new">
            <Rich text={suggestion.text} sources={sources} onOpen={onOpenArtifact} />
          </p>
          <div className="dk-dx-sga">
            <button
              type="button"
              className="dk-ax-btn dk-ghost"
              onClick={() => setSuggestion(null)}
            >
              {t("Keep original")}
            </button>
            <button
              type="button"
              className="dk-ax-btn dk-ghost"
              onClick={() => {
                const again = suggestion;
                setSuggestion(null);
                setBusy(block.id);
                rewriteMutation.mutate({ block, mode: again.mode, ask: again.ask });
              }}
            >
              {t("Try again")}
            </button>
            <button type="button" className="dk-ax-btn dk-ink" onClick={accept}>
              <ArtIcon name="check" size={12} stroke={2.4} />
              {t("Accept")}
            </button>
          </div>
        </div>
      );
    }

    const rich = (text: string) => <Rich text={text} sources={sources} onOpen={onOpenArtifact} />;
    switch (block.type) {
      case "h":
        return (
          <h3 key={key} {...common}>
            {rich(block.text ?? "")}
          </h3>
        );
      case "sum":
      case "p":
        return (
          <p key={key} {...common}>
            {rich(block.text ?? "")}
          </p>
        );
      case "ol":
      case "ul": {
        const List = block.type;
        return (
          <List key={key} {...common}>
            {(block.items ?? []).map((item, at) => (
              <li key={at}>{rich(item)}</li>
            ))}
          </List>
        );
      }
      case "quote":
        return (
          <blockquote key={key} {...common}>
            {rich(block.text ?? "")}
          </blockquote>
        );
      case "table":
        return (
          <div key={key} {...common}>
            <table>
              <thead>
                <tr>
                  {(block.header ?? []).map((cell, at) => (
                    <th key={at}>{rich(cell)}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {(block.rows ?? []).map((row, at) => (
                  <tr key={at}>
                    {row.map((cell, column) => (
                      <td key={column}>{rich(cell)}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        );
      case "code":
        return (
          <pre key={key} {...common}>
            {block.raw.replace(/^```[^\n]*\n?/u, "").replace(/\n?```$/u, "")}
          </pre>
        );
      case "rule":
        return <hr key={key} className="dk-dx-rule" />;
    }
  };

  if (doc.body.trim() === "") {
    return <div className="dk-ax-pad dk-ax-oldnote">{t("This document is empty.")}</div>;
  }

  const byline = [
    current.lineageSeq > 1 && doc.editedBy === "person"
      ? doc.editor || t("You")
      : doc.author || t("Agent"),
    index === latestIndex && now - current.createdAt < 60
      ? t("edited just now")
      : t("written {0}", shortTime(current.createdAt)),
    doc.basis || t("from this conversation"),
  ].join(" · ");
  const initial = (doc.editedBy === "person" ? doc.editor || t("You") : doc.author || "A")
    .trim()
    .charAt(0)
    .toUpperCase();

  return (
    <div className={cn("dk-dx", editing && "dk-editing")}>
      <div className="dk-dx-bar">
        <div className="dk-dx-seg" role="group" aria-label={t("Mode")}>
          <button
            type="button"
            className={mode === "read" ? "dk-on" : undefined}
            onClick={() => switchMode("read")}
          >
            {t("Read")}
          </button>
          <button
            type="button"
            className={mode === "edit" ? "dk-on" : undefined}
            disabled={old}
            onClick={() => switchMode("edit")}
          >
            {t("Edit")}
          </button>
        </div>
        <span className="dk-axv" ref={versionsRef}>
          <button
            type="button"
            className={cn("dk-axv-b", versionsOpen && "dk-on", old && "dk-old")}
            title={t("Versions")}
            aria-expanded={versionsOpen}
            onClick={() => setVersionsOpen((value) => !value)}
          >
            v{current.lineageSeq}
            <ArtIcon name="down" size={9} stroke={2.6} />
          </button>
          {versionsOpen && (
            <div className="dk-axv-pop" role="listbox">
              {[...versions].reverse().map((version) => {
                const at = versions.indexOf(version);
                const versionDoc = documentFrom(version);
                return (
                  <button
                    key={version.id}
                    type="button"
                    role="option"
                    aria-selected={at === index}
                    className={cn("dk-axv-r", at === index && "dk-on")}
                    onClick={() => {
                      setView(at === latestIndex ? null : at);
                      setVersionsOpen(false);
                      setMode("read");
                      setSuggestion(null);
                    }}
                  >
                    <b>v{version.lineageSeq}</b>
                    <span>
                      <em>{versionDoc.versionNote || t("First draft")}</em>
                      <i>
                        {versionAuthor(version)} · {shortTime(version.createdAt)}
                        {at === latestIndex ? ` · ${t("Latest")}` : ""}
                      </i>
                    </span>
                    {at === index && <ArtIcon name="check" size={12} stroke={2.4} />}
                  </button>
                );
              })}
            </div>
          )}
        </span>
        <span className="flex-1" />
        <DocMenu label={t("Export")} icon={<ArtIcon name="dl" size={13} />}>
          <button
            type="button"
            onClick={() => {
              downloadFromUrl(artifactDocumentUrl(threadId, current.id, "pdf"));
              setNotice(t("Downloading PDF…"));
            }}
          >
            <span className="dk-dx-fx">PDF</span>
            {t("Download as PDF")}
          </button>
          <button
            type="button"
            onClick={() => {
              downloadFromUrl(artifactDocumentUrl(threadId, current.id, "docx"));
              setNotice(t("Downloading .docx…"));
            }}
          >
            <span className="dk-dx-fx">DOC</span>
            {t("Word document")}
          </button>
          <button
            type="button"
            onClick={() => void copy(`# ${current.title}\n\n${doc.body}`, t("Copied as Markdown"))}
          >
            <span className="dk-dx-fx">MD</span>
            {t("Copy as Markdown")}
          </button>
          <button
            type="button"
            onClick={() =>
              void copy(`${current.title}\n\n${plainText(doc.body)}`, t("Copied plain text"))
            }
          >
            <ArtIcon name="copy" size={13} />
            {t("Copy text")}
          </button>
        </DocMenu>
      </div>
      {headings.length > 1 && (
        <nav className="dk-dx-toc" aria-label={t("Contents")}>
          {headings.map((heading) => (
            <button key={heading.id} type="button" onClick={() => scrollTo(heading.id)}>
              {plainText(heading.text ?? "")}
            </button>
          ))}
        </nav>
      )}
      {old && (
        <div className="dk-dx-oldbar">
          <span>
            {t("Viewing")} <b>v{current.lineageSeq}</b> · {doc.versionNote || t("First draft")}
          </span>
          <button type="button" onClick={() => setView(null)}>
            {t("Back to latest")}
          </button>
          <button
            type="button"
            className="dk-ink"
            disabled={restoreMutation.isPending}
            onClick={() => restoreMutation.mutate(current.id)}
          >
            {t("Restore")}
          </button>
        </div>
      )}
      {/* oxlint-disable-next-line jsx-a11y/no-noninteractive-element-interactions -- selecting text is how a passage is chosen to rewrite; the keyboard path is the Edit mode */}
      <article className="dk-dx-paper" ref={paperRef} onMouseUp={onMouseUp}>
        <div className="dk-dx-kick">
          <span>{doc.docType || t("Document")}</span>
          <i />
          <span>
            {t(
              "{0, plural, one {# word} other {# words}} · {1} min read",
              words,
              readMinutes(words),
            )}
          </span>
        </div>
        <h2 className="dk-dx-h">{current.title}</h2>
        <div className="dk-dx-by">
          <span className="dk-dx-av">{initial}</span>
          <span>{byline}</span>
        </div>
        {blocks.map(renderBlock)}
        {doc.sources.length > 0 && (
          <section className="dk-dx-srcs">
            <h4>{t("Sources")}</h4>
            {doc.sources.map((source) =>
              source.artifactId !== "" ? (
                <button
                  key={source.n}
                  type="button"
                  className="dk-go"
                  onClick={() => onOpenArtifact(source.artifactId)}
                >
                  <span className="dk-dx-c">{source.n}</span>
                  <span>
                    <b>{source.label}</b>
                    <code>{source.tool}</code>
                  </span>
                  <ArtIcon name="ext" size={11} />
                </button>
              ) : (
                <div key={source.n}>
                  <span className="dk-dx-c">{source.n}</span>
                  <span>
                    <b>{source.label}</b>
                    <code>{source.tool}</code>
                  </span>
                </div>
              ),
            )}
          </section>
        )}
        {selection && (
          // oxlint-disable-next-line jsx-a11y/no-static-element-interactions -- keeps the text selected while a choice is clicked
          <div
            className="dk-dx-sel"
            style={{ left: selection.x, top: selection.y }}
            onMouseDown={(event) => {
              if (!(event.target instanceof HTMLInputElement)) event.preventDefault();
            }}
          >
            <button type="button" onClick={() => rewrite("shorter")}>
              {t("Shorter")}
            </button>
            <button type="button" onClick={() => rewrite("plain")}>
              {t("Plainer")}
            </button>
            <i />
            <input
              value={ask}
              onChange={(event) => setAsk(event.target.value)}
              placeholder={t("Ask to change…")}
              aria-label={t("Ask to change…")}
              onKeyDown={(event) => {
                event.stopPropagation();
                if (event.key === "Enter" && ask.trim()) rewrite("ask", ask.trim());
                if (event.key === "Escape") setSelection(null);
              }}
            />
          </div>
        )}
      </article>
      {editing && (
        <div className="dk-dx-save">
          <span className={dirty ? "dk-on" : undefined}>
            {dirty ? t("Unsaved changes") : t("Click any paragraph to edit")}
          </span>
          <span className="flex-1" />
          <button type="button" className="dk-ax-btn dk-ghost" onClick={discard}>
            {dirty ? t("Discard") : t("Done")}
          </button>
          {dirty && (
            <button
              type="button"
              className="dk-ax-btn dk-ink"
              disabled={saveMutation.isPending}
              onClick={save}
            >
              {t("Save as v{0}", versions[latestIndex].lineageSeq + 1)}
            </button>
          )}
        </div>
      )}
      {notice && (
        <div className="dk-dx-toast" role="status">
          <ArtIcon name="check" size={12} stroke={2.4} />
          {notice}
        </div>
      )}
    </div>
  );
}
