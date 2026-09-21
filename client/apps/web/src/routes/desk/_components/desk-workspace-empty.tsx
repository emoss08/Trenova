import { useT } from "@trenova/shared/i18n/use-t";
import { LayoutGridIcon } from "lucide-react";

/**
 * The workspace before anything has landed in it.
 *
 * It says what the half of the window is for rather than apologising for
 * being empty, because on the first visit this is most of what a person
 * sees and "nothing here" teaches them nothing. The three lines are the
 * three things that actually arrive here, in the order they usually do.
 */
export function DeskWorkspaceEmpty() {
  const t = useT();

  return (
    <div className="flex min-h-0 flex-1 items-center justify-center p-10">
      <div className="max-w-sm">
        <LayoutGridIcon className="text-muted-foreground/60 mb-4 size-5" />
        <h2 className="text-base font-semibold">{t("The work lands here")}</h2>
        <p className="text-muted-foreground mt-1.5 text-sm">
          {t(
            "Ask on the left and what comes back opens on this side, whole: a table you can sort and download, a message you can edit before it sends, a plan whose steps tick off as they run.",
          )}
        </p>
      </div>
    </div>
  );
}
