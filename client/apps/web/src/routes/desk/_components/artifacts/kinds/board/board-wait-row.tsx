import { DESK_ACTION_ROW } from "@/components/desk-chat/conversation/desk-message-actions";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * What says a reply's board is still being read: the breathing dot and its
 * words, set in the row the reply's actions take once the board lands. The
 * row's box is the actions' own, so the actions replace it in the same frame
 * and the reply keeps its height.
 */
export function BoardWaitRow() {
  const t = useT();
  return (
    <div className={cn(DESK_ACTION_ROW, "flex items-center gap-2.5 text-dsk-base text-dsk-subtle")}>
      <i
        className="size-1.5 shrink-0 animate-[dk-breathe_1.1s_ease-in-out_infinite] rounded-full bg-dsk-subtle"
        aria-hidden="true"
      />
      {t("Reading what the answer rests on…")}
    </div>
  );
}
