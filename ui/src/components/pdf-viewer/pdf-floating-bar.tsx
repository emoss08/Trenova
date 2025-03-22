import { useDraggable } from "@dnd-kit/core";
import {
  faDownload,
  faInfo,
  faMaximize,
  faPrint,
  faSearch,
} from "@fortawesome/pro-solid-svg-icons";
import { DragHandleDots2Icon } from "@radix-ui/react-icons";
import "@react-pdf-viewer/core/lib/styles/index.css";
import {
  EnterFullScreenProps,
  RenderEnterFullScreenProps,
} from "@react-pdf-viewer/full-screen";
import "@react-pdf-viewer/full-screen/lib/styles/index.css";
import { DownloadProps, RenderDownloadProps } from "@react-pdf-viewer/get-file";
import "@react-pdf-viewer/print/lib/styles/index.css";
import {
  RenderShowPropertiesProps,
  ShowPropertiesProps,
} from "@react-pdf-viewer/properties";
import "@react-pdf-viewer/properties/lib/styles/index.css";
import { RenderSearchProps, SearchProps } from "@react-pdf-viewer/search";
import "@react-pdf-viewer/search/lib/styles/index.css";
import React from "react";
import { Button } from "../ui/button";
import { Icon } from "../ui/icons";
import { Input } from "../ui/input";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";

export function PDFFloatingBar({
  EnterFullScreen,
  Download,
  Search,
  ShowProperties,
  setPrintDialogOpen,
  printDialogOpen,
  position,
  ref,
}: {
  EnterFullScreen: (props: EnterFullScreenProps) => React.ReactNode;
  Download: (props: DownloadProps) => React.ReactNode;
  Search: (props: SearchProps) => React.ReactNode;
  ShowProperties: (props: ShowPropertiesProps) => React.ReactNode;
  setPrintDialogOpen: (open: boolean) => void;
  printDialogOpen: boolean;
  position: { x: number; y: number };
  ref: React.Ref<HTMLDivElement>;
}) {
  // Setup draggable functionality
  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: "pdf-floating-bar",
  });

  // Style for positioning the floating bar
  const style = {
    position: "fixed" as const,
    top: position.y,
    left: position.x,
    zIndex: 50,
    transform: transform
      ? `translate3d(${transform.x}px, ${transform.y}px, 0)`
      : undefined,
  };

  return (
    <TooltipProvider>
      <div
        id="pdf-floating-bar"
        ref={(node) => {
          // Set both refs - the one from useDraggable and the forwarded ref
          setNodeRef(node);
          if (typeof ref === "function") {
            ref(node);
          } else if (ref) {
            ref.current = node;
          }
        }}
        style={style}
        className="bg-background border border-border rounded-md shadow-lg py-1.5 px-2 flex items-center gap-2"
      >
        {/* Drag handle */}
        <div
          className="w-5 h-5 flex items-center justify-center cursor-move"
          {...listeners}
          {...attributes}
        >
          <DragHandleDots2Icon className="size-4" />
        </div>
        <div className="flex-1">
          <Search>
            {(renderSearchProps: RenderSearchProps) => (
              <Input
                icon={
                  <Icon
                    icon={faSearch}
                    className="size-3 text-muted-foreground"
                  />
                }
                type="text"
                placeholder="Search..."
                className="w-30 h-7"
                value={renderSearchProps.keyword}
                onChange={(e) => {
                  if (e.target.value === "") {
                    renderSearchProps.clearKeyword();
                  } else {
                    renderSearchProps.setKeyword(e.target.value);
                  }
                }}
                onKeyDown={(e) => {
                  if (e.keyCode === 13 && renderSearchProps.keyword) {
                    renderSearchProps.search();
                  }
                }}
              />
            )}
          </Search>
        </div>

        <div className="w-px h-6 bg-border mx-1" />
        <Tooltip>
          <EnterFullScreen>
            {(props: RenderEnterFullScreenProps) => (
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={props.onClick}
                  title="Fullscreen"
                >
                  <Icon icon={faMaximize} className="size-4" />
                </Button>
              </TooltipTrigger>
            )}
          </EnterFullScreen>
          <TooltipContent>
            <p>Fullscreen</p>
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setPrintDialogOpen(!printDialogOpen)}
              title="Print"
            >
              <Icon icon={faPrint} className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <p>Print</p>
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <Download>
            {(props: RenderDownloadProps) => (
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  title="Download"
                  onClick={props.onClick}
                >
                  <Icon icon={faDownload} className="size-4" />
                </Button>
              </TooltipTrigger>
            )}
          </Download>
          <TooltipContent>
            <p>Download</p>
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <ShowProperties>
            {(props: RenderShowPropertiesProps) => (
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  title="properties"
                  onClick={props.onClick}
                >
                  <Icon icon={faInfo} className="size-4" />
                </Button>
              </TooltipTrigger>
            )}
          </ShowProperties>
          <TooltipContent>
            <p>Properties</p>
          </TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  );
}
