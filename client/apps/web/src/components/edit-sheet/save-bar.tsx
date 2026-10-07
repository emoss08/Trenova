import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import {
  AlertCircleIcon,
  AlertTriangleIcon,
  CheckIcon,
  ChevronDownIcon,
} from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import type { EditFlow } from "./use-edit-flow";

type SaveBarProps = {
  flow: EditFlow;
  /** The list of changes the "unsaved changes" control opens. */
  review?: ReactNode;
  saveLabel?: string;
  /** Anything the editor shows beside the state, such as a test button. */
  leading?: ReactNode;
};

/**
 * The foot of every editor: what is unsaved, why it cannot be saved, and the save. While
 * closing would lose changes it asks first, in place.
 */
export function SaveBar({ flow, review, saveLabel, leading }: SaveBarProps) {
  const t = useT();
  const count = flow.changed.length;

  if (flow.confirmingClose) {
    return (
      <div className="flex items-center gap-2" role="alertdialog" aria-live="polite">
        <AlertTriangleIcon className="size-3.5 text-warning" />
        <span className="font-medium">
          {count === 1
            ? t("Discard 1 unsaved change?")
            : t("Discard {0} unsaved changes?", count)}
        </span>
        <span className="flex-1" />
        <Button size="sm" variant="outline" autoFocus onClick={flow.keepEditing}>
          {t("Keep editing")}
        </Button>
        <Button size="sm" variant="destructive" onClick={flow.close}>
          {t("Discard and close")}
        </Button>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      {flow.dirty ? (
        <Popover open={flow.reviewing && Boolean(review)} onOpenChange={flow.setReviewing}>
          <PopoverTrigger
            render={
              <button
                type="button"
                className={cn(
                  "ui-focus-ring inline-flex items-center gap-1.5 rounded-control px-2 py-1 text-xs font-medium",
                  flow.reviewing ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted",
                )}
              />
            }
          >
            <span className="size-1.5 rounded-full bg-brand" />
            {count === 1 ? t("1 unsaved change") : t("{0} unsaved changes", count)}
            <ChevronDownIcon
              className={cn("size-3 transition-transform", flow.reviewing && "rotate-180")}
            />
          </PopoverTrigger>
          {review && (
            <PopoverContent side="top" align="start" className="w-[28rem] p-0">
              {review}
            </PopoverContent>
          )}
        </Popover>
      ) : flow.saved ? (
        <span className="inline-flex items-center gap-1 text-xs font-medium text-success">
          <CheckIcon className="size-3" />
          {t("Saved")}
        </span>
      ) : (
        <span className="text-xs text-muted-foreground">
          {flow.create ? t("Nothing is created until you save") : t("No changes")}
        </span>
      )}
      {flow.invalid && (
        <span className="inline-flex items-center gap-1 text-xs text-warning">
          <AlertCircleIcon className="size-3" />
          {flow.invalid}
        </span>
      )}
      {leading}
      <span className="flex-1" />
      {flow.dirty && !flow.create && (
        <Button size="sm" variant="ghost" onClick={flow.discard}>
          {t("Discard")}
        </Button>
      )}
      <Button
        size="sm"
        disabled={!flow.canSave}
        isLoading={flow.saving}
        loadingText={t("Saving")}
        onClick={flow.save}
      >
        {saveLabel ?? (flow.create ? t("Create") : t("Save changes"))}
        <KbdGroup className="ml-1 opacity-70">
          <Kbd>⌘</Kbd>
          <Kbd>S</Kbd>
        </KbdGroup>
      </Button>
    </div>
  );
}
