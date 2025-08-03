/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { cn, getFileClass, getFileIcon } from "@/lib/utils";
import { FileStatus } from "@/types/file-uploader";
import { faTimesCircle } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "../ui/icons";

export function FileTypeCard({
  status,
  fileType,
}: {
  status: FileStatus;
  fileType: string;
}) {
  return (
    <div
      className={cn(
        "bg-muted border flex shrink-0 size-7 items-center justify-center rounded-sm",
        status === "error"
          ? "bg-red-50 dark:bg-red-950/50 "
          : getFileClass(fileType).bgColor,
        status === "error"
          ? "border-red-500 dark:border-red-800"
          : getFileClass(fileType).borderColor,
      )}
    >
      <Icon
        icon={status === "error" ? faTimesCircle : getFileIcon(fileType)}
        className={cn(
          "size-4",
          status === "error"
            ? "text-red-500"
            : getFileClass(fileType).iconColor,
        )}
      />
    </div>
  );
}
