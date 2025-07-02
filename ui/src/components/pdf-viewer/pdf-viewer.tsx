import { useResizeObserver } from "@wojtekmaj/react-hooks";
import type { PDFDocumentProxy } from "pdfjs-dist";
import { useCallback, useEffect, useRef, useState } from "react";
import { pdfjs } from "react-pdf";
import { PDFDocumentOutline } from "./pdf-document-outline";
import { PDFDocumentViewer } from "./pdf-document-viewer";
import { PDFFooter } from "./pdf-footer";
import { PDFNavigationBar, PDFSearchBar } from "./pdf-toolbar";
import { PDFViewerInner } from "./pdf-viewer-inner";
import type { PDFFile } from "./types";

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "pdfjs-dist/build/pdf.worker.min.mjs",
  import.meta.url,
).toString();

const options = {
  cMapUrl: "/cmaps/",
  standardFontDataUrl: "/standard_fonts/",
};

const resizeObserverOptions = {};

interface PDFViewerProps {
  fileUrl: PDFFile;
  className?: string;
}

/**
 * Highlights search text in the PDF document
 * @param text Text to highlight
 * @param pattern Search pattern
 * @returns Text with highlighted pattern
 */
function highlightPattern(text: string, pattern: string | RegExp): string {
  if (!pattern || pattern === "") return text;

  try {
    // Create a case-insensitive RegExp if the pattern is a string
    const regExp =
      typeof pattern === "string"
        ? new RegExp(
            `(${pattern.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`,
            "gi",
          )
        : pattern;

    return text.replace(regExp, "<mark>$1</mark>");
  } catch (error) {
    console.error("Error highlighting pattern:", error);
    return text;
  }
}

export default function PDFViewer({ fileUrl, className = "" }: PDFViewerProps) {
  const [numPages, setNumPages] = useState<number>(0);
  const [pageNumber, setPageNumber] = useState<number>(1);
  const [containerRef, setContainerRef] = useState<HTMLElement | null>(null);
  const [containerWidth, setContainerWidth] = useState<number>();
  const [scale, setScale] = useState<number>(0.6);
  const [rotation, setRotation] = useState<number>(0);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [showOutline, setShowOutline] = useState<boolean>(false);
  const [hasOutline, setHasOutline] = useState<boolean>(false);
  const viewerRef = useRef<HTMLDivElement>(null);
  const outlineRef = useRef<HTMLDivElement>(null);

  // Search state
  const [searchText, setSearchText] = useState<string>("");
  const [showSearch, setShowSearch] = useState<boolean>(false);
  const [currentMatchIndex, setCurrentMatchIndex] = useState<number>(0);
  const [totalMatches, setTotalMatches] = useState<number>(0);

  const onResize = useCallback<ResizeObserverCallback>((entries) => {
    const [entry] = entries;

    if (entry) {
      setContainerWidth(entry.contentRect.width);
    }
  }, []);

  useResizeObserver(containerRef, resizeObserverOptions, onResize);

  function onDocumentLoadSuccess(pdf: PDFDocumentProxy) {
    setNumPages(pdf.numPages);
    setIsLoading(false);

    // Check if the document has an outline
    pdf
      .getOutline()
      .then((outline) => {
        setHasOutline(outline !== null && outline.length > 0);
      })
      .catch(() => {
        setHasOutline(false);
      });
  }

  function onDocumentLoadError() {
    setIsLoading(false);
  }

  const changePage = useCallback(
    (offset: number) => {
      setPageNumber((prevPageNumber) => {
        const newPageNumber = prevPageNumber + offset;
        return Math.max(1, Math.min(numPages, newPageNumber));
      });
    },
    [numPages],
  );

  const previousPage = useCallback(() => changePage(-1), [changePage]);
  const nextPage = useCallback(() => changePage(1), [changePage]);

  const goToPage = useCallback(
    (pageNum: number) => {
      const page = Math.max(1, Math.min(numPages, pageNum));
      setPageNumber(page);
    },
    [numPages],
  );

  const zoomIn = () => setScale((prevScale) => Math.min(3, prevScale + 0.1));
  const zoomOut = () => setScale((prevScale) => Math.max(0.5, prevScale - 0.1));
  const resetZoom = () => setScale(1);

  const rotate = (angle: number) => {
    setRotation((prevRotation) => (prevRotation + angle) % 360);
  };

  // Handler for when an outline item is clicked
  const onItemClick = useCallback((data: { pageNumber: number }) => {
    // Navigate to the page
    setPageNumber(data.pageNumber);

    // Close the outline
    setShowOutline(false);

    // Scroll to top of viewer
    if (viewerRef.current) {
      viewerRef.current.scrollTop = 0;
    }
  }, []);

  // Handler for outline load success
  const onOutlineLoadSuccess = useCallback((outline: unknown) => {
    setHasOutline(!!outline && Array.isArray(outline) && outline.length > 0);
  }, []);

  // Handler for outline load error
  const onOutlineLoadError = useCallback((error: Error) => {
    console.error("Error loading outline:", error);
    setHasOutline(false);
  }, []);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      switch (e.key) {
        case "ArrowRight":
        case "ArrowDown":
          nextPage();
          break;
        case "ArrowLeft":
        case "ArrowUp":
          previousPage();
          break;
        case "Home":
          goToPage(1);
          break;
        case "End":
          goToPage(numPages);
          break;
        case "+":
        case "=":
          zoomIn();
          break;
        case "-":
          zoomOut();
          break;
        case "0":
          resetZoom();
          break;
        case "r":
          rotate(90);
          break;
        case "f":
          if (e.ctrlKey || e.metaKey) {
            e.preventDefault();
            setShowSearch(true);
          }
          break;
        case "Escape":
          if (showSearch) {
            setShowSearch(false);
            setSearchText("");
          }
          break;
      }
    },
    [numPages, goToPage, nextPage, previousPage, showSearch],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [handleKeyDown]);

  // Text renderer for highlighting search text
  const textRenderer = useCallback(
    (textItem: { str: string }) => highlightPattern(textItem.str, searchText),
    [searchText],
  );

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchText(e.target.value);
    // Reset match tracking when search changes
    setCurrentMatchIndex(0);
    setTotalMatches(0);
  };

  // Function to count matches in text content
  const updateMatchCount = useCallback(
    (textContent: string) => {
      if (!searchText) {
        setTotalMatches(0);
        return;
      }

      try {
        const regExp = new RegExp(
          searchText.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
          "gi",
        );
        const matches = textContent.match(regExp);
        setTotalMatches(matches ? matches.length : 0);
      } catch (error) {
        console.error("Error counting matches:", error);
        setTotalMatches(0);
      }
    },
    [searchText],
  );

  // Function to navigate between matches
  const navigateMatches = useCallback(
    (direction: "next" | "prev") => {
      if (totalMatches === 0) return;

      if (direction === "next") {
        setCurrentMatchIndex((prev) => (prev + 1) % totalMatches);
      } else {
        setCurrentMatchIndex(
          (prev) => (prev - 1 + totalMatches) % totalMatches,
        );
      }

      // Find and scroll to the match
      const marks = document.getElementsByTagName("mark");
      if (marks.length > 0) {
        const nextIndex =
          direction === "next"
            ? (currentMatchIndex + 1) % totalMatches
            : (currentMatchIndex - 1 + totalMatches) % totalMatches;
        marks[nextIndex]?.scrollIntoView({
          behavior: "smooth",
          block: "center",
        });
      }
    },
    [totalMatches, currentMatchIndex],
  );

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      if (e.shiftKey) {
        navigateMatches("prev");
      } else {
        navigateMatches("next");
      }
    }
  };

  // Update match count when page content changes
  useEffect(() => {
    const updateMatchesFromPage = () => {
      const textLayer = document.querySelector(".react-pdf__Page__textContent");
      if (textLayer) {
        updateMatchCount(textLayer.textContent || "");
      }
    };

    // Small delay to ensure text layer is rendered
    const timeoutId = setTimeout(updateMatchesFromPage, 100);
    return () => clearTimeout(timeoutId);
  }, [searchText, pageNumber, updateMatchCount]);

  const toggleSearch = () => {
    setShowSearch(!showSearch);
    if (!showSearch) {
      // Focus the search input when opened
      setTimeout(() => {
        const searchInput = document.getElementById("pdf-search-input");
        if (searchInput) {
          searchInput.focus();
        }
      }, 100);
    } else {
      // Clear search when closing
      setSearchText("");
    }
  };

  return (
    <PDFViewerInner className={className} setContainerRef={setContainerRef}>
      {showSearch && (
        <PDFSearchBar
          searchText={searchText}
          handleSearchChange={handleSearchChange}
          handleInputKeyDown={handleInputKeyDown}
        />
      )}

      {/* Toolbar */}
      <PDFNavigationBar
        showOutline={showOutline}
        setShowOutline={setShowOutline}
        hasOutline={hasOutline}
        previousPage={previousPage}
        nextPage={nextPage}
        pageNumber={pageNumber}
        numPages={numPages}
        zoomOut={zoomOut}
        zoomIn={zoomIn}
        rotate={rotate}
        toggleSearch={toggleSearch}
        searchText={searchText}
        showSearch={showSearch}
        totalMatches={totalMatches}
        currentMatchIndex={currentMatchIndex}
        navigateMatches={navigateMatches}
        scale={scale}
      />

      {/* Document viewer area with optional outline */}
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar for outline when shown */}
        {showOutline && (
          <PDFDocumentOutline
            setShowOutline={setShowOutline}
            hasOutline={hasOutline}
          />
        )}

        {/* PDF Document */}
        <PDFDocumentViewer
          viewerRef={viewerRef}
          showOutline={showOutline}
          hasOutline={hasOutline}
          isLoading={isLoading}
          numPages={numPages}
          onDocumentLoadSuccess={onDocumentLoadSuccess}
          onDocumentLoadError={onDocumentLoadError}
          onItemClick={onItemClick}
          onOutlineLoadSuccess={onOutlineLoadSuccess}
          onOutlineLoadError={onOutlineLoadError}
          outlineRef={outlineRef}
          pageNumber={pageNumber}
          scale={scale}
          rotation={rotation}
          containerWidth={containerWidth}
          fileUrl={fileUrl}
          options={options}
          searchText={searchText}
          textRenderer={textRenderer}
        />
      </div>

      {/* Bottom toolbar/status bar - now sticky at the bottom */}
      <PDFFooter
        numPages={numPages}
        pageNumber={pageNumber}
        scale={scale}
        rotation={rotation}
        searchText={searchText}
      />
    </PDFViewerInner>
  );
}
