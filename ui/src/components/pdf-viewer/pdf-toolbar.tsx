/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { cn } from "@/lib/utils";
import {
  faBarsSort,
  faChevronLeft,
  faChevronRight,
  faMagnifyingGlass,
  faMinus,
  faPlus,
  faRotate,
} from "@fortawesome/pro-solid-svg-icons";
import { Button } from "../ui/button";
import { Icon } from "../ui/icons";
import { Input } from "../ui/input";

export function PDFSearchBar({
  searchText,
  handleSearchChange,
  handleInputKeyDown,
}: {
  searchText: string;
  handleSearchChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  handleInputKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
}) {
  return (
    <div className="bg-background border-b border-input pl-4 pr-12 py-3">
      <div className="flex items-center gap-2">
        <div className="relative w-full">
          <Input
            icon={<Icon icon={faMagnifyingGlass} className="size-3" />}
            id="pdf-search-input"
            type="search"
            placeholder="Search document..."
            value={searchText}
            onChange={handleSearchChange}
            onKeyDown={handleInputKeyDown}
            className="w-full"
          />
        </div>
      </div>
    </div>
  );
}

export function PDFNavigationBar({
  showOutline,
  setShowOutline,
  hasOutline,
  previousPage,
  nextPage,
  pageNumber,
  numPages,
  zoomOut,
  zoomIn,
  rotate,
  toggleSearch,
  searchText,
  showSearch,
  totalMatches,
  currentMatchIndex,
  navigateMatches,
  scale,
}: {
  showOutline: boolean;
  setShowOutline: (showOutline: boolean) => void;
  hasOutline: boolean;
  previousPage: () => void;
  nextPage: () => void;
  pageNumber: number;
  numPages: number;
  zoomOut: () => void;
  zoomIn: () => void;
  rotate: (degrees: number) => void;
  toggleSearch: () => void;
  searchText: string;
  showSearch: boolean;
  totalMatches: number;
  currentMatchIndex: number;
  navigateMatches: (direction: "prev" | "next") => void;
  scale: number;
}) {
  return (
    <div className="sticky top-0 z-10 bg-background border-b border-input pl-4 pr-12 py-2">
      <div className="flex items-center justify-between">
        {/* Left side: Primary controls */}
        <div className="flex items-center space-x-2">
          {/* Document outline button */}
          <Button
            onClick={() => setShowOutline(!showOutline)}
            disabled={!hasOutline && !showOutline}
            variant="outline"
            className={cn(
              "text-muted-foreground",
              showOutline
                ? "bg-muted-foreground/20 border-muted-foreground/20"
                : "",
              !hasOutline && !showOutline
                ? "opacity-50 cursor-not-allowed"
                : "",
            )}
            aria-label="Toggle outline"
          >
            <Icon icon={faBarsSort} className="size-4 mr-1" />
            <span className="hidden sm:inline">Outline</span>
            {!hasOutline && !showOutline && (
              <span className="hidden sm:inline"> (None)</span>
            )}
          </Button>

          {/* Page navigation */}
          <div className="flex border rounded-md overflow-hidden">
            <Button
              onClick={previousPage}
              disabled={pageNumber <= 1}
              variant="ghost"
              size="sm"
              className="rounded-none h-7 px-2 text-muted-foreground border-r"
              aria-label="Previous page"
            >
              <Icon icon={faChevronLeft} className="size-3" />
            </Button>

            <div className="flex items-center border-r px-3 h-7 text-sm text-muted-foreground bg-background">
              <span>
                {pageNumber} of {numPages || 0}
              </span>
            </div>

            <Button
              onClick={nextPage}
              disabled={pageNumber >= numPages}
              variant="ghost"
              size="sm"
              className="rounded-none h-7 px-2 text-muted-foreground"
              aria-label="Next page"
            >
              <Icon icon={faChevronRight} className="size-3" />
            </Button>
          </div>
        </div>

        {/* Right side: Secondary controls */}
        <div className="flex items-center space-x-2">
          {/* Zoom controls */}
          <div className="flex border rounded-md overflow-hidden">
            <Button
              onClick={zoomOut}
              variant="ghost"
              size="sm"
              className="rounded-none h-7 px-2 text-muted-foreground border-r"
              aria-label="Zoom out"
            >
              <Icon icon={faMinus} className="size-3" />
            </Button>

            <div className="flex items-center border-r px-3 h-7 text-sm text-muted-foreground bg-background min-w-[4rem] justify-center">
              {Math.round(scale * 100)}%
            </div>

            <Button
              onClick={zoomIn}
              variant="ghost"
              size="sm"
              className="rounded-none h-7 px-2 text-muted-foreground"
              aria-label="Zoom in"
            >
              <Icon icon={faPlus} className="size-3" />
            </Button>
          </div>

          {/* Rotate control */}
          <Button
            onClick={() => rotate(90)}
            variant="outline"
            size="sm"
            className="h-7 px-2 text-muted-foreground"
            aria-label="Rotate document"
          >
            <Icon icon={faRotate} className="size-3" />
          </Button>

          {/* Search button */}
          <Button
            onClick={toggleSearch}
            variant="outline"
            size="sm"
            className={cn(
              "h-7 px-2 text-muted-foreground",
              showSearch
                ? "bg-muted-foreground/20 border-muted-foreground/20"
                : "",
            )}
            aria-label="Search document"
          >
            <Icon icon={faMagnifyingGlass} className="size-3" />
          </Button>

          {/* Search navigation buttons */}
          {showSearch && searchText && totalMatches > 0 && (
            <div className="flex items-center space-x-2">
              <Button
                onClick={() => navigateMatches("prev")}
                variant="outline"
                size="sm"
                className="h-7 px-2 text-muted-foreground"
                aria-label="Previous match"
              >
                <Icon icon={faChevronLeft} className="size-3" />
              </Button>
              <span className="text-sm text-muted-foreground">
                {currentMatchIndex + 1} of {totalMatches}
              </span>
              <Button
                onClick={() => navigateMatches("next")}
                variant="outline"
                size="sm"
                className="h-7 px-2 text-muted-foreground"
                aria-label="Next match"
              >
                <Icon icon={faChevronRight} className="size-3" />
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
