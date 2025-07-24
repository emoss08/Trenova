/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
import { upperFirst } from "@/lib/utils";
import { type FileUploadErrorDialogProps } from "@/types/file-uploader";

export function FileUploadErrorDialog({
  errorsByType,
  clearErrors,
  ...props
}: FileUploadErrorDialogProps) {
  return (
    <Dialog {...props}>
      <DialogContent className="sm:max-w-lg">
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
              <div
                key={errorType}
                className="space-y-2 bg-red-500/10 p-2 rounded-md border border-red-500/20"
              >
                <div className="space-y-2">
                  <div className="flex items-center justify-between gap-2 border-b border-red-500/20 pb-2">
                    <div className="flex items-center gap-2">
                      <div className="size-2 bg-red-500 rounded-full" />
                      <div className="text-sm font-medium">{errorType}</div>
                    </div>
                    <div className="text-xs text-red-500">
                      {errors.length} {errors.length === 1 ? "file" : "files"}
                    </div>
                  </div>
                  {errors.map((error, index) => (
                    <div key={`${error.fileName}-${index}`} className="text-sm">
                      <div className="font-medium">{error.fileName}</div>
                      {error.details && (
                        <p className="text-pretty text-xs font-mono overflow-x-auto text-red-500">
                          {upperFirst(error.details)}
                        </p>
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
