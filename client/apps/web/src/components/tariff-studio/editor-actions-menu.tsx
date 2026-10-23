import { promoteToInput } from "@/components/formula-editor/tariff-actions";
import { goToDefinitionAtCursor, startRename } from "@/components/formula-editor/tariff-navigate";
import { findReferences } from "@/components/formula-editor/tariff-references";
import type { EditorView } from "@codemirror/view";
import { DotsHorizontalIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShiftShortcut, formatShortcut, isMacPlatform } from "@trenova/shared/lib/shortcuts";
import type { TariffFileKind } from "@trenova/shared/types/tariff";

type EditorCommand = (view: EditorView) => boolean;

/**
 * The editor's menu: the language commands that act at the cursor, with their keys, for a person
 * who reaches for a menu rather than a shortcut. Each runs on the editor with the cursor where the
 * person left it, and the editor keeps focus afterwards (or rename's field takes it).
 */
export function EditorActionsMenu({
  view,
  kind,
  readOnly = false,
}: {
  /** The editor the commands act on; null before it has mounted. */
  view: () => EditorView | null;
  /** A library has no inputs, so nothing in it promotes to one. */
  kind: TariffFileKind;
  readOnly?: boolean;
}) {
  const t = useT();
  const mac = isMacPlatform();
  const run = (command: EditorCommand) => {
    const editor = view();
    if (!editor) return;
    editor.focus();
    command(editor);
  };
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={t("Editor actions")}
            className="text-foreground-muted"
          />
        }
      >
        <DotsHorizontalIcon className="size-3.5" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top" className="min-w-56" finalFocus={false}>
        <DropdownMenuGroup>
          <DropdownMenuItem
            title={t("Go to definition")}
            endContent={<DropdownMenuShortcut>F12</DropdownMenuShortcut>}
            onClick={() => run(goToDefinitionAtCursor)}
          />
          <DropdownMenuItem
            title={t("Find references")}
            endContent={
              <DropdownMenuShortcut>{formatShiftShortcut("F12", mac)}</DropdownMenuShortcut>
            }
            onClick={() => run(findReferences)}
          />
          {!readOnly && (
            <>
              <DropdownMenuItem
                title={t("Rename")}
                endContent={<DropdownMenuShortcut>F2</DropdownMenuShortcut>}
                onClick={() => run((editor) => startRename(editor))}
              />
              {kind !== "library" && (
                <DropdownMenuItem
                  title={t("Promote to input")}
                  endContent={
                    <DropdownMenuShortcut>{formatShortcut(".", mac)}</DropdownMenuShortcut>
                  }
                  onClick={() => run(promoteToInput)}
                />
              )}
            </>
          )}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
