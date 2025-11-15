import { Icon } from "@/components/ui/icons";
import { NotificationSchema } from "@/lib/schemas/notification-schema";
import { ReportFormatSchema } from "@/lib/schemas/report-schema";
import { cn, formatFileSize, getFileClass } from "@/lib/utils";
import {
  faFile,
  faFileCsv,
  faFileExcel,
} from "@fortawesome/pro-regular-svg-icons";
import { DownloadIcon } from "@radix-ui/react-icons";

const formatToIcon = (format: ReportFormatSchema) => {
  switch (format) {
    case ReportFormatSchema.enum.Csv:
      return faFileCsv;
    case ReportFormatSchema.enum.Excel:
      return faFileExcel;
    default:
      return faFile;
  }
};

export function DownloadReportNotificationItem({
  notification,
}: {
  notification: NotificationSchema;
}) {
  const ReportIcon = formatToIcon(notification.data.reportFormat);
  const fileClass = getFileClass(notification.data.reportFileName);

  return (
    <button
      className={cn(
        "flex w-full cursor-pointer flex-row items-center justify-start gap-2 rounded-md border border-transparent bg-muted/70 p-2 hover:border-border hover:bg-muted",
        fileClass.bgColor,
      )}
    >
      <Icon icon={ReportIcon} className={cn("size-4", fileClass.iconColor)} />
      <span className="flex w-full flex-col items-start justify-start">
        <span className="flex w-full items-center justify-between">
          <span className="max-w-[300px] truncate text-xs font-medium">
            {notification.data.reportFileName}
          </span>
          <span className="flex items-center justify-center">
            <DownloadIcon className="size-3" />
          </span>
        </span>
        <p className="text-xs text-muted-foreground">
          {formatFileSize(notification.data.reportSize as number)}
        </p>
      </span>
    </button>
  );
}
