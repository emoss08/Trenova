import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import type { SuggestionOptions, SuggestionProps } from "@tiptap/suggestion";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
  type RefObject,
  type UIEvent,
} from "react";
import { useDebounce } from "../../hooks/use-debounce";
import { cn } from "../../lib/utils";
import { Spinner } from "../ui/spinner";

export interface EditorSuggestionItem {
  id: string;
  label: string;
  description?: string | null;
  kind?: string;
}

export interface SuggestionPage {
  items: EditorSuggestionItem[];
  hasMore: boolean;
}

export type SuggestionFetcher = (query: string, page: number) => Promise<SuggestionPage>;

export interface SuggestionListHandle {
  onKeyDown: (event: KeyboardEvent) => boolean;
}

export interface SuggestionSessionState {
  prefix: string;
  query: string;
  getAnchorRect: () => DOMRect | null;
  command: (item: EditorSuggestionItem) => void;
}

export interface SuggestionController {
  open: (session: SuggestionSessionState) => void;
  close: () => void;
  handleKeyDown: (event: KeyboardEvent) => boolean;
}

interface SuggestionListProps {
  prefix: string;
  query: string;
  fetchPage: SuggestionFetcher;
  command: (item: EditorSuggestionItem) => void;
}

const KIND_LABELS: Record<string, string> = {
  shipment: "Shipment",
  worker: "Worker",
  customer: "Customer",
};

const QUERY_DEBOUNCE_MS = 250;
const SCROLL_BUFFER_PX = 50;

export const SuggestionList = forwardRef<SuggestionListHandle, SuggestionListProps>(
  ({ prefix, query, fetchPage, command }, ref) => {
    const [activeIndex, setActiveIndex] = useState(0);
    const debouncedQuery = useDebounce(query, QUERY_DEBOUNCE_MS);

    const { data, isLoading, isError, isFetching, isFetchingNextPage, hasNextPage, fetchNextPage } =
      useInfiniteQuery({
        queryKey: ["comment-suggestions", prefix, debouncedQuery],
        queryFn: ({ pageParam }) => fetchPage(debouncedQuery, pageParam),
        initialPageParam: 1,
        getNextPageParam: (lastPage, allPages) =>
          lastPage.hasMore ? allPages.length + 1 : undefined,
        placeholderData: keepPreviousData,
        staleTime: 60_000,
        retry: 1,
      });

    const items = useMemo(() => data?.pages.flatMap((page) => page.items) ?? [], [data?.pages]);

    useEffect(() => {
      setActiveIndex(0);
    }, [debouncedQuery, prefix]);

    useEffect(() => {
      if (items.length > 0 && activeIndex >= items.length) {
        setActiveIndex(items.length - 1);
      }
    }, [items.length, activeIndex]);

    useImperativeHandle(ref, () => ({
      onKeyDown: (event: KeyboardEvent) => {
        if (event.key === "ArrowDown") {
          setActiveIndex((index) => (items.length === 0 ? 0 : (index + 1) % items.length));
          return true;
        }
        if (event.key === "ArrowUp") {
          setActiveIndex((index) =>
            items.length === 0 ? 0 : (index - 1 + items.length) % items.length,
          );
          return true;
        }
        if (event.key === "Enter" || event.key === "Tab") {
          const selected = items[activeIndex];
          if (selected) {
            command(selected);
            return true;
          }
          return false;
        }
        return false;
      },
    }));

    const handleScroll = useCallback(
      (event: UIEvent<HTMLDivElement>) => {
        const target = event.currentTarget;
        const distanceFromBottom = target.scrollHeight - (target.scrollTop + target.clientHeight);
        if (distanceFromBottom <= SCROLL_BUFFER_PX && hasNextPage && !isFetching) {
          void fetchNextPage();
        }
      },
      [hasNextPage, isFetching, fetchNextPage],
    );

    const listRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
      const active = listRef.current?.querySelector<HTMLElement>(
        `[data-suggestion-index="${activeIndex}"]`,
      );
      active?.scrollIntoView({ block: "nearest" });
    }, [activeIndex]);

    if (isLoading) {
      return (
        <div className="flex justify-center p-4">
          <Spinner className="size-4" />
        </div>
      );
    }

    if (isError) {
      return <div className="p-2 text-sm text-red-500">Failed to load suggestions</div>;
    }

    if (items.length === 0) {
      return <div className="p-2 text-sm text-muted-foreground">No matches found</div>;
    }

    return (
      <div
        ref={listRef}
        role="listbox"
        aria-label={prefix === "@" ? "Mention suggestions" : "Reference suggestions"}
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain py-1"
        onScroll={handleScroll}
      >
        {items.map((item, index) => (
          <button
            key={`${item.kind ?? "user"}-${item.id}`}
            type="button"
            role="option"
            data-suggestion-index={index}
            aria-selected={index === activeIndex}
            className={cn(
              "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-accent",
              index === activeIndex && "bg-accent",
            )}
            onMouseEnter={() => setActiveIndex(index)}
            onMouseDown={(event) => {
              event.preventDefault();
              command(item);
            }}
          >
            <span className="flex min-w-0 flex-1 flex-col items-start">
              <span className="w-full truncate">{item.label}</span>
              {item.description && (
                <span className="w-full truncate text-2xs text-muted-foreground">
                  {item.description}
                </span>
              )}
            </span>
            {item.kind && KIND_LABELS[item.kind] && (
              <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-2xs text-muted-foreground">
                {KIND_LABELS[item.kind]}
              </span>
            )}
          </button>
        ))}
        {isFetchingNextPage && (
          <div className="flex justify-center p-2">
            <Spinner className="size-3.5" />
          </div>
        )}
        {hasNextPage && !isFetchingNextPage && (
          <div className="p-1.5 text-center text-2xs text-muted-foreground">Scroll for more</div>
        )}
      </div>
    );
  },
);

SuggestionList.displayName = "SuggestionList";

function toSession(
  prefix: string,
  props: SuggestionProps<EditorSuggestionItem>,
): SuggestionSessionState {
  return {
    prefix,
    query: props.query,
    getAnchorRect: () => props.clientRect?.() ?? null,
    command: props.command,
  };
}

export function createSuggestionRender(
  prefix: string,
  controllerRef: RefObject<SuggestionController>,
): NonNullable<SuggestionOptions<EditorSuggestionItem>["render"]> {
  return () => ({
    onStart: (props: SuggestionProps<EditorSuggestionItem>) => {
      controllerRef.current.open(toSession(prefix, props));
    },
    onUpdate: (props: SuggestionProps<EditorSuggestionItem>) => {
      controllerRef.current.open(toSession(prefix, props));
    },
    onKeyDown: (props: { event: KeyboardEvent }) => {
      if (props.event.key === "Escape") {
        controllerRef.current.close();
        return true;
      }
      return controllerRef.current.handleKeyDown(props.event);
    },
    onExit: () => {
      controllerRef.current.close();
    },
  });
}
