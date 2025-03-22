import { PDF_STORAGE_KEY } from "@/constants/env";
import { PrintPages } from "@/types/pdf";
import { DndContext, DragEndEvent } from "@dnd-kit/core";
import { faFilePdf } from "@fortawesome/pro-solid-svg-icons";
import {
  OpenFile,
  SpecialZoomLevel,
  Viewer,
  Worker,
} from "@react-pdf-viewer/core";
import "@react-pdf-viewer/core/lib/styles/index.css";
import { fullScreenPlugin } from "@react-pdf-viewer/full-screen";
import "@react-pdf-viewer/full-screen/lib/styles/index.css";
import { getFilePlugin } from "@react-pdf-viewer/get-file";
import {
  getAllPagesNumbers,
  getCustomPagesNumbers,
  printPlugin,
} from "@react-pdf-viewer/print";
import "@react-pdf-viewer/print/lib/styles/index.css";
import { propertiesPlugin } from "@react-pdf-viewer/properties";
import { searchPlugin } from "@react-pdf-viewer/search";
import "@react-pdf-viewer/search/lib/styles/index.css";
import { themePlugin } from "@react-pdf-viewer/theme";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTheme } from "../theme-provider";
import { BetaTag } from "../ui/beta-tag";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Icon } from "../ui/icons";
import { Progress } from "../ui/progress";
import { ScrollArea } from "../ui/scroll-area";
import { PDFFloatingBar } from "./pdf-floating-bar";
import { PDFPrintDialog } from "./pdf-print-dialog";

export default function PDFViewer({ fileUrl }: { fileUrl: string }) {
  const { theme } = useTheme();

  // * Plugins
  const fullScreenPluginInstance = fullScreenPlugin();
  const { EnterFullScreen } = fullScreenPluginInstance;
  const themePluginInstance = themePlugin();
  const getFilePluginInstance = getFilePlugin({
    fileNameGenerator: (file: OpenFile) => {
      // `file.name` is the URL of opened file
      const fileName = file.name.substring(file.name.lastIndexOf("/") + 1);
      return `a-copy-of-${fileName}`;
    },
  });
  const { Download } = getFilePluginInstance;
  const searchPluginInstance = searchPlugin();
  const { Search } = searchPluginInstance;
  const propertiesPluginInstance = propertiesPlugin();
  const { ShowProperties } = propertiesPluginInstance;

  // Reference to the floating bar element
  const floatingBarRef = useRef<HTMLDivElement | null>(null);

  // Utility functions for storage operations
  const savePositionToStorage = useCallback(
    (position: { x: number; y: number }) => {
      try {
        // Ensure we're explicitly setting x and y properties to avoid any reference issues
        const positionData = {
          x: Number(position.x),
          y: Number(position.y),
        };

        const positionJson = JSON.stringify(positionData);
        localStorage.setItem(PDF_STORAGE_KEY, positionJson);
        console.log(
          "🔄 Floating bar position saved:",
          positionData,
          "JSON:",
          positionJson,
        );
      } catch (error) {
        console.error("❌ Failed to save floating bar position:", error);
      }
    },
    [],
  );

  const getPositionFromStorage = useCallback(() => {
    try {
      const positionJson = localStorage.getItem(PDF_STORAGE_KEY);
      console.log("📋 Raw localStorage value:", positionJson);

      if (!positionJson) return null;

      const position = JSON.parse(positionJson);

      // Validate the loaded data has numeric x and y properties
      if (typeof position.x === "number" && typeof position.y === "number") {
        console.log("✅ Loaded valid floating bar position:", position);
        return position;
      } else {
        console.warn(
          "⚠️ Invalid floating bar position data structure:",
          position,
        );
        return null;
      }
    } catch (error) {
      console.error("❌ Error loading floating bar position:", error);
      return null;
    }
  }, []);

  // * State
  const [printPages, setPrintPages] = useState<PrintPages>(PrintPages.All);
  const [customPages, setCustomPages] = useState<string>("");
  const [customPagesInvalid, setCustomPagesInvalid] = useState<boolean>(false);
  const [printDialogOpen, setPrintDialogOpen] = useState<boolean>(false);

  // Set initial position from localStorage or default to bottom center
  const [floatingBarPosition, setFloatingBarPosition] = useState(() => {
    // Try to get saved position from localStorage
    const savedPosition = getPositionFromStorage();

    if (savedPosition) {
      return savedPosition;
    }

    // Default position if nothing found in localStorage
    return {
      x: window.innerWidth / 2 - 150, // Approximate center
      y: window.innerHeight - 100, // Near the bottom of the screen
    };
  });

  // Position the floating bar initially
  useEffect(() => {
    if (floatingBarRef.current) {
      // If position was loaded from localStorage, validate it's still within viewport
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;
      const barWidth = floatingBarRef.current.offsetWidth;
      const barHeight = floatingBarRef.current.offsetHeight;

      const newPosition = { ...floatingBarPosition };
      let needsUpdate = false;

      // Check if the bar is outside viewport boundaries and adjust if needed
      if (newPosition.x < 0 || newPosition.x > viewportWidth - barWidth) {
        newPosition.x = Math.max(0, viewportWidth / 2 - barWidth / 2);
        needsUpdate = true;
      }

      if (newPosition.y < 0 || newPosition.y > viewportHeight - barHeight) {
        newPosition.y = viewportHeight - 100; // 100px from the bottom
        needsUpdate = true;
      }

      // Update position if needed
      if (needsUpdate) {
        setFloatingBarPosition(newPosition);
        savePositionToStorage(newPosition);
      }
    }
  }, [savePositionToStorage, floatingBarPosition]);

  // Effect to save position whenever it changes
  useEffect(() => {
    savePositionToStorage(floatingBarPosition);
  }, [floatingBarPosition, savePositionToStorage]);

  // Handle drag end event to update position
  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { delta } = event;

    // Update position with constraints to keep within viewport
    setFloatingBarPosition((prev: { x: number; y: number }) => {
      // Get the viewport dimensions
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      // Get the floating bar element to calculate its dimensions
      const floatingBar = document.getElementById("pdf-floating-bar");
      const barWidth = floatingBar?.offsetWidth || 0;
      const barHeight = floatingBar?.offsetHeight || 0;

      // Calculate new position with constraints
      const newX = Math.max(
        0,
        Math.min(prev.x + delta.x, viewportWidth - barWidth),
      );
      const newY = Math.max(
        0,
        Math.min(prev.y + delta.y, viewportHeight - barHeight),
      );

      return { x: newX, y: newY };
    });
  }, []);

  // Handle window resize to ensure the floating bar stays in bounds
  useEffect(() => {
    const handleResize = () => {
      if (floatingBarRef.current) {
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;
        const barWidth = floatingBarRef.current.offsetWidth;
        const barHeight = floatingBarRef.current.offsetHeight;

        setFloatingBarPosition((prev: { x: number; y: number }) => {
          // Adjust position if needed due to viewport size change
          const newX = Math.max(0, Math.min(prev.x, viewportWidth - barWidth));
          const newY = Math.max(
            0,
            Math.min(prev.y, viewportHeight - barHeight),
          );

          return { x: newX, y: newY };
        });
      }
    };

    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, [savePositionToStorage]);

  const renderProgressBar = useCallback(
    (numLoadedPages: number, numPages: number, onCancel: () => void) => (
      <Dialog open={true} onOpenChange={() => {}}>
        <DialogContent withClose={false}>
          <DialogHeader>
            <DialogTitle>Printing...</DialogTitle>
          </DialogHeader>
          <DialogBody>
            <div className="flex flex-col space-y-3">
              <p className="text-sm text-muted-foreground">
                Preparing {numLoadedPages}/{numPages} pages ...
              </p>
              <Progress value={(numLoadedPages / numPages) * 100} />
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    ),
    [],
  );

  const printPluginInstance = printPlugin({
    renderProgressBar,
  });
  const { print, setPages } = printPluginInstance;

  const handlePrintAllPages = () => {
    setPrintPages(PrintPages.All);
    setPages(getAllPagesNumbers);
  };

  const handleSetCustomPages = (value: string) => {
    setCustomPages(value);
    if (!/^([0-9,-\s])+$/.test(value)) {
      setCustomPagesInvalid(true);
    } else {
      setCustomPagesInvalid(false);
      setPages(getCustomPagesNumbers(value));
    }
  };

  const handlePrintPages = () => {
    if (printPages === PrintPages.All) {
      setPages(getAllPagesNumbers);
    } else if (printPages === PrintPages.CustomPages && !customPagesInvalid) {
      setPages(getCustomPagesNumbers(customPages));
    }
    print();
  };

  const handleValueChange = (value: string) => {
    const newPrintPages = value as PrintPages;
    setPrintPages(newPrintPages);

    if (newPrintPages === PrintPages.All) {
      setPages(getAllPagesNumbers);
    } else if (
      newPrintPages === PrintPages.CustomPages &&
      !customPagesInvalid &&
      customPages
    ) {
      setPages(getCustomPagesNumbers(customPages));
    }
  };

  return (
    <>
      <div className="h-full w-full flex flex-col relative">
        <div className="p-2 flex justify-between items-center">
          <div className="flex items-center gap-2">
            <Icon icon={faFilePdf} className="size-4 text-foreground" />
            <div className="flex flex-col">
              <span className="font-semibold leading-none tracking-tight flex items-center gap-x-1">
                Document Viewer
                <BetaTag />
              </span>
              <span className="text-2xs text-muted-foreground font-normal">
                View and print PDF documents.
              </span>
            </div>
          </div>
        </div>

        <div className="flex-1 border-t border-border">
          <Worker workerUrl="https://unpkg.com/pdfjs-dist@3.11.174/build/pdf.worker.min.js">
            <ScrollArea className="flex max-h-[80vh] flex-col overflow-y-auto rounded-b-lg p-2">
              <Viewer
                fileUrl={fileUrl}
                theme={theme}
                defaultScale={SpecialZoomLevel.PageFit}
                plugins={[
                  getFilePluginInstance,
                  printPluginInstance,
                  fullScreenPluginInstance,
                  themePluginInstance,
                  searchPluginInstance,
                  propertiesPluginInstance,
                ]}
                renderLoader={(percentages: number) => (
                  <div className="flex flex-col items-center justify-center h-full">
                    <Progress value={percentages} />
                  </div>
                )}
              />
            </ScrollArea>
          </Worker>
        </div>

        <DndContext onDragEnd={handleDragEnd}>
          <PDFFloatingBar
            ShowProperties={ShowProperties}
            ref={floatingBarRef}
            EnterFullScreen={EnterFullScreen}
            Download={Download}
            Search={Search}
            setPrintDialogOpen={setPrintDialogOpen}
            printDialogOpen={printDialogOpen}
            position={floatingBarPosition}
          />
        </DndContext>
      </div>
      {printDialogOpen && (
        <PDFPrintDialog
          printDialogOpen={printDialogOpen}
          setPrintDialogOpen={setPrintDialogOpen}
          printPages={printPages}
          handleValueChange={handleValueChange}
          handlePrintAllPages={handlePrintAllPages}
          handleSetCustomPages={handleSetCustomPages}
          handlePrintPages={handlePrintPages}
          customPages={customPages}
          customPagesInvalid={customPagesInvalid}
        />
      )}
    </>
  );
}
