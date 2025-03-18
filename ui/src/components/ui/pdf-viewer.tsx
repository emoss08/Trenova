import { TableSheetProps } from "@/types/data-table";
import {
  faFilePdf,
  faMaximize,
  faPrint,
} from "@fortawesome/pro-solid-svg-icons";
import { SpecialZoomLevel, Viewer, Worker } from "@react-pdf-viewer/core";
import "@react-pdf-viewer/core/lib/styles/index.css";
import {
  fullScreenPlugin,
  RenderEnterFullScreenProps,
} from "@react-pdf-viewer/full-screen";
import "@react-pdf-viewer/full-screen/lib/styles/index.css";
import {
  getAllPagesNumbers,
  getCustomPagesNumbers,
  printPlugin,
} from "@react-pdf-viewer/print";
import "@react-pdf-viewer/print/lib/styles/index.css";
import { themePlugin } from "@react-pdf-viewer/theme";
import { useState } from "react";
import { useTheme } from "../theme-provider";
import { BetaTag } from "./beta-tag";
import { Button } from "./button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./dialog";
import { Icon } from "./icons";
import { Input } from "./input";
import { Label } from "./label";
import { RadioGroup, RadioGroupItem } from "./radio-group";
import { ScrollArea } from "./scroll-area";
import { VisuallyHidden } from "./visually-hidden";

enum PrintPages {
  All = "All",
  CustomPages = "CustomPages",
}

export function PDFViewer({ fileUrl }: { fileUrl: string }) {
  const { theme } = useTheme();
  const printPluginInstance = printPlugin();
  const fullScreenPluginInstance = fullScreenPlugin();
  const { EnterFullScreen } = fullScreenPluginInstance;
  const themePluginInstance = themePlugin();
  const { print, setPages } = printPluginInstance;
  const [printPages, setPrintPages] = useState<PrintPages>(PrintPages.All);
  const [customPages, setCustomPages] = useState<string>("");
  const [customPagesInvalid, setCustomPagesInvalid] = useState<boolean>(false);
  const [printDialogOpen, setPrintDialogOpen] = useState<boolean>(false);

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
    <div className="h-full w-full flex flex-col">
      <div className="p-3 mb-1 flex justify-between items-center">
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

        <div className="flex items-center gap-2">
          <Dialog open={printDialogOpen} onOpenChange={setPrintDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">
                <Icon icon={faPrint} className="size-4" />
                <span>Print</span>
              </Button>
            </DialogTrigger>
            <DialogContent withClose={false} className="sm:max-w-[425px]">
              <DialogHeader>
                <DialogTitle>Print PDF</DialogTitle>
              </DialogHeader>
              <DialogBody>
                <RadioGroup
                  defaultValue={printPages}
                  onValueChange={handleValueChange}
                >
                  <div className="flex flex-col gap-y-2">
                    <div className="flex items-center space-x-2">
                      <RadioGroupItem
                        onChange={handlePrintAllPages}
                        value={PrintPages.All}
                        id="r1"
                      />
                      <Label htmlFor="r1">All</Label>
                    </div>
                    <div className="flex items-center space-x-2">
                      <RadioGroupItem value={PrintPages.CustomPages} id="r2" />
                      <Label htmlFor="r2">Custom</Label>
                    </div>
                  </div>
                </RadioGroup>
                {printPages === PrintPages.CustomPages && (
                  <div className="mt-4">
                    <Label
                      htmlFor="customPages"
                      className="text-sm font-medium"
                    >
                      Page numbers (e.g., 1,3-5)
                    </Label>
                    <Input
                      id="customPages"
                      type="text"
                      className="mt-1"
                      value={customPages}
                      onChange={(e) => handleSetCustomPages(e.target.value)}
                    />
                    {customPagesInvalid && (
                      <p className="text-red-500 text-sm mt-1">
                        Please enter valid page numbers (e.g., 1,3-5)
                      </p>
                    )}
                  </div>
                )}
              </DialogBody>
              <DialogFooter className="mt-2 gap-2">
                <Button
                  variant="outline"
                  onClick={() => setPrintDialogOpen(!printDialogOpen)}
                >
                  Cancel
                </Button>
                <Button onClick={handlePrintPages}>Print</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <EnterFullScreen>
            {(props: RenderEnterFullScreenProps) => (
              <Button variant="outline" size="sm" onClick={props.onClick}>
                <Icon icon={faMaximize} className="size-4" />
                <span>Fullscreen</span>
              </Button>
            )}
          </EnterFullScreen>
        </div>
      </div>

      <div className="flex-1 border-t border-border">
        <Worker workerUrl="https://unpkg.com/pdfjs-dist@3.11.174/build/pdf.worker.min.js">
          <ScrollArea className="flex max-h-[80vh] flex-col overflow-y-auto rounded-b-lg">
            <Viewer
              fileUrl={fileUrl}
              theme={theme === "dark" ? "dark" : "light"}
              defaultScale={SpecialZoomLevel.PageFit}
              plugins={[
                printPluginInstance,
                fullScreenPluginInstance,
                themePluginInstance,
              ]}
            />
          </ScrollArea>
        </Worker>
      </div>
    </div>
  );
}

type PDFViewerDialogProps = {
  fileUrl: string;
} & TableSheetProps;

export function PDFViewerDialog({ fileUrl, ...props }: PDFViewerDialogProps) {
  return (
    <Dialog {...props}>
      <DialogContent
        withClose={false}
        className="max-h-[90vh] max-w-4xl p-0 overflow-hidden"
      >
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>PDF Viewer</DialogTitle>
            <DialogDescription>
              View the PDF file in the dialog.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <DialogBody className="p-0">
          <PDFViewer fileUrl={fileUrl} />
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
