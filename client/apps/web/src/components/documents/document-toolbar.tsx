import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import {
  ArrowDownAZIcon,
  ArrowUpAZIcon,
  LayoutGridIcon,
  ListIcon,
  SearchIcon,
  UploadIcon,
} from "lucide-react";

export type FileTypeFilter = "all" | "pdf" | "images" | "documents" | "spreadsheets";
export type SortField = "name" | "date" | "size";
export type SortDirection = "asc" | "desc";
export type ViewMode = "grid" | "list";

interface DocumentToolbarProps {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  fileTypeFilter: FileTypeFilter;
  onFileTypeFilterChange: (filter: FileTypeFilter) => void;
  sortField: SortField;
  onSortFieldChange: (field: SortField) => void;
  sortDirection: SortDirection;
  onSortDirectionChange: (direction: SortDirection) => void;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  onUploadClick?: () => void;
  disabled?: boolean;
  className?: string;
}

const fileTypeOptions: { value: FileTypeFilter; label: string }[] = [
  { value: "all", label: "All Files" },
  { value: "pdf", label: "PDF" },
  { value: "images", label: "Images" },
  { value: "documents", label: "Documents" },
  { value: "spreadsheets", label: "Spreadsheets" },
];

const sortFieldOptions: { value: SortField; label: string }[] = [
  { value: "date", label: "Date" },
  { value: "name", label: "Name" },
  { value: "size", label: "Size" },
];

export function DocumentToolbar({
  searchQuery,
  onSearchChange,
  fileTypeFilter,
  onFileTypeFilterChange,
  sortField,
  onSortFieldChange,
  sortDirection,
  onSortDirectionChange,
  viewMode,
  onViewModeChange,
  onUploadClick,
  disabled,
  className,
}: DocumentToolbarProps) {
  const toggleSortDirection = () => {
    onSortDirectionChange(sortDirection === "asc" ? "desc" : "asc");
  };

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      <div className="flex-1">
        <Input
          placeholder="Search..."
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          leftElement={<SearchIcon className="size-4 text-muted-foreground" />}
          className="max-w-[200px]"
        />
      </div>

      <Select
        value={fileTypeFilter}
        items={fileTypeOptions}
        onValueChange={(val) => onFileTypeFilterChange(val as FileTypeFilter)}
      >
        <SelectTrigger>
          <SelectValue placeholder="Select file type" />
        </SelectTrigger>
        <SelectContent>
          {fileTypeOptions.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <ButtonGroup>
        <Select
          value={sortField}
          items={sortFieldOptions}
          onValueChange={(val) => onSortFieldChange(val as SortField)}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select sort field" />
          </SelectTrigger>
          <SelectContent>
            {sortFieldOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="icon-sm"
          onClick={toggleSortDirection}
          aria-label={`Sort ${sortDirection === "asc" ? "descending" : "ascending"}`}
        >
          {sortDirection === "asc" ? (
            <ArrowUpAZIcon className="size-4" />
          ) : (
            <ArrowDownAZIcon className="size-4" />
          )}
        </Button>
      </ButtonGroup>

      <ButtonGroup>
        <Button
          variant={viewMode === "grid" ? "secondary" : "outline"}
          size="icon-sm"
          onClick={() => onViewModeChange("grid")}
          aria-label="Grid view"
          aria-pressed={viewMode === "grid"}
        >
          <LayoutGridIcon className="size-4" />
        </Button>
        <Button
          variant={viewMode === "list" ? "secondary" : "outline"}
          size="icon-sm"
          onClick={() => onViewModeChange("list")}
          aria-label="List view"
          aria-pressed={viewMode === "list"}
        >
          <ListIcon className="size-4" />
        </Button>
      </ButtonGroup>

      {onUploadClick && (
        <Button variant="secondary" size="sm" onClick={onUploadClick} disabled={disabled}>
          <UploadIcon className="size-4" />
          Upload
        </Button>
      )}
    </div>
  );
}
