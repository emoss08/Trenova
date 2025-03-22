import "@react-pdf-viewer/core/lib/styles/index.css";
import "@react-pdf-viewer/full-screen/lib/styles/index.css";
import "@react-pdf-viewer/print/lib/styles/index.css";
import "@react-pdf-viewer/search/lib/styles/index.css";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import { RadioGroup, RadioGroupItem } from "../ui/radio-group";
import { PrintPages } from "@/types/pdf";

export function PDFPrintDialog({
  printDialogOpen,
  setPrintDialogOpen,
  printPages,
  handleValueChange,
  handlePrintAllPages,
  handleSetCustomPages,
  handlePrintPages,
  customPages,
  customPagesInvalid,
}: {
  printDialogOpen: boolean;
  setPrintDialogOpen: (open: boolean) => void;
  printPages: PrintPages;
  handleValueChange: (value: string) => void;
  handlePrintAllPages: () => void;
  handleSetCustomPages: (value: string) => void;
  handlePrintPages: () => void;
  customPages: string;
  customPagesInvalid: boolean;
}) {
  return (
    <Dialog open={printDialogOpen} onOpenChange={setPrintDialogOpen}>
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
              <Label htmlFor="customPages" className="text-sm font-medium">
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
  );
}
