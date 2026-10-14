import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { countMatches, createMatcher } from "@trenova/shared/lib/highlight/code";
import { cn } from "@trenova/shared/lib/utils";
import { SearchLgIcon } from "@trenova/shared/components/icons";
import { useMemo, useState } from "react";
import { buildJsonPathIndex } from "./audit-json-path";

const LINE_HEIGHT_PX = 18;
const PADDING_PX = 24;
const MIN_HEIGHT = 56;
const MAX_HEIGHT = 288;

const REDACTED = /\[REDACTED\]|\*{4,}/;

function SensitiveBadge() {
  const t = useT();
  return (
    <Badge
      variant="warning"
      title={t("Sensitive data omitted")}
      className="ml-1.5 h-4 px-1.5 align-middle text-2xs select-none"
    >
      {t("Sensitive")}
    </Badge>
  );
}

function sensitiveLineEnd(text: string) {
  return REDACTED.test(text) ? <SensitiveBadge /> : null;
}

export function AuditJsonBlock({
  value,
  className,
  searchable = false,
  copyPath = false,
}: {
  value: unknown;
  className?: string;
  searchable?: boolean;
  copyPath?: boolean;
}) {
  const t = useT();
  const [searchQuery, setSearchQuery] = useState("");
  const debouncedQuery = useDebounce(searchQuery, 150);
  const { copy } = useCopyToClipboard();

  const jsonCode = useMemo(() => {
    try {
      return JSON.stringify(value ?? null, null, 2);
    } catch {
      return '"Unable to serialize value"';
    }
  }, [value]);

  const scrollHeight = useMemo(() => {
    const lineCount = jsonCode.split("\n").length;
    const estimated = lineCount * LINE_HEIGHT_PX + PADDING_PX;
    return Math.min(Math.max(estimated, MIN_HEIGHT), MAX_HEIGHT);
  }, [jsonCode]);

  const pathIndex = useMemo(
    () => (copyPath ? buildJsonPathIndex(jsonCode) : new Map<number, string>()),
    [jsonCode, copyPath],
  );

  const query = searchable ? debouncedQuery : "";
  const matchCount = useMemo(() => countMatches(jsonCode, createMatcher(query)), [jsonCode, query]);

  const onLineClick =
    copyPath && pathIndex.size > 0
      ? (line: number) => {
          const path = pathIndex.get(line);
          if (path) void copy(path, { withToast: true });
        }
      : undefined;

  const bare = !(searchable || copyPath);
  const code = (
    <ScrollArea
      className={bare ? cn("border-border/80 rounded-md border", className) : undefined}
      style={{ height: scrollHeight }}
    >
      <HighlightedCode
        code={jsonCode}
        language="json"
        query={query}
        className="p-3"
        lineClassName={cn(copyPath && "hover:bg-muted/50 cursor-pointer")}
        lineEnd={sensitiveLineEnd}
        onLineClick={onLineClick}
      />
    </ScrollArea>
  );

  if (bare) {
    return code;
  }

  return (
    <div className={cn("border-border/80 overflow-hidden rounded-md border", className)}>
      <div className="border-border/60 flex items-center gap-2 border-b px-2 py-1.5">
        {searchable && (
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t("Search JSON...")}
            leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
            rightElement={
              query.trim() ? (
                <span className="text-muted-foreground pr-1 text-2xs">
                  {t("{0, plural, one {# match} other {# matches}}", matchCount)}
                </span>
              ) : undefined
            }
            className="h-6 bg-transparent text-xs"
          />
        )}
        {copyPath && (
          <span className="text-muted-foreground shrink-0 text-2xs">
            {t("Click line to copy path")}
          </span>
        )}
      </div>
      {code}
    </div>
  );
}
