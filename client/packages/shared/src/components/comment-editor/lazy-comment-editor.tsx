import { useT } from "@trenova/shared/i18n/use-t";
import {
  BoldIcon,
  ItalicIcon,
  LinkIcon,
  ListIcon,
  ListOrderedIcon,
  UnderlineIcon,
} from "lucide-react";
import { lazy, Suspense, useCallback, useState } from "react";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import type { CommentEditorHandle, CommentEditorProps } from "./comment-editor";

const CommentEditor = lazy(() =>
  import("./comment-editor").then((module) => ({ default: module.CommentEditor })),
);

let preloadPromise: Promise<unknown> | null = null;

/**
 * Pulls the TipTap/ProseMirror chunk in ahead of first use. Idempotent — repeat
 * calls reuse the in-flight import.
 */
export function preloadCommentEditor(): Promise<unknown> {
  preloadPromise ??= import("./comment-editor");
  return preloadPromise;
}

const TOOLBAR_ICONS = [BoldIcon, ItalicIcon, UnderlineIcon, ListIcon, ListOrderedIcon, LinkIcon];

/**
 * Non-interactive stand-in for {@link CommentEditor}. It reproduces the real
 * editor's frame — bordered box, the same min-height content area, the same
 * six formatting buttons and the caller's own toolbar on the right — so the
 * composer does not move when the editor chunk lands. Clicking or focusing it
 * hands over to the real editor.
 */
function CommentEditorPlaceholder({
  compact,
  disabled,
  placeholder,
  toolbar,
  onActivate,
}: Pick<CommentEditorProps, "compact" | "disabled" | "placeholder" | "toolbar"> & {
  onActivate?: () => void;
}) {
  const t = useT();

  return (
    <div
      className={cn(
        "relative rounded-md border border-border bg-background transition-[border-color,box-shadow] duration-200 ease-in-out",
        disabled && "opacity-60",
      )}
    >
      <div
        role="textbox"
        aria-multiline="true"
        aria-label={t("Comment editor")}
        aria-busy={onActivate == null}
        tabIndex={disabled ? -1 : 0}
        className={cn(
          "w-full cursor-text px-3 py-2 text-sm wrap-break-word text-muted-foreground focus:outline-none",
          compact ? "min-h-[44px]" : "min-h-[100px]",
        )}
        onFocus={onActivate}
        onPointerDown={onActivate}
      >
        {placeholder}
      </div>
      <div className="flex items-center gap-0.5 rounded-b-md border-t border-border bg-muted px-2 py-1">
        {TOOLBAR_ICONS.map((Icon, index) => (
          <Button
            key={index}
            type="button"
            variant="ghost"
            size="xs"
            disabled
            tabIndex={-1}
            aria-hidden
            className="size-6 text-muted-foreground/50"
          >
            <Icon className="size-3.5" />
          </Button>
        ))}
        {toolbar && <div className="ml-auto flex items-center gap-1">{toolbar}</div>}
      </div>
    </div>
  );
}

/**
 * Defers TipTap until someone actually intends to type. Comment threads are
 * read far more often than they are written, and the editor is by far the
 * heaviest thing on the panel, so it is fetched on first focus, pointer-down or
 * autoFocus rather than with the thread itself.
 */
export function LazyCommentEditor({
  ref,
  ...props
}: CommentEditorProps & { ref?: React.Ref<CommentEditorHandle> }) {
  const [activatedByUser, setActivatedByUser] = useState(false);
  const active = activatedByUser || props.autoFocus === true;

  const activate = useCallback(() => {
    void preloadCommentEditor();
    setActivatedByUser(true);
  }, []);

  if (!active) {
    return (
      <CommentEditorPlaceholder
        compact={props.compact}
        disabled={props.disabled}
        placeholder={props.placeholder}
        toolbar={props.toolbar}
        onActivate={activate}
      />
    );
  }

  return (
    <Suspense
      fallback={
        <CommentEditorPlaceholder
          compact={props.compact}
          disabled={props.disabled}
          placeholder={props.placeholder}
          toolbar={props.toolbar}
        />
      }
    >
      <CommentEditor {...props} autoFocus={props.autoFocus || activatedByUser} ref={ref} />
    </Suspense>
  );
}
