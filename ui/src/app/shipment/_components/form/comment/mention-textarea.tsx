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
import {
  Popover,
  PopoverAnchor,
  PopoverContent,
} from "@/components/ui/popover";
import { useDebouncedCallback } from "@/hooks/use-debounce";
import { UserSchema } from "@/lib/schemas/user-schema";
import { cn } from "@/lib/utils";
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";

interface MentionTextareaProps {
  value?: string;
  onChange?: (value: string) => void;
  searchUsers: (query: string) => Promise<UserSchema[]>;
  isInvalid?: boolean;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  rows?: number;
}

export interface MentionTextareaRef {
  focus: () => void;
  getValue: () => string;
}

/**
 * MentionTextarea component that displays mentions with blue background highlights inline.
 * Uses contenteditable div to render rich text with styled mentions.
 */
export const MentionTextarea = forwardRef<
  MentionTextareaRef,
  MentionTextareaProps
>(
  (
    {
      value = "",
      onChange,
      searchUsers,
      placeholder = "Add a comment... Use @ to mention users",
      disabled = false,
      isInvalid,
      className,
      rows = 3,
    },
    ref,
  ) => {
    const [mentionedUserIds, setMentionedUserIds] = useState<string[]>([]);
    const [showMentions, setShowMentions] = useState(false);
    const [mentionQuery, setMentionQuery] = useState("");
    const [users, setUsers] = useState<UserSchema[]>([]);
    const [isSearching, setIsSearching] = useState(false);
    const [selectedIndex, setSelectedIndex] = useState(0);
    const editorRef = useRef<HTMLDivElement>(null);
    const [isEmpty, setIsEmpty] = useState(true);

    // Expose ref to parent
    useImperativeHandle(ref, () => ({
      focus: () => editorRef.current?.focus(),
      getValue: () => getPlainTextValue(),
    }));

    // Sync editor content when value changes from parent (e.g., form reset)
    useEffect(() => {
      if (editorRef.current) {
        const currentText = getPlainTextValue();
        // Only update if the value actually changed from parent
        if (value !== currentText) {
          editorRef.current.innerHTML = renderContentWithMentions(value);
          setIsEmpty(value.length === 0);
        }
      }
    }, [value]);

    // Parse text and render mentions with blue background
    const renderContentWithMentions = (text: string): string => {
      if (!text) return "";

      // Escape HTML to prevent injection
      const escapeHtml = (str: string) => {
        const div = document.createElement("div");
        div.textContent = str;
        return div.innerHTML;
      };

      // Replace mentions (including partial ones) with styled spans
      // Matches @ followed by zero or more word characters
      const mentionRegex = /@(\w*)/g;
      const html = escapeHtml(text).replace(mentionRegex, (match, username) => {
        // Only style if there's an @ symbol (match will include @)
        if (match === "@") {
          return `<span class="inline bg-blue-100 text-blue-700 px-1 py-0.5 rounded-sm font-medium" data-mention="">@</span>`;
        }
        return `<span class="inline bg-blue-100 text-blue-700 px-1 py-0.5 rounded-sm font-medium" data-mention="${username}">${match}</span>`;
      });

      return html;
    };

    // Get plain text from contenteditable
    const getPlainTextValue = (): string => {
      if (!editorRef.current) return "";
      return editorRef.current.textContent || "";
    };

    // Search users debounced
    const { setValue: searchUsersDebounced } = useDebouncedCallback(
      async (query: string) => {
        if (!query || query.length < 1) {
          setUsers([]);
          return;
        }

        setIsSearching(true);
        try {
          const results = await searchUsers(query);
          setUsers(results);
          setSelectedIndex(0);
        } catch (error) {
          console.error("Failed to search users:", error);
          setUsers([]);
        } finally {
          setIsSearching(false);
        }
      },
      300,
    );

    useEffect(() => {
      if (mentionQuery) {
        searchUsersDebounced(mentionQuery);
      } else {
        setUsers([]);
      }
    }, [mentionQuery, searchUsersDebounced]);

    // Save cursor position before updating HTML
    const saveCursorPosition = () => {
      const selection = window.getSelection();
      if (!selection || selection.rangeCount === 0 || !editorRef.current)
        return null;

      const range = selection.getRangeAt(0);
      const preCaretRange = range.cloneRange();
      preCaretRange.selectNodeContents(editorRef.current);
      preCaretRange.setEnd(range.endContainer, range.endOffset);

      return {
        offset: preCaretRange.toString().length,
        atEnd:
          range.collapsed &&
          range.endOffset === range.endContainer.textContent?.length,
      };
    };

    // Restore cursor position after updating HTML
    const restoreCursorPosition = (
      savedPosition: { offset: number; atEnd: boolean } | null,
    ) => {
      if (!savedPosition || !editorRef.current) return;

      const selection = window.getSelection();
      if (!selection) return;

      const range = document.createRange();
      let charCount = 0;
      const nodeStack: Node[] = [editorRef.current];
      let foundNode: Node | null = null;
      let foundOffset = 0;

      // Simple approach: walk through text nodes until we reach the desired position
      while (nodeStack.length > 0 && !foundNode) {
        const node = nodeStack.pop()!;

        if (node.nodeType === Node.TEXT_NODE) {
          const nodeLength = node.textContent?.length || 0;
          if (charCount + nodeLength >= savedPosition.offset) {
            foundNode = node;
            foundOffset = savedPosition.offset - charCount;
            break;
          }
          charCount += nodeLength;
        } else {
          // Add child nodes in reverse order to process them in correct order
          for (let i = node.childNodes.length - 1; i >= 0; i--) {
            nodeStack.push(node.childNodes[i]);
          }
        }
      }

      if (foundNode) {
        try {
          range.setStart(
            foundNode,
            Math.min(foundOffset, foundNode.textContent?.length || 0),
          );
          range.collapse(true);
          selection.removeAllRanges();
          selection.addRange(range);
        } catch {
          // Fallback: set cursor at the end
          const textNodes = getAllTextNodes(editorRef.current);
          if (textNodes.length > 0) {
            const lastTextNode = textNodes[textNodes.length - 1];
            range.setStart(lastTextNode, lastTextNode.textContent?.length || 0);
            range.collapse(true);
            selection.removeAllRanges();
            selection.addRange(range);
          }
        }
      }
    };

    // Helper to get all text nodes
    const getAllTextNodes = (element: Node): Text[] => {
      const textNodes: Text[] = [];
      const walker = document.createTreeWalker(
        element,
        NodeFilter.SHOW_TEXT,
        null,
      );

      let node;
      while ((node = walker.nextNode())) {
        textNodes.push(node as Text);
      }
      return textNodes;
    };

    const handleInput = () => {
      const plainText = getPlainTextValue();

      // Save cursor position before any DOM manipulation
      const savedPosition = saveCursorPosition();
      const cursorOffset = savedPosition?.offset || 0;

      onChange?.(plainText);
      setIsEmpty(plainText.length === 0);

      // Update HTML with mentions styled
      if (editorRef.current) {
        const currentText = editorRef.current.textContent || "";
        const newHTML = renderContentWithMentions(currentText);
        if (editorRef.current.innerHTML !== newHTML) {
          editorRef.current.innerHTML = newHTML;

          // Restore cursor position immediately after DOM update
          restoreCursorPosition(savedPosition);
        }
      }

      // Check if we're typing an @ mention
      const beforeCursor = plainText.substring(0, cursorOffset);
      const lastAtIndex = beforeCursor.lastIndexOf("@");

      // Check if we're currently in a mention (@ followed by non-space characters)
      if (lastAtIndex !== -1) {
        const afterAt = beforeCursor.substring(lastAtIndex + 1);
        const isValidMention =
          !afterAt.includes(" ") && !afterAt.includes("\n");

        if (isValidMention) {
          // We're in a valid mention context
          if (!showMentions) {
            setShowMentions(true);
          }
          setMentionQuery(afterAt);
        } else {
          // Space or newline after @, close mentions
          if (showMentions) {
            setShowMentions(false);
            setMentionQuery("");
          }
        }
      } else {
        // No @ found before cursor
        if (showMentions) {
          setShowMentions(false);
          setMentionQuery("");
        }
      }
    };

    const insertMention = (user: UserSchema) => {
      const plainText = getPlainTextValue();
      const savedPosition = saveCursorPosition();
      const cursorOffset = savedPosition?.offset || 0;
      const beforeCursor = plainText.substring(0, cursorOffset);
      const afterCursor = plainText.substring(cursorOffset);
      const lastAtIndex = beforeCursor.lastIndexOf("@");

      if (lastAtIndex !== -1) {
        const beforeAt = plainText.substring(0, lastAtIndex);
        const newValue = `${beforeAt}@${user.username} ${afterCursor}`;

        // Update the content and render with mentions
        if (editorRef.current) {
          editorRef.current.innerHTML = renderContentWithMentions(newValue);
          onChange?.(newValue);
        }

        // Track mentioned user
        if (user.id && !mentionedUserIds.includes(user.id)) {
          setMentionedUserIds([...mentionedUserIds, user.id]);
        }

        // Set cursor position after the mention
        const newPosition = lastAtIndex + user.username.length + 2; // +2 for @ and space
        restoreCursorPosition({ offset: newPosition, atEnd: false });
        editorRef.current?.focus();
      }

      setShowMentions(false);
      setMentionQuery("");
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
      if (!showMentions || users.length === 0) {
        // Allow Enter key for new lines when not in mention mode
        return;
      }

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((prev) => (prev + 1) % users.length);
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((prev) => (prev - 1 + users.length) % users.length);
          break;
        case "Enter":
          e.preventDefault();
          if (users[selectedIndex]) {
            insertMention(users[selectedIndex]);
          }
          break;
        case "Escape":
          e.preventDefault();
          setShowMentions(false);
          setMentionQuery("");
          break;
      }
    };

    const handlePaste = (e: React.ClipboardEvent) => {
      e.preventDefault();
      const text = e.clipboardData.getData("text/plain");
      document.execCommand("insertText", false, text);
    };

    const minHeight = `${rows * 1.5}rem`;

    return (
      <Popover open={showMentions} onOpenChange={setShowMentions}>
        <PopoverAnchor asChild>
          <div className="relative">
            <div
              ref={editorRef}
              contentEditable={!disabled}
              onInput={handleInput}
              onKeyDown={handleKeyDown}
              onPaste={handlePaste}
              role="textbox"
              aria-multiline="true"
              aria-label={placeholder}
              className={cn(
                "block w-full rounded-md border border-muted-foreground/20 bg-muted px-3 py-2 text-sm",
                "min-h-[80px] max-h-[300px] overflow-y-auto",
                "shadow-xs",
                "focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-ring",
                "focus-visible:border-blue-600 focus-visible:ring-4 focus-visible:ring-blue-600/20",
                "transition-[border-color,box-shadow] duration-200 ease-in-out",
                "disabled:cursor-not-allowed disabled:opacity-50",
                "whitespace-pre-wrap break-words",
                isInvalid &&
                  "border-red-500 bg-red-500/20 ring-0 ring-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
                className,
              )}
              style={{ minHeight }}
              suppressContentEditableWarning
            />
            {isEmpty && !disabled && (
              <div className="pointer-events-none absolute inset-0 px-3 py-2 text-sm text-muted-foreground">
                {placeholder}
              </div>
            )}
          </div>
        </PopoverAnchor>
        <PopoverContent
          className="w-[300px] p-1"
          align="start"
          side="top"
          sideOffset={5}
          onOpenAutoFocus={(e) => {
            // Prevent focus from moving to popover
            e.preventDefault();
            editorRef.current?.focus();
          }}
        >
          <Command shouldFilter={false}>
            <CommandList>
              <CommandEmpty>
                {isSearching ? "Searching..." : "No users found"}
              </CommandEmpty>
              <CommandGroup>
                {users.map((user, index) => (
                  <CommandItem
                    key={user.id}
                    value={user.username}
                    onSelect={() => {
                      insertMention(user);
                      editorRef.current?.focus();
                    }}
                    className={cn(
                      "flex items-center gap-2 p-2 cursor-pointer",
                      index === selectedIndex && "bg-accent",
                    )}
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
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    );
  },
);

MentionTextarea.displayName = "MentionTextarea";
