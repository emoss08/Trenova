import { Icon } from "@/components/ui/icons";
import { SearchResult, SiteSearchQuickOptionProps } from "@/types/search";
import { faTruck } from "@fortawesome/pro-regular-svg-icons";
import { ShipmentStatusBadge } from "../status-badge";
import Highlight from "../ui/highlight";

interface SearchResultItemProps {
  result: SearchResult;
  searchQuery: string;
  onClick: (result: SearchResult) => void;
  highlighted?: boolean;
}

// Shipment result component
export function ShipmentResultItem(props: SearchResultItemProps) {
  const { result, onClick, searchQuery } = props;

  return (
    <div
      className="flex items-center gap-3 p-1 cursor-pointer rounded-md transition-colors hover:bg-accent"
      onClick={() => onClick(result)}
    >
      <div className="flex-shrink-0 flex items-center justify-center size-10 rounded-full bg-accent/30">
        <div className="flex items-center justify-center bg-transparent border border-border size-8 rounded-md">
          <Icon icon={faTruck} className="size-5 text-foreground" />
        </div>
      </div>
      <div className="flex-grow min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="font-medium truncate">
            <Highlight highlight={searchQuery} text={result.title} />
          </h4>
          <ShipmentStatusBadge
            status={result.metadata?.status}
            withDot={false}
          />
        </div>
        <p className="text-xs text-muted-foreground truncate">
          <Highlight highlight={searchQuery} text={result.metadata?.bol} />
        </p>
      </div>
    </div>
  );
}

export function SiteSearchQuickOption({
  icon,
  label,
  description,
  onClick,
}: SiteSearchQuickOptionProps) {
  return (
    <button
      onClick={onClick}
      className="flex items-center gap-2 p-2 rounded-md hover:bg-accent/50 transition-colors cursor-pointer outline-none"
    >
      <Icon icon={icon} className="size-4 text-muted-foreground" />
      <div className="flex flex-col text-left">
        <span className="text-sm font-medium">{label}</span>
        <span className="text-2xs text-muted-foreground">{description}</span>
      </div>
    </button>
  );
}

// Factory function to get the correct component based on result type
export function getResultComponent(
  result: SearchResult,
  props: Omit<SearchResultItemProps, "result">,
) {
  switch (result.type) {
    case "shipment":
      return <ShipmentResultItem result={result} {...props} />;
    default:
      // Fallback for unknown types
      return (
        <div className="px-4 py-2 hover:bg-accent/30 cursor-pointer rounded-md">
          {result.title}
        </div>
      );
  }
}
