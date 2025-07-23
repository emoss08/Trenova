/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { cn, formatFileSize } from "@/lib/utils";
import { type FileUploadCardProps } from "@/types/file-uploader";
import { faTrash } from "@fortawesome/pro-regular-svg-icons";
import { Badge } from "../ui/badge";
import { Icon } from "../ui/icons";
import { FileTypeCard } from "./file-type-card";

export function FileUploadCard({
  fileInfo,
  index,
  removeFile,
}: FileUploadCardProps) {
  return (
    <div
      key={`${fileInfo.file.name}-${index}`}
      className="p-2 border border-border rounded-md bg-background"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2 overflow-hidden">
          <div className="relative flex size-8 shrink-0 overflow-hidden rounded-sm">
            <FileTypeCard
              status={fileInfo.status}
              fileType={fileInfo.file.type}
            />
          </div>
          <div className="flex items-center justify-center gap-x-2">
            <span
              className="text-sm font-medium max-w-[150px] truncate"
              title={fileInfo.file.name}
            >
              {fileInfo.file.name}
            </span>
            <span className="text-xs text-muted-foreground">
              {fileInfo.fileSize || formatFileSize(fileInfo.file.size)}
            </span>

            {fileInfo.status === "error" && (
              <Badge
                withDot={false}
                variant="inactive"
                className="text-2xs font-normal"
              >
                Failed
              </Badge>
            )}

            {fileInfo.status === "success" && (
              <Badge
                withDot={false}
                variant="active"
                className="text-2xs font-normal"
              >
                Uploaded
              </Badge>
            )}
          </div>
        </div>

        <div className="flex items-center space-x-2">
          <Button
            title="Remove file"
            variant="ghost"
            size="icon"
            onClick={(e) => {
              e.stopPropagation();
              removeFile(index);
            }}
          >
            <Icon icon={faTrash} className="size-4" />
          </Button>
        </div>
      </div>
      {/* TODO(Wolfred): Add dropdown for document classification */}

      <div className="flex items-center gap-2">
        <Progress
          value={fileInfo.progress}
          className="h-2"
          indicatorClassName={cn(
            fileInfo.status === "error"
              ? "bg-red-500"
              : fileInfo.status === "success" && "bg-green-500",
          )}
        />
        <span
          className={cn(
            "text-xs",
            fileInfo.status === "error" ? "text-red-500" : "text-foreground",
          )}
        >
          {`${fileInfo.progress}%`}
        </span>
      </div>
    </div>
  );
}
