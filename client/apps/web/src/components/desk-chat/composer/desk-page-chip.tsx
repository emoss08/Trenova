import { derivePageContext, stripAppTitle } from "@/components/assistant/page-context";
import { usePageContext } from "@/components/assistant/use-page-context";
import { useDeskStore } from "@/stores/desk-store";
import { useRecentPages } from "@/stores/recent-pages-store";
import type { AssistantPageContext } from "@/types/assistant";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useMemo, useRef, useState } from "react";
import { useLocation } from "react-router";
import { DeskIcon, type DeskIconName } from "../desk-icons";
import { useOutsideDismiss } from "../use-outside-dismiss";

/** The page a person came to the Desk from. */
export type DeskPage = { path: string; title: string };

const PAGE_ICONS: Array<[RegExp, DeskIconName]> = [
  [/^\/(shipments?|dispatch|loads?|orders?)/, "truck"],
  [/^\/(billing|invoices?|accounting|settlements?)/, "receipt"],
  [/^\/(customers?|crm)/, "headset"],
  [/^\/(reports?|insights?|analytics)/, "compass"],
  [/^\/(workers?|drivers?|fleet)/, "route"],
  [/^\/(carriers?|compliance)/, "shield"],
  [/^\/(documents?|capture|intake)/, "file"],
];

export function pageIcon(path: string): DeskIconName {
  return PAGE_ICONS.find(([pattern]) => pattern.test(path))?.[1] ?? "eye";
}

/** The Desk itself and the sign-in pages are never the page a question is about. */
function isAskablePage(path: string): boolean {
  return !path.startsWith("/desk") && !path.startsWith("/login") && path !== "/";
}

/**
 * The page questions are about, and whether they carry it.
 *
 * At the Desk it is the last page the person had open before coming in: the
 * Desk is a room of its own, so the page that matters is the one they walked
 * in from, the shipment list they were looking at when they thought of the
 * question. Asked from the assistant over a page (`onScreen`), it is the page
 * on screen, read with its filters and table at send time.
 */
export function useDeskPage({ onScreen = false }: { onScreen?: boolean } = {}) {
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const recent = useRecentPages(organizationId ?? undefined);
  const share = useDeskStore((state) => state.sharePage);
  const setShare = useDeskStore((state) => state.setSharePage);
  const { pathname } = useLocation();
  const readScreen = usePageContext();
  const page = useMemo<DeskPage | null>(() => {
    if (onScreen) {
      if (!isAskablePage(pathname)) {
        return null;
      }
      const recorded = recent.find((candidate) => candidate.path === pathname)?.title;
      const title =
        recorded || (typeof document === "undefined" ? "" : stripAppTitle(document.title));
      return { path: pathname, title: title || pathname };
    }
    return recent.find((candidate) => isAskablePage(candidate.path)) ?? null;
  }, [onScreen, pathname, recent]);

  const context = useCallback((): AssistantPageContext | null => {
    if (!share || !page) {
      return null;
    }
    if (onScreen) {
      return readScreen();
    }
    const derived = derivePageContext({ pathname: page.path, search: "", title: page.title });
    return derived ? { ...derived, title: page.title } : null;
  }, [onScreen, page, readScreen, share]);

  return { page, share: share && page !== null, setShare, context, onScreen };
}

/**
 * The page the Desk can see, as a chip in the composer, and a card saying
 * what is shared and a switch to stop sharing it. With a page shared, the
 * card offers to explain it. Asked from over a page (`onScreen`), the page is
 * the one on screen and the assistant, not the Desk, is what can see it.
 */
export function DeskPageChip({
  page,
  share,
  onShareChange,
  onExplain,
  onScreen = false,
}: {
  page: DeskPage | null;
  share: boolean;
  onShareChange: (share: boolean) => void;
  onExplain?: () => void;
  onScreen?: boolean;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLSpanElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutsideDismiss(rootRef, open, close);
  const shown = share ? page : null;

  return (
    <span className="dk-pc" ref={rootRef}>
      <Button
        variant="bare"
        size="bare"
        className={cn(
          "dk-pc-b relative size-7 justify-center rounded-full text-sm text-dsk-muted transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg",
          !shown && "dk-off pr-1.5",
          open && "dk-open bg-dsk-hover text-dsk-fg",
        )}
        title={
          shown
            ? onScreen
              ? t("The assistant can see {0}", shown.title)
              : t("Desk can see {0}", shown.title)
            : t("Not sharing a page")
        }
        aria-label={shown ? t("Sharing {0}", shown.title) : t("Not sharing a page")}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span className="dk-pc-ic">
          <DeskIcon name={shown ? pageIcon(shown.path) : "eye"} size={12} stroke={2} />
        </span>
        <span className="dk-pc-t">{shown ? shown.title : t("No page")}</span>
      </Button>
      {open && (
        <div className="dk-pc-pop dk-p1">
          <div className="dk-pc-h1">
            <span className="dk-pc-hic">
              <DeskIcon name={page ? pageIcon(page.path) : "eye"} size={16} stroke={2} />
            </span>
            <span>
              <b>{page ? page.title : onScreen ? t("Assistant") : t("Desk")}</b>
              <em>
                {page
                  ? onScreen
                    ? t("The page you're on")
                    : t("The page you came from")
                  : t("You're not on a page")}
              </em>
            </span>
          </div>
          {page && (
            <dl className="dk-pc-dl1">
              <div>
                <dt>{t("Page")}</dt>
                <dd>{page.path}</dd>
              </div>
            </dl>
          )}
          {page && (
            <div className="dk-pc-tg1">
              <span>
                <b>
                  {onScreen
                    ? t("Share this page with the assistant")
                    : t("Share this page with Desk")}
                </b>
                <em>
                  {share
                    ? t("Sent with each message so the agent knows what you're looking at")
                    : t("The agent won't see what you're looking at")}
                </em>
              </span>
              <Button
                variant="bare"
                size="bare"
                role="switch"
                aria-checked={share}
                aria-label={
                  onScreen
                    ? t("Share this page with the assistant")
                    : t("Share this page with Desk")
                }
                className={cn(
                  "dk-pc-sw relative h-4.5 w-8 flex-none rounded-full bg-dsk-b-strong transition-colors duration-200",
                  share && "bg-dsk-success",
                )}
                onClick={() => onShareChange(!share)}
              >
                <i
                  className={cn(
                    "absolute top-0.5 left-0.5 size-3.5 rounded-full bg-dsk-on-solid transition-transform duration-250 ease-(--dk-spring)",
                    share && "translate-x-3.5",
                  )}
                />
              </Button>
            </div>
          )}
          {share && page && onExplain && (
            <Button
              variant="bare"
              size="bare"
              className="flex w-full gap-2.25 rounded-lg px-2 py-2.25 text-left text-sm text-dsk-fg2 hover:bg-dsk-hover hover:text-dsk-fg"
              onClick={() => {
                onExplain();
                setOpen(false);
              }}
            >
              <DeskIcon name="info" size={13} stroke={2} />
              {t("Explain what's on this page")}
              <span className="dk-kbd ml-auto">/explain</span>
            </Button>
          )}
        </div>
      )}
    </span>
  );
}

/** Under a sent question: the page it was asked from. */
export function DeskPageSent({ context }: { context: AssistantPageContext | null | undefined }) {
  const t = useT();
  if (!context || !isAskablePage(context.path)) {
    return null;
  }
  const title = context.title || context.path;
  return (
    <div className="dk-pc-sent">
      <DeskIcon name={pageIcon(context.path)} size={11} stroke={2} />
      <span>
        {t("Asked from")} <b>{title}</b>
      </span>
    </div>
  );
}
