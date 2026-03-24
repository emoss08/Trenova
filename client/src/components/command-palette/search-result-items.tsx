import Highlight from "@/components/highlight";
import { Badge } from "@/components/ui/badge";
import { CommandItem } from "@/components/ui/command";
import { cn } from "@/lib/utils";
import type { GlobalSearchHit } from "@/services/global-search";
import { ArrowRight, FileText, Search, Truck, User, Users } from "lucide-react";

const entityIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  shipment: Truck,
  customer: User,
  worker: Users,
  document: FileText,
};

function ResultItemContainer({
  className,
  children,
  ...props
}: React.ComponentProps<typeof CommandItem>) {
  return (
    <CommandItem
      className={cn(
        "group h-full rounded-md border border-transparent !px-3 font-medium",
        "data-[selected=true]:border-input data-[selected=true]:bg-input/50",
        className,
      )}
      {...props}
    >
      {children}
    </CommandItem>
  );
}

export function ShipmentResultItem({
  hit,
  searchValue,
  onSelect,
  onPreview,
}: {
  hit: GlobalSearchHit;
  searchValue: string;
  onSelect: () => void;
  onPreview?: (id: string) => void;
}) {
  const meta = hit.metadata;
  return (
    <ResultItemContainer
      value={`${searchValue} ${hit.title} ${hit.subtitle ?? ""} shipment`}
      onSelect={onSelect}
      data-id={hit.id}
      data-entity-type="shipment"
      onMouseEnter={() => onPreview?.(hit.id)}
      onFocus={() => onPreview?.(hit.id)}
    >
      <Truck className="size-4 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-col">
        <p className="truncate text-sm font-medium">
          <Highlight highlight={searchValue} text={meta?.proNumber ?? hit.title} />
        </p>
        <div className="flex items-center gap-2 overflow-hidden text-2xs text-muted-foreground">
          {(meta?.customerName || meta?.customerCode) && (
            <p className="max-w-[220px] truncate">
              {meta?.customerName ? (
                <>
                  <Highlight highlight={searchValue} text={meta.customerName} />
                  {meta?.customerCode && <span className="opacity-70"> ({meta.customerCode})</span>}
                </>
              ) : (
                meta?.customerCode
              )}
            </p>
          )}
          {meta?.bol && (meta?.customerName || meta?.customerCode) && (
            <span className="text-border">&bull;</span>
          )}
          {meta?.bol && (
            <p className="max-w-[140px] truncate">
              <span className="opacity-70">BOL</span>
              <span className="ml-1">
                <Highlight highlight={searchValue} text={meta.bol} />
              </span>
            </p>
          )}
          {meta?.serviceTypeCode && (meta?.customerName || meta?.customerCode || meta?.bol) && (
            <span className="text-border">&bull;</span>
          )}
          {meta?.serviceTypeCode && (
            <Badge variant="secondary" className="max-h-5 px-1.5 py-0 text-[10px]">
              {meta.serviceTypeCode}
            </Badge>
          )}
        </div>
      </div>
      <ArrowRight className="ml-auto size-4 opacity-0 transition-opacity group-hover:opacity-100 group-data-[selected=true]:opacity-100" />
    </ResultItemContainer>
  );
}

export function CustomerResultItem({
  hit,
  searchValue,
  onSelect,
}: {
  hit: GlobalSearchHit;
  searchValue: string;
  onSelect: () => void;
}) {
  const meta = hit.metadata;
  return (
    <ResultItemContainer
      value={`${searchValue} ${hit.title} ${hit.subtitle ?? ""} customer`}
      onSelect={onSelect}
    >
      <User className="size-4 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-col">
        <p className="truncate text-sm font-medium">
          <Highlight highlight={searchValue} text={hit.title} />
        </p>
        <div className="flex items-center gap-2 text-2xs text-muted-foreground">
          {meta?.code && (
            <p className="truncate">
              <Highlight highlight={searchValue} text={meta.code} />
            </p>
          )}
          {hit.subtitle && (
            <>
              {meta?.code && <span className="text-border">&bull;</span>}
              <p className="truncate">
                <Highlight highlight={searchValue} text={hit.subtitle} />
              </p>
            </>
          )}
        </div>
      </div>
      <ArrowRight className="ml-auto size-4 opacity-0 transition-opacity group-hover:opacity-100 group-data-[selected=true]:opacity-100" />
    </ResultItemContainer>
  );
}

export function WorkerResultItem({
  hit,
  searchValue,
  onSelect,
}: {
  hit: GlobalSearchHit;
  searchValue: string;
  onSelect: () => void;
}) {
  const meta = hit.metadata;
  return (
    <ResultItemContainer
      value={`${searchValue} ${hit.title} ${hit.subtitle ?? ""} worker`}
      onSelect={onSelect}
    >
      <Users className="size-4 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-col">
        <p className="truncate text-sm font-medium">
          <Highlight highlight={searchValue} text={hit.title} />
        </p>
        <div className="flex items-center gap-2 text-2xs text-muted-foreground">
          {meta?.workerType && (
            <Badge variant="secondary" className="max-h-5 px-1.5 py-0 text-[10px]">
              {meta.workerType}
            </Badge>
          )}
          {hit.subtitle && (
            <p className="truncate">
              <Highlight highlight={searchValue} text={hit.subtitle} />
            </p>
          )}
        </div>
      </div>
      <ArrowRight className="ml-auto size-4 opacity-0 transition-opacity group-hover:opacity-100 group-data-[selected=true]:opacity-100" />
    </ResultItemContainer>
  );
}

export function DocumentResultItem({
  hit,
  searchValue,
  onSelect,
}: {
  hit: GlobalSearchHit;
  searchValue: string;
  onSelect: () => void;
}) {
  const meta = hit.metadata;
  return (
    <ResultItemContainer
      value={`${searchValue} ${hit.title} ${hit.subtitle ?? ""} document`}
      onSelect={onSelect}
    >
      <FileText className="size-4 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-col">
        <p className="truncate text-sm font-medium">
          <Highlight highlight={searchValue} text={hit.title} />
        </p>
        <div className="flex items-center gap-2 text-2xs text-muted-foreground">
          {meta?.documentType && (
            <Badge variant="secondary" className="max-h-5 px-1.5 py-0 text-[10px]">
              {meta.documentType}
            </Badge>
          )}
          {hit.subtitle && (
            <p className="truncate">
              <Highlight highlight={searchValue} text={hit.subtitle} />
            </p>
          )}
        </div>
      </div>
      <ArrowRight className="ml-auto size-4 opacity-0 transition-opacity group-hover:opacity-100 group-data-[selected=true]:opacity-100" />
    </ResultItemContainer>
  );
}

export function GenericResultItem({
  hit,
  searchValue,
  onSelect,
}: {
  hit: GlobalSearchHit;
  searchValue: string;
  onSelect: () => void;
}) {
  const Icon = entityIcons[hit.entityType] ?? Search;
  return (
    <ResultItemContainer
      value={`${searchValue} ${hit.title} ${hit.subtitle ?? ""} ${hit.entityType}`}
      onSelect={onSelect}
    >
      <Icon className="size-4 shrink-0" />
      <div className="flex min-w-0 flex-1 flex-col">
        <p className="truncate text-sm font-medium">
          <Highlight highlight={searchValue} text={hit.title} />
        </p>
        {hit.subtitle && (
          <p className="truncate text-2xs text-muted-foreground">
            <Highlight highlight={searchValue} text={hit.subtitle} />
          </p>
        )}
      </div>
      <ArrowRight className="ml-auto size-4 opacity-0 transition-opacity group-hover:opacity-100 group-data-[selected=true]:opacity-100" />
    </ResultItemContainer>
  );
}

export function SearchResultItem({
  hit,
  searchValue,
  onSelect,
  onPreview,
}: {
  hit: GlobalSearchHit;
  searchValue: string;
  onSelect: () => void;
  onPreview?: (id: string) => void;
}) {
  switch (hit.entityType) {
    case "shipment":
      return (
        <ShipmentResultItem
          hit={hit}
          searchValue={searchValue}
          onSelect={onSelect}
          onPreview={onPreview}
        />
      );
    case "customer":
      return <CustomerResultItem hit={hit} searchValue={searchValue} onSelect={onSelect} />;
    case "worker":
      return <WorkerResultItem hit={hit} searchValue={searchValue} onSelect={onSelect} />;
    case "document":
      return <DocumentResultItem hit={hit} searchValue={searchValue} onSelect={onSelect} />;
    default:
      return <GenericResultItem hit={hit} searchValue={searchValue} onSelect={onSelect} />;
  }
}
