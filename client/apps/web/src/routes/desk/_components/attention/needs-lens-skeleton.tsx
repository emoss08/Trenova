import { useT } from "@trenova/shared/i18n/use-t";
import { NEEDS_DETAIL_TRANSITION_CLASS, NEEDS_LIST_TRANSITION_CLASS } from "../desk-motion";
import { DeskListSkeleton } from "../list/desk-list-skeleton";
import { DeskViewDetail, DeskViewList } from "../list/desk-views";
import type { NeedsLens } from "./lens";

/**
 * A lens of Needs you while it is read: its rows under the page's head, and
 * nothing beside them yet. The head is the page's and is already drawn, so
 * only what the lens brings waits.
 */
export function NeedsLensSkeleton({ lens }: { lens: NeedsLens }) {
  const t = useT();

  return (
    <>
      <DeskViewList className={NEEDS_LIST_TRANSITION_CLASS[lens]}>
        <DeskListSkeleton label={t("Reading what needs you")} />
      </DeskViewList>
      <DeskViewDetail reading={false} className={NEEDS_DETAIL_TRANSITION_CLASS[lens]} />
    </>
  );
}
