import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The search field of a toolbar or panel header: the product's input at the toolbar's 28px, the
 * search mark at its start, and a clear button at its end while it holds text (Escape clears it
 * too). The caller owns the text and any debounce; `className` sets its width (`w-56`).
 *
 *   <SearchInput
 *     value={search}
 *     onValueChange={setSearch}
 *     placeholder={t("Search changes")}
 *     className="w-56"
 *   />
 */
export function SearchInput({
  value,
  onValueChange,
  placeholder,
  "aria-label": ariaLabel,
  className,
}: {
  value: string;
  onValueChange: (value: string) => void;
  placeholder: string;
  /** What a screen reader calls the field; the placeholder when left out. */
  "aria-label"?: string;
  className?: string;
}) {
  const t = useT();
  return (
    <Input
      type="text"
      enterKeyHint="search"
      value={value}
      onChange={(event) => onValueChange(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === "Escape" && value !== "") {
          event.preventDefault();
          event.stopPropagation();
          onValueChange("");
        }
      }}
      placeholder={placeholder}
      aria-label={ariaLabel ?? placeholder}
      inputContainerClassName={className}
      leftElement={<SearchLgIcon className="size-3.5 text-muted-foreground" />}
      rightElement={
        value !== "" ? (
          <Button
            type="button"
            size="icon-xs"
            variant="ghost"
            aria-label={t("Clear the search")}
            onClick={() => onValueChange("")}
          >
            <XCloseIcon className="size-3" />
          </Button>
        ) : null
      }
    />
  );
}
