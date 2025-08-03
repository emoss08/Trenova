/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { LazyImage } from "@/components/ui/image";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useDebouncedCallback } from "@/hooks/use-debounce";
import { UserSchema } from "@/lib/schemas/user-schema";
import { cn } from "@/lib/utils";
import type { Editor } from "@tiptap/core";
import { FloatingMenu } from "@tiptap/react/menus";
import { useCallback, useEffect, useRef, useState } from "react";

interface MentionFloatingMenuProps {
  editor: Editor;
  searchUsers: (query: string) => Promise<UserSchema[]>;
  onMentionedUsersChange?: (userIds: string[]) => void;
}

export function MentionFloatingMenu({
  editor,
  searchUsers,
  onMentionedUsersChange,
}: MentionFloatingMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [users, setUsers] = useState<UserSchema[]>([]);
  const [loading, setLoading] = useState(false);
  const commandRef = useRef<HTMLDivElement>(null);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const itemRefs = useRef<(HTMLDivElement | null)[]>([]);

  const { setValue: debouncedSearch } = useDebouncedCallback(
    async (query: string) => {
      if (!query || query.length < 1) {
        setUsers([]);
        return;
      }

      setLoading(true);
      try {
        const results = await searchUsers(query);
        setUsers(results);
        setSelectedIndex(results.length > 0 ? 0 : -1);
      } catch (error) {
        console.error("Failed to search users:", error);
        setUsers([]);
      } finally {
        setLoading(false);
      }
    },
    300,
  );

  useEffect(() => {
    debouncedSearch(search);
  }, [search, debouncedSearch]);

  const insertMention = useCallback(
    (user: UserSchema) => {
      if (!editor || !user.id) return;

      try {
        const { from, $from } = editor.state.selection;
        const text = $from.parent.textContent;
        const beforeCursor = text.slice(0, $from.parentOffset);
        const mentionMatch = beforeCursor.match(/@(\w*)$/);

        if (!mentionMatch) return;

        const mentionStart = $from.start() + (mentionMatch.index || 0);
        const mentionEnd = from;

        // Check if there's already a space before the @ symbol
        const charBeforeMention =
          mentionMatch.index && mentionMatch.index > 0
            ? beforeCursor[mentionMatch.index - 1]
            : "";
        const needsSpaceBefore =
          charBeforeMention &&
          charBeforeMention !== " " &&
          charBeforeMention !== "\n";

        const content = [];

        // Add space before mention if needed
        if (needsSpaceBefore) {
          content.push({
            type: "text",
            text: " ",
          });
        }

        // Add the mention
        content.push({
          type: "mention",
          attrs: {
            id: user.id,
            label: user.username,
          },
        });

        // Add space after mention
        content.push({
          type: "text",
          text: " ",
        });

        const chain = editor
          .chain()
          .focus()
          .deleteRange({
            from: mentionStart,
            to: mentionEnd,
          })
          .insertContent(content);

        chain.run();

        // Force cursor to the end after a small delay
        setTimeout(() => {
          const { to } = editor.state.selection;
          editor.commands.setTextSelection(to);
        }, 0);

        // Track mentioned users
        if (onMentionedUsersChange) {
          const mentions: string[] = [];
          editor.state.doc.descendants((node) => {
            if (node.type.name === "mention" && node.attrs.id) {
              mentions.push(node.attrs.id);
            }
          });
          onMentionedUsersChange(mentions);
        }
      } catch (error) {
        console.error("Error inserting mention:", error);
      } finally {
        setIsOpen(false);
        setSearch("");
        setSelectedIndex(-1);
      }
    },
    [editor, onMentionedUsersChange],
  );

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isOpen || !editor) return;

      const preventDefault = () => {
        e.preventDefault();
        e.stopImmediatePropagation();
      };

      switch (e.key) {
        case "ArrowDown":
          preventDefault();
          setSelectedIndex((prev) => {
            if (prev === -1) return 0;
            return prev < users.length - 1 ? prev + 1 : 0;
          });
          break;

        case "ArrowUp":
          preventDefault();
          setSelectedIndex((prev) => {
            if (prev === -1) return users.length - 1;
            return prev > 0 ? prev - 1 : users.length - 1;
          });
          break;

        case "Enter": {
          preventDefault();
          // If we have users but nothing is selected, select the first one
          const targetIndex =
            selectedIndex === -1 && users.length > 0 ? 0 : selectedIndex;
          if (targetIndex >= 0 && users[targetIndex]) {
            insertMention(users[targetIndex]);
          }
          break;
        }

        case "Escape":
          preventDefault();
          setIsOpen(false);
          setSelectedIndex(-1);
          break;
      }
    },
    [isOpen, selectedIndex, users, insertMention, editor],
  );

  useEffect(() => {
    if (!editor?.options.element) return;

    const editorElement = editor.options.element;
    const handleEditorKeyDown = (e: Event) => handleKeyDown(e as KeyboardEvent);

    // Use capture phase to handle the event before TipTap
    editorElement.addEventListener("keydown", handleEditorKeyDown, true);
    return () =>
      editorElement.removeEventListener("keydown", handleEditorKeyDown, true);
  }, [handleKeyDown, editor]);

  useEffect(() => {
    // Reset selection when search changes
    if (users.length > 0) {
      setSelectedIndex(0);
    } else {
      setSelectedIndex(-1);
    }
  }, [users]);

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
        strategy: "fixed",
        placement: "top-start",
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

        const mentionMatch = currentLineText.match(/@(\w*)$/);
        const isMention =
          mentionMatch &&
          $from.parent.type.name !== "codeBlock" &&
          $from.parentOffset === currentLineText.length;

        if (!isMention) {
          if (isOpen) setIsOpen(false);
          return false;
        }

        const query = mentionMatch[1] || "";
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
            {loading ? (
              <CommandEmpty className="py-3 text-center text-sm text-muted-foreground">
                Searching...
              </CommandEmpty>
            ) : users.length === 0 ? (
              <CommandEmpty className="py-3 text-center text-sm text-muted-foreground">
                No users found
              </CommandEmpty>
            ) : (
              <CommandGroup>
                {users.map((user, index) => (
                  <CommandItem
                    role="option"
                    key={user.id}
                    value={user.username}
                    onSelect={() => insertMention(user)}
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
                    <LazyImage
                      src={
                        user.profilePicUrl ||
                        `https://avatar.vercel.sh/${user.username}.svg`
                      }
                      alt={user.name}
                      className="size-6 rounded-full"
                    />
                    <div className="flex flex-col">
                      <span className="text-sm font-medium">{user.name}</span>
                      <span className="text-xs text-muted-foreground">
                        @{user.username}
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
