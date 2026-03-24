import { cn } from "@/lib/utils";
import {
  FileIcon,
  FileSpreadsheetIcon,
  FileTextIcon,
  ImageIcon,
} from "lucide-react";
import { getFileCategory } from "./document-utils";

type IconSize = "sm" | "md" | "lg" | "xl";

interface DocumentFileTypeIconProps {
  fileType: string;
  fileName?: string;
  size?: IconSize;
  className?: string;
}

const sizeClasses: Record<IconSize, { container: string; icon: string }> = {
  sm: { container: "size-8", icon: "size-4" },
  md: { container: "size-10", icon: "size-5" },
  lg: { container: "size-12", icon: "size-6" },
  xl: { container: "size-16", icon: "size-8" },
};

const categoryStyles: Record<
  string,
  { bg: string; text: string; icon: typeof FileIcon }
> = {
  pdf: {
    bg: "bg-red-100 dark:bg-red-950/50",
    text: "text-red-600 dark:text-red-400",
    icon: FileTextIcon,
  },
  image: {
    bg: "bg-purple-100 dark:bg-purple-950/50",
    text: "text-purple-600 dark:text-purple-400",
    icon: ImageIcon,
  },
  spreadsheet: {
    bg: "bg-green-100 dark:bg-green-950/50",
    text: "text-green-600 dark:text-green-400",
    icon: FileSpreadsheetIcon,
  },
  document: {
    bg: "bg-blue-100 dark:bg-blue-950/50",
    text: "text-blue-600 dark:text-blue-400",
    icon: FileTextIcon,
  },
  default: {
    bg: "bg-muted",
    text: "text-muted-foreground",
    icon: FileIcon,
  },
};

export function DocumentFileTypeIcon({
  fileType,
  fileName,
  size = "md",
  className,
}: DocumentFileTypeIconProps) {
  const category = getFileCategory(fileType, fileName);
  const styles = categoryStyles[category] ?? categoryStyles.default;
  const sizes = sizeClasses[size];
  const Icon = styles.icon;

  return (
    <div
      className={cn(
        "flex shrink-0 items-center justify-center rounded-lg",
        styles.bg,
        sizes.container,
        className,
      )}
    >
      <Icon className={cn(styles.text, sizes.icon)} />
    </div>
  );
}
