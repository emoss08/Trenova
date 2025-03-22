import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { type FileUploadErrorDialogProps } from "@/types/file-uploader";

export function FileUploadErrorDialog({
  errorsByType,
  clearErrors,
  ...props
}: FileUploadErrorDialogProps) {
  return (
    <Dialog {...props}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Upload Errors
          </DialogTitle>
          <DialogDescription>
            The following errors occurred during file upload.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <div className="space-y-4 pr-4">
            {Object.entries(errorsByType).map(([errorType, errors]) => (
              <div key={errorType} className="space-y-2">
                <div className="space-y-1">
                  {errors.map((error, index) => (
                    <div key={`${error.fileName}-${index}`} className="text-sm">
                      <div className="font-medium">{error.fileName}</div>
                      {error.details && (
                        <div className="text-muted-foreground text-xs">
                          {error.details}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={clearErrors}>
            Dismiss
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
