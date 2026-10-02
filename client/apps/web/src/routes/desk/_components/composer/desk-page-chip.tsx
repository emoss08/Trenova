import { derivePageContext } from "@/components/assistant/page-context";
import { useDeskStore } from "@/stores/desk-store";
import { useRecentPages } from "@/stores/recent-pages-store";
import type { AssistantPageContext } from "@/types/assistant";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useMemo, useRef, useState } from "react";
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
 * The last page the person had open before the Desk, and whether questions
 * carry it. The Desk is a room of its own, so the page that matters is the
 * one they walked in from: the shipment list they were looking at when they
 * thought of the question.
 */
export function useDeskPage() {
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const recent = useRecentPages(organizationId ?? undefined);
  const share = useDeskStore((state) => state.sharePage);
  const setShare = useDeskStore((state) => state.setSharePage);
  const page = useMemo<DeskPage | null>(
    () => recent.find((candidate) => isAskablePage(candidate.path)) ?? null,
    [recent],
  );

  const context = useCallback((): AssistantPageContext | null => {
    if (!share || !page) {
      return null;
    }
    const derived = derivePageContext({ pathname: page.path, search: "", title: page.title });
    return derived ? { ...derived, title: page.title } : null;
  }, [page, share]);

  return { page, share: share && page !== null, setShare, context };
}

/**
 * The page the Desk can see, as a chip in the composer, and a card saying
 * what is shared and a switch to stop sharing it. With a page shared, the
 * card offers to explain it.
 */
export function DeskPageChip({
  page,
  share,
  onShareChange,
  onExplain,
}: {
  page: DeskPage | null;
  share: boolean;
  onShareChange: (share: boolean) => void;
  onExplain?: () => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLSpanElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutsideDismiss(rootRef, open, close);
  const shown = share ? page : null;

  return (
    <span className="dk-pc" ref={rootRef}>
      <button
        type="button"
        className={cn("dk-pc-b", !shown && "dk-off", open && "dk-open")}
        title={shown ? t("Desk can see {0}", shown.title) : t("Not sharing a page")}
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span className="dk-pc-ic">
          <DeskIcon name={shown ? pageIcon(shown.path) : "eye"} size={12} stroke={2} />
        </span>
        <span className="dk-pc-t">{shown ? shown.title : t("No page")}</span>
      </button>
      {open && (
        <div className="dk-pc-pop dk-p1">
          <div className="dk-pc-h1">
            <span className="dk-pc-hic">
              <DeskIcon name={page ? pageIcon(page.path) : "eye"} size={16} stroke={2} />
            </span>
            <span>
              <b>{page ? page.title : t("Desk")}</b>
              <em>{page ? t("The page you came from") : t("You're not on a page")}</em>
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
                <b>{t("Share this page with Desk")}</b>
                <em>
                  {share
                    ? t("Sent with each message so the agent knows what you're looking at")
                    : t("The agent won't see what you're looking at")}
                </em>
              </span>
              <button
                type="button"
                role="switch"
                aria-checked={share}
                aria-label={t("Share this page with Desk")}
                className={cn("dk-pc-sw", share && "dk-on")}
                onClick={() => onShareChange(!share)}
              >
                <i />
              </button>
            </div>
          )}
          {share && page && onExplain && (
            <button
              type="button"
              className="dk-pc-ex"
              onClick={() => {
                onExplain();
                setOpen(false);
              }}
            >
              <DeskIcon name="info" size={13} stroke={2} />
              {t("Explain what's on this page")}
              <span className="dk-kbd">/explain</span>
            </button>
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
