"use no memo";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { faTriangleExclamation } from "@fortawesome/pro-solid-svg-icons";
import type { Editor } from "@tiptap/core";
import { FloatingMenu } from "@tiptap/react/menus";
import { useCallback, useEffect, useRef, useState } from "react";
import { COMMENT_TYPES, type CommentType } from "../utils";

interface SlashCommandMenuProps {
  editor: Editor;
  onCommentTypeChange?: (type: CommentType | null) => void;
}

export function SlashCommandMenu({
  editor,
  onCommentTypeChange,
}: SlashCommandMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState("");
  const commandRef = useRef<HTMLDivElement>(null);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const itemRefs = useRef<(HTMLDivElement | null)[]>([]);
  const [hasExistingCommentType, setHasExistingCommentType] = useState(false);

  const filteredTypes = COMMENT_TYPES.filter((type) =>
    type.label.toLowerCase().includes(search.toLowerCase()),
  );

  const insertCommentType = useCallback(
    (type: (typeof COMMENT_TYPES)[number]) => {
      if (!editor) return;

      try {
        // This shouldn't be called when there's already a comment type
        // but adding as safety check
        if (hasExistingCommentType) {
          setIsOpen(false);
          setSearch("");
          return;
        }

        const { from, $from } = editor.state.selection;
        const text = $from.parent.textContent;
        const beforeCursor = text.slice(0, $from.parentOffset);
        const slashMatch = beforeCursor.match(/\/(\w*)$/);

        if (!slashMatch) return;

        const slashStart = $from.start() + (slashMatch.index || 0);
        const slashEnd = from;

        editor
          .chain()
          .focus()
          .deleteRange({
            from: slashStart,
            to: slashEnd,
          })
          .insertCommentType(type.value)
          .run();

        onCommentTypeChange?.(type.value);
      } catch (error) {
        console.error("Error inserting comment type:", error);
      } finally {
        setIsOpen(false);
        setSearch("");
        setSelectedIndex(0);
      }
    },
    [editor, onCommentTypeChange, hasExistingCommentType],
  );

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isOpen || !editor) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          e.stopPropagation();
          setSelectedIndex((prev) => {
            const newIndex = prev < filteredTypes.length - 1 ? prev + 1 : 0;
            return newIndex;
          });
          return true;

        case "ArrowUp":
          e.preventDefault();
          e.stopPropagation();
          setSelectedIndex((prev) => {
            const newIndex = prev > 0 ? prev - 1 : filteredTypes.length - 1;
            return newIndex;
          });
          return true;

        case "Enter":
          e.preventDefault();
          e.stopPropagation();
          if (filteredTypes[selectedIndex]) {
            insertCommentType(filteredTypes[selectedIndex]);
          }
          return true;

        case "Escape":
          e.preventDefault();
          e.stopPropagation();
          setIsOpen(false);
          setSelectedIndex(0);
          return true;
      }

      return false;
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      isOpen,
      selectedIndex,
      filteredTypes,
      insertCommentType,
      editor,
      hasExistingCommentType,
    ],
  );

  useEffect(() => {
    if (!isOpen || !editor?.options.element) return;

    const editorElement = editor.options.element;
    const handleEditorKeyDown = (e: Event) => {
      const keyEvent = e as KeyboardEvent;
      if (["ArrowDown", "ArrowUp", "Enter", "Escape"].includes(keyEvent.key)) {
        handleKeyDown(keyEvent);
      }
    };

    if (editorElement instanceof HTMLElement) {
      editorElement.addEventListener("keydown", handleEditorKeyDown, true);
      return () =>
        editorElement.removeEventListener("keydown", handleEditorKeyDown, true);
    }
  }, [handleKeyDown, editor, isOpen]);

  useEffect(() => {
    if (filteredTypes.length > 0) {
      setSelectedIndex(0);
    } else {
      setSelectedIndex(-1);
    }
  }, [search, filteredTypes.length]);

  useEffect(() => {
    if (selectedIndex >= 0 && itemRefs.current[selectedIndex]) {
      itemRefs.current[selectedIndex]?.scrollIntoView({
        block: "nearest",
      });
    }
  }, [selectedIndex]);

  return (
    <FloatingMenu
      editor={editor}
      className="z-50"
      options={{
        strategy: "absolute",
        placement: "bottom-start",
        offset: 10,
        flip: true,
        shift: true,
        arrow: false,
      }}
      shouldShow={({ state }) => {
        if (!editor) return false;

        const { $from } = state.selection;
        const currentLineText = $from.parent.textBetween(
          0,
          $from.parentOffset,
          "\n",
          " ",
        );

        const slashMatch = currentLineText.match(/(^|\s)\/(\w*)$/);
        const isSlashCommand =
          slashMatch &&
          $from.parent.type.name !== "codeBlock" &&
          $from.parentOffset === currentLineText.length;

        if (!isSlashCommand) {
          if (isOpen) {
            setIsOpen(false);
            setHasExistingCommentType(false);
            onCommentTypeChange?.(null);
          }
          return false;
        }

        // Check if a comment type already exists
        let hasCommentType = false;
        state.doc.descendants((node) => {
          if (node.type.name === "commentType") {
            hasCommentType = true;
            return false; // Stop iteration
          }
        });

        setHasExistingCommentType(hasCommentType);

        const query = slashMatch[2] || "";
        if (query !== search) setSearch(query);
        if (!isOpen) setIsOpen(true);
        return true;
      }}
    >
      <Command
        role="listbox"
        ref={commandRef}
        className="z-50 w-72 overflow-hidden rounded-lg border bg-popover shadow-lg"
      >
        <ScrollArea className="max-h-[200px]">
          <CommandList>
            {hasExistingCommentType ? (
              <CommandEmpty className="p-1 text-xs">
                <div className="flex items-center gap-2 text-orange-600 dark:text-orange-400">
                  <Icon icon={faTriangleExclamation} className="size-3" />
                  <span>Only one comment type is allowed per comment</span>
                </div>
              </CommandEmpty>
            ) : filteredTypes.length === 0 ? (
              <CommandEmpty className="py-3 text-center text-sm text-muted-foreground">
                No comment types found
              </CommandEmpty>
            ) : (
              <CommandGroup heading="Comment Types">
                {filteredTypes.map((type, index) => (
                  <CommandItem
                    role="option"
                    key={type.value}
                    value={type.label}
                    onSelect={() => insertCommentType(type)}
                    onMouseEnter={() => setSelectedIndex(index)}
                    className={cn(
                      "flex items-center gap-2 p-2 cursor-pointer",
                      index === selectedIndex && "bg-accent",
                    )}
                    aria-selected={index === selectedIndex}
                    ref={(el) => {
                      itemRefs.current[index] = el;
                    }}
                    tabIndex={index === selectedIndex ? 0 : -1}
                  >
                    <Icon
                      icon={type.icon}
                      className={cn("size-3", type.iconClassName)}
                    />
                    <div className="flex flex-col">
                      <span className="text-sm font-medium">{type.label}</span>
                      <span className="text-xs text-muted-foreground">
                        {type.description}
                      </span>
                    </div>
                    <kbd className="ml-auto flex h-5 items-center rounded bg-muted px-1.5 text-xs text-muted-foreground">
                      ↵
                    </kbd>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>
        </ScrollArea>
      </Command>
    </FloatingMenu>
  );
}
