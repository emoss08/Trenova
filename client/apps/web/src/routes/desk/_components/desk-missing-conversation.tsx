import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId } from "react";
import { Link } from "react-router";

const GHOST_LINE = "block h-1.5 rounded-full bg-dsk-sunken";

/**
 * A conversation address that leads nowhere: deleted, or someone else's.
 * Drawn in the Desk's own palette, centred where the transcript would be: an
 * empty outline of a conversation with its chat mark, what happened, and the
 * two ways on, back to the Desk or searching for the one meant.
 */
export function DeskMissingConversation({ onSearch }: { onSearch: () => void }) {
  const t = useT();
  const titleId = useId();

  return (
    <section
      aria-labelledby={titleId}
      className="cc-fade-in flex h-full min-h-0 flex-col items-center justify-center gap-6 px-4 pb-16 text-center"
    >
      <div aria-hidden className="relative w-60 pt-4">
        <div className="flex flex-col gap-3 rounded-xl border border-dashed border-dsk-b-strong bg-dsk-raised px-4 pt-6 pb-5 [mask-image:linear-gradient(to_bottom,black_40%,transparent)]">
          <span className="ml-auto block h-5 w-24 rounded-full bg-dsk-sunken" />
          <span className="flex flex-col gap-1.5">
            <span className={`${GHOST_LINE} w-full`} />
            <span className={`${GHOST_LINE} w-5/6`} />
            <span className={`${GHOST_LINE} w-2/3`} />
          </span>
          <span className="ml-auto block h-5 w-16 rounded-full bg-dsk-sunken" />
        </div>
        <span className="absolute top-0 left-1/2 grid size-8 -translate-x-1/2 place-items-center rounded-full border border-dsk-b bg-dsk-raised text-dsk-subtle">
          <DeskIcon name="chat" size={15} />
        </span>
      </div>
      <div className="flex max-w-xs flex-col items-center gap-1.5">
        <h2 id={titleId} className="text-base font-semibold text-dsk-fg">
          {t("This conversation is not here")}
        </h2>
        <p className="text-sm text-pretty text-dsk-subtle">
          {t("It may have been deleted, or it belongs to someone else.")}
        </p>
      </div>
      <div className="flex flex-wrap items-center justify-center gap-2">
        <Button size="sm" nativeButton={false} render={<Link to="/desk" />}>
          {t("Back to the desk")}
        </Button>
        <Button size="sm" variant="outline" onClick={onSearch}>
          <DeskIcon name="search" size={13} />
          {t("Search conversations")}
        </Button>
      </div>
    </section>
  );
}
