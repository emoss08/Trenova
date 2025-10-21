/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import type { FileUploadDockProps } from "@/types/file-uploader";
import {
  faCircleExclamation,
  faUpload,
} from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "../ui/icons";
import { PulsatingDots } from "../ui/pulsating-dots";

export function FileUploadDock({
  fileStats,
  isDirty,
  isSubmitting,
  onCancel,
  onUpload,
}: FileUploadDockProps) {
  if (!isDirty && fileStats.pending === 0) {
    return null;
  }

  return (
    <>
      <div className="fixed bottom-6 z-50 left-1/2 transform -translate-x-1/2 w-[450px]">
        <div className="bg-foreground rounded-lg p-2 shadow-lg flex items-center gap-x-10">
          <div className="flex items-center gap-x-3">
            <Icon
              icon={faCircleExclamation}
              className="text-amber-400 bg-amber-400/10 dark:text-amber-600 rounded-full"
            />
            <div className="flex flex-col">
              <span className="text-sm font-medium text-background">
                {fileStats.pending === 1
                  ? "1 file ready to upload"
                  : `${fileStats.pending} files ready to upload`}
              </span>
              <span className="text-2xs text-background/80">
                {fileStats.total === fileStats.pending
                  ? "Please upload or cancel to discard."
                  : `${fileStats.success} uploaded, ${fileStats.error} failed`}
              </span>
            </div>
          </div>
          <div className="ml-auto flex items-center space-x-2">
            <Button
              variant="ghost"
              onClick={onCancel}
              className="text-background hover:text-background/80 hover:bg-foreground/90"
            >
              Cancel
            </Button>
            <Button
              onClick={onUpload}
              disabled={fileStats.pending === 0 || isSubmitting}
              className="bg-background text-foreground hover:bg-background/80 min-w-20"
            >
              {isSubmitting ? (
                <PulsatingDots size={1} color="foreground" />
              ) : (
                <>
                  <Icon icon={faUpload} className="size-4" />
                  Upload
                </>
              )}
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}
