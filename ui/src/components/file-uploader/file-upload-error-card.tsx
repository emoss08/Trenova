/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { type FileUploadErrorCardProps } from "@/types/file-uploader";
import { faExclamationTriangle } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "../ui/icons";

export function FileUploadErrorCard({
  errors,
  onOpenChange,
}: FileUploadErrorCardProps) {
  return (
    <div className="mt-4 p-3 bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800 rounded-md">
      <div className="flex items-start justify-between">
        <div className="flex items-start gap-2">
          <Icon
            icon={faExclamationTriangle}
            className="text-red-500 size-5 mt-0.5"
          />
          <div>
            <p className="font-medium text-red-700 dark:text-red-400 text-sm">
              {errors.length === 1
                ? "1 error occurred during upload"
                : `${errors.length} errors occurred during upload`}
            </p>
            <p className="text-xs text-red-600 dark:text-red-300">
              Click to view error details
            </p>
          </div>
        </div>
        <Button
          size="sm"
          variant="outline"
          onClick={() => onOpenChange(true)}
          className="text-red-600 border-red-300 hover:bg-red-50 dark:text-red-400 dark:border-red-700 dark:hover:bg-red-950/70"
        >
          View Errors
        </Button>
      </div>
    </div>
  );
}
