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
import { COMMENT_TYPES, CommentType } from "./utils";

interface MentionTextareaProps {
  value?: string;
  onChange?: (value: string) => void;
  onMentionedUsersChange?: (userIds: string[]) => void;
  onCommentTypeChange?: (type: CommentType | null) => void;
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

export const MentionTextarea = forwardRef<
  MentionTextareaRef,
  MentionTextareaProps
>(
  (
    {
      value = "",
      onChange,
      onMentionedUsersChange,
      onCommentTypeChange,
      searchUsers,
      placeholder = "Add a comment... Use @ to mention users, / for comment types",
      disabled = false,
      isInvalid,
      className,
      rows = 3,
    },
    ref,
  ) => {
    const [showMentions, setShowMentions] = useState(false);
    const [mentionQuery, setMentionQuery] = useState("");
    const [users, setUsers] = useState<UserSchema[]>([]);
    const [isSearching, setIsSearching] = useState(false);
    const [selectedIndex, setSelectedIndex] = useState(0);
    const [showSlashCommands, setShowSlashCommands] = useState(false);
    const [slashQuery, setSlashQuery] = useState("");
    const [selectedSlashIndex, setSelectedSlashIndex] = useState(0);
    const [selectedCommentType, setSelectedCommentType] =
      useState<CommentType | null>(null);
    const editorRef = useRef<HTMLDivElement>(null);
    const [isEmpty, setIsEmpty] = useState(true);

    const [mentionedUsers, setMentionedUsers] = useState<Map<string, string>>(
      new Map(),
    );

    useImperativeHandle(ref, () => ({
      focus: () => editorRef.current?.focus(),
      getValue: () => getPlainTextValue(),
    }));

    useEffect(() => {
      const userIds = Array.from(mentionedUsers.values());
      onMentionedUsersChange?.(userIds);
    }, [mentionedUsers, onMentionedUsersChange]);

    useEffect(() => {
      onCommentTypeChange?.(selectedCommentType);
    }, [selectedCommentType, onCommentTypeChange]);

    const updateMentionedUsersFromText = (text: string) => {
      const mentionRegex = /@(\w+)/g;
      const matches = text.matchAll(mentionRegex);
      const currentUsernames = new Set<string>();

      for (const match of matches) {
        const username = match[1];
        if (username) {
          currentUsernames.add(username);
        }
      }

      setMentionedUsers((prev) => {
        const newMap = new Map(prev);
        for (const [username] of newMap) {
          if (!currentUsernames.has(username)) {
            newMap.delete(username);
          }
        }
        return newMap;
      });
    };

    useEffect(() => {
      if (editorRef.current) {
        const currentText = getPlainTextValue();
        if (value !== currentText) {
          if (value.startsWith("/")) {
            const match = value.match(/^\/(\w+)\s/);
            if (match) {
              const commandLabel = match[1];
              const foundType = COMMENT_TYPES.find(
                (t) => t.label.toLowerCase() === commandLabel.toLowerCase(),
              );
              if (foundType) {
                setSelectedCommentType(foundType.value);
              }
            }
          }

          editorRef.current.innerHTML = renderContentWithMentions(value);
          setIsEmpty(value.length === 0);
          if (!value) {
            setMentionedUsers(new Map());
            setSelectedCommentType(null);
          }
        }
      }
    }, [value]);

    const renderContentWithMentions = (text: string): string => {
      if (!text) return "";

      const escapeHtml = (str: string) => {
        const div = document.createElement("div");
        div.textContent = str;
        return div.innerHTML;
      };

      let html = escapeHtml(text);

      COMMENT_TYPES.forEach((type) => {
        const regex = new RegExp(`^(/${type.label})(\\s|$)`, "gi");
        html = html.replace(regex, (_, command, space) => {
          return `<span class="inline px-1 py-0.5 rounded-sm font-medium ${type.className}" data-command="${type.value}">${command}</span>${space}`;
        });
      });

      const mentionRegex = /@(\w*)/g;
      html = html.replace(mentionRegex, (match, username) => {
        if (match === "@") {
          return `<span class="inline bg-blue-100 text-blue-700 px-1 py-0.5 rounded-sm font-medium" data-mention="">@</span>`;
        }
        return `<span class="inline bg-blue-100 text-blue-700 px-1 py-0.5 rounded-sm font-medium" data-mention="${username}">${match}</span>`;
      });

      return html;
    };

    const getPlainTextValue = (): string => {
      if (!editorRef.current) return "";
      return editorRef.current.textContent || "";
    };

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

    const saveCursorPosition = () => {
      const selection = window.getSelection();
      if (!selection || selection.rangeCount === 0 || !editorRef.current)
        return null;

      const range = selection.getRangeAt(0);
      const preCaretRange = range.cloneRange();
      preCaretRange.selectNodeContents(editorRef.current);
      preCaretRange.setEnd(range.endContainer, range.endOffset);

      const textUpToCursor = preCaretRange.toString();

      return {
        offset: textUpToCursor.length,
        atEnd:
          range.collapsed &&
          range.endOffset === range.endContainer.textContent?.length,
      };
    };

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

      const savedPosition = saveCursorPosition();
      const cursorOffset = savedPosition?.offset || 0;

      onChange?.(plainText);
      setIsEmpty(plainText.length === 0);

      updateMentionedUsersFromText(plainText);

      // Check if the comment type should be cleared
      if (!plainText.startsWith("/")) {
        setSelectedCommentType(null);
      }

      if (editorRef.current) {
        const currentText = editorRef.current.textContent || "";
        const newHTML = renderContentWithMentions(currentText);
        if (editorRef.current.innerHTML !== newHTML) {
          editorRef.current.innerHTML = newHTML;

          restoreCursorPosition(savedPosition);
        }
      }

      const beforeCursor = plainText.substring(0, cursorOffset);

      let isHandlingSlashCommand = false;
      if (plainText.startsWith("/") && cursorOffset <= 20) {
        const slashMatch = plainText.match(/^\/(\w*)/);
        if (slashMatch) {
          const query = slashMatch[1];
          const matchedType = COMMENT_TYPES.find(
            (type) => type.label.toLowerCase() === query.toLowerCase(),
          );

          if (
            matchedType &&
            (plainText === `/${matchedType.label}` ||
              plainText.startsWith(`/${matchedType.label} `))
          ) {
            setShowSlashCommands(false);
            setSlashQuery("");
            // Set the comment type when a complete match is found
            setSelectedCommentType(matchedType.value);
          } else if (cursorOffset <= slashMatch[0].length) {
            isHandlingSlashCommand = true;
            setShowSlashCommands(true);
            setSlashQuery(query);
            setShowMentions(false);
            setMentionQuery("");
          }
        }
      }

      if (!isHandlingSlashCommand) {
        setShowSlashCommands(false);
        setSlashQuery("");
      }

      // Always check for mentions regardless of slash commands
      const lastAtIndex = beforeCursor.lastIndexOf("@");

      if (lastAtIndex !== -1) {
        const afterAt = beforeCursor.substring(lastAtIndex + 1);
        const isValidMention =
          !afterAt.includes(" ") && !afterAt.includes("\n");

        if (isValidMention) {
          setShowMentions(true);
          setMentionQuery(afterAt);
          if (!isHandlingSlashCommand) {
            setShowSlashCommands(false);
            setSlashQuery("");
          }
        } else {
          setShowMentions(false);
          setMentionQuery("");
        }
      } else {
        setShowMentions(false);
        setMentionQuery("");
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

        if (editorRef.current) {
          editorRef.current.innerHTML = renderContentWithMentions(newValue);
          onChange?.(newValue);
        }

        if (user.id && user.username) {
          setMentionedUsers((prev) => {
            const newMap = new Map(prev);
            newMap.set(user.username, user.id || "");
            return newMap;
          });
        }

        const newPosition = lastAtIndex + user.username.length + 2; // +2 for @ and space
        restoreCursorPosition({ offset: newPosition, atEnd: false });
        editorRef.current?.focus();
      }

      setShowMentions(false);
      setMentionQuery("");
    };

    const insertSlashCommand = (commandType: CommentType) => {
      const plainText = getPlainTextValue();
      const typeConfig = COMMENT_TYPES.find((t) => t.value === commandType);

      if (!typeConfig) return;

      // Replace the slash command at the beginning
      const newValue = plainText.replace(/^\/\w*\s?/, `/${typeConfig.label} `);

      if (editorRef.current) {
        // Update content and notify parent
        editorRef.current.innerHTML = renderContentWithMentions(newValue);
        onChange?.(newValue);

        // Set the selected comment type
        setSelectedCommentType(commandType);

        // Position cursor after the slash command
        setTimeout(() => {
          if (editorRef.current) {
            const range = document.createRange();
            const selection = window.getSelection();
            const textNodes = getAllTextNodes(editorRef.current);

            if (textNodes.length > 0) {
              const firstTextNode = textNodes[0];
              const offset = Math.min(
                `/${typeConfig.label} `.length,
                firstTextNode.textContent?.length || 0,
              );
              range.setStart(firstTextNode, offset);
              range.collapse(true);

              if (selection) {
                selection.removeAllRanges();
                selection.addRange(range);
              }
            }

            editorRef.current.focus();
          }
        }, 0);
      }

      setShowSlashCommands(false);
      setSlashQuery("");
    };

    const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
      // Handle slash commands
      if (showSlashCommands) {
        const filteredTypes = COMMENT_TYPES.filter((type) =>
          type.label.toLowerCase().includes(slashQuery.toLowerCase()),
        );

        switch (e.key) {
          case "ArrowDown":
            e.preventDefault();
            setSelectedSlashIndex((prev) => (prev + 1) % filteredTypes.length);
            break;
          case "ArrowUp":
            e.preventDefault();
            setSelectedSlashIndex(
              (prev) =>
                (prev - 1 + filteredTypes.length) % filteredTypes.length,
            );
            break;
          case "Enter":
            e.preventDefault();
            if (filteredTypes[selectedSlashIndex]) {
              insertSlashCommand(filteredTypes[selectedSlashIndex].value);
            }
            break;
          case "Escape":
            e.preventDefault();
            setShowSlashCommands(false);
            setSlashQuery("");
            break;
        }
        return;
      }

      // Handle mentions
      if (!showMentions || users.length === 0) {
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
      <>
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

        {/* Slash Commands Popover */}
        <Popover open={showSlashCommands} onOpenChange={setShowSlashCommands}>
          <PopoverAnchor asChild>
            <div className="absolute" />
          </PopoverAnchor>
          <PopoverContent
            className="w-[250px] p-1"
            align="start"
            side="top"
            sideOffset={5}
            onOpenAutoFocus={(e) => {
              e.preventDefault();
              editorRef.current?.focus();
            }}
          >
            <Command shouldFilter={false}>
              <CommandList>
                <CommandEmpty>No comment types found</CommandEmpty>
                <CommandGroup heading="Comment Types">
                  {COMMENT_TYPES.filter((type) =>
                    type.label.toLowerCase().includes(slashQuery.toLowerCase()),
                  ).map((type, index) => (
                    <CommandItem
                      key={type.value}
                      value={type.value}
                      onSelect={() => {
                        insertSlashCommand(type.value);
                        editorRef.current?.focus();
                      }}
                      className={cn(
                        "flex items-center gap-2 p-2 cursor-pointer",
                        index === selectedSlashIndex && "bg-accent",
                      )}
                    >
                      <span className="text-lg">{type.icon}</span>
                      <div className="flex flex-col">
                        <span className="text-sm font-medium">
                          {type.label}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {type.description}
                        </span>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      </>
    );
  },
);

MentionTextarea.displayName = "MentionTextarea";
