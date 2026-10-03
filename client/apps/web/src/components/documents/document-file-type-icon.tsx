import { cn } from "@trenova/shared/lib/utils";
import {
  File04Icon,
  File06Icon,
  FileCode01Icon,
  FileSpreadsheetIcon,
  Image01Icon,
} from "@trenova/shared/components/icons";
import { getFileCategory } from "./document-utils";

type IconSize = "sm" | "md" | "lg" | "xl";

interface DocumentFileTypeIconProps {
  fileType?: string;
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

const categoryStyles: Record<string, { bg: string; text: string; icon: typeof File04Icon }> = {
  pdf: {
    bg: "bg-danger-subtle",
    text: "text-danger-foreground",
    icon: File06Icon,
  },
  image: {
    bg: "bg-accent-violet-subtle",
    text: "text-accent-violet-on-subtle",
    icon: Image01Icon,
  },
  spreadsheet: {
    bg: "bg-success-subtle",
    text: "text-success-foreground",
    icon: FileSpreadsheetIcon,
  },
  document: {
    bg: "bg-info-subtle",
    text: "text-info-foreground",
    icon: File06Icon,
  },
  data: {
    bg: "bg-warning-subtle",
    text: "text-warning-foreground",
    icon: FileCode01Icon,
  },
  default: {
    bg: "bg-muted",
    text: "text-muted-foreground",
    icon: File04Icon,
  },
};

export function DocumentFileTypeIcon({
  fileType = "",
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
