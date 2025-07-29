/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
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
import { Textarea } from "@/components/ui/textarea";
import { UserSchema } from "@/lib/schemas/user-schema";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback, useEffect, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import * as z from "zod/v4";

// Create a schema for the form that only includes the fields we need
const commentFormSchema = z.object({
  comment: z.string().min(1, {
    error: "Comment is required",
  }),
  isHighPriority: z.boolean(),
});

type CommentFormValues = z.infer<typeof commentFormSchema>;

interface CommentFormProps {
  onSubmit: (comment: string, mentionedUsers: string[]) => Promise<void>;
  searchUsers: (query: string) => Promise<UserSchema[]>;
  className?: string;
  placeholder?: string;
  disabled?: boolean;
}

export function CommentForm({
  onSubmit,
  searchUsers,
  className,
  placeholder = "Add a comment... Use @ to mention users",
  disabled = false,
}: CommentFormProps) {
  const [mentionedUserIds, setMentionedUserIds] = useState<string[]>([]);
  const [showMentions, setShowMentions] = useState(false);
  const [mentionQuery, setMentionQuery] = useState("");
  const [users, setUsers] = useState<UserSchema[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [cursorPosition, setCursorPosition] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const form = useForm<CommentFormValues>({
    resolver: zodResolver(commentFormSchema),
    defaultValues: {
      comment: "",
      isHighPriority: false,
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { isSubmitting },
  } = form;

  const commentValue = watch("comment");
  const hasContent = commentValue.trim().length > 0;

  // Search users debounced
  const searchUsersDebounced = useCallback(
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
    [searchUsers],
  );

  useEffect(() => {
    if (mentionQuery) {
      searchUsersDebounced(mentionQuery);
    } else {
      setUsers([]);
    }
  }, [mentionQuery, searchUsersDebounced]);

  const handleInputChange = (value: string, cursorPos?: number) => {
    setValue("comment", value);
    const currentCursorPos =
      cursorPos ?? textareaRef.current?.selectionStart ?? 0;
    setCursorPosition(currentCursorPos);

    // Check if we're typing an @ mention
    const beforeCursor = value.substring(0, currentCursorPos);
    const lastAtIndex = beforeCursor.lastIndexOf("@");

    // Check if we're currently in a mention (@ followed by non-space characters)
    if (lastAtIndex !== -1) {
      const afterAt = beforeCursor.substring(lastAtIndex + 1);
      const isValidMention = !afterAt.includes(" ") && !afterAt.includes("\n");

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
    const value = commentValue;
    const beforeCursor = value.substring(0, cursorPosition);
    const afterCursor = value.substring(cursorPosition);
    const lastAtIndex = beforeCursor.lastIndexOf("@");

    if (lastAtIndex !== -1) {
      const beforeAt = value.substring(0, lastAtIndex);
      const newValue = `${beforeAt}@${user.username} ${afterCursor}`;
      setValue("comment", newValue);

      // Track mentioned user
      if (user.id && !mentionedUserIds.includes(user.id)) {
        setMentionedUserIds([...mentionedUserIds, user.id]);
      }

      // Set cursor position after the mention
      setTimeout(() => {
        if (textareaRef.current) {
          const newPosition = lastAtIndex + user.username.length + 2; // +2 for @ and space
          textareaRef.current.setSelectionRange(newPosition, newPosition);
          textareaRef.current.focus();
        }
      }, 0);
    }

    setShowMentions(false);
    setMentionQuery("");
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (!showMentions || users.length === 0) return;

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

  const handleFormSubmit = async (values: CommentFormValues) => {
    try {
      await onSubmit(values.comment, mentionedUserIds);
      // Reset form and mentioned users on successful submission
      reset();
      setMentionedUserIds([]);
    } catch (error) {
      console.error("Failed to submit comment:", error);
    }
  };

  const handleCancel = () => {
    reset();
    setMentionedUserIds([]);
  };

  return (
    <form
      onSubmit={handleSubmit(handleFormSubmit)}
      className={cn("space-y-2", className)}
    >
      <div className="relative">
        <Popover open={showMentions} onOpenChange={setShowMentions}>
          <PopoverAnchor asChild>
            <Controller
              name="comment"
              control={control}
              render={({ field, fieldState }) => (
                <Textarea
                  ref={textareaRef}
                  value={field.value}
                  onChange={(e) => {
                    handleInputChange(e.target.value, e.target.selectionStart);
                  }}
                  onKeyDown={handleKeyDown}
                  placeholder={placeholder}
                  rows={3}
                  className="min-h-[80px] resize-none"
                  isInvalid={!!fieldState.error}
                  disabled={disabled || isSubmitting}
                />
              )}
            />
          </PopoverAnchor>
          <PopoverContent
            className="w-[300px] p-1"
            align="start"
            side="top"
            sideOffset={5}
            onOpenAutoFocus={(e) => {
              // Prevent focus from moving to popover
              e.preventDefault();
              textareaRef.current?.focus();
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
                        textareaRef.current?.focus();
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
      </div>
      <div className="flex justify-end gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleCancel}
          disabled={!hasContent || isSubmitting || disabled}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          size="sm"
          disabled={!hasContent || isSubmitting || disabled}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            handleSubmit(handleFormSubmit)(e);
          }}
        >
          {isSubmitting ? "Posting..." : "Post Comment"}
        </Button>
      </div>
    </form>
  );
}
