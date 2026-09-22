import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, ClipboardIcon } from "lucide-react";

/**
 * A secret shown the one time it can be: a token that is stored only as a
 * hash, or a URL built from one. The warning says so, because a person who
 * closes the page without copying it has to rotate to get another.
 */
export function CopyableSecret({
  value,
  title,
  description,
  className,
}: {
  value: string;
  title: string;
  description: string;
  className?: string;
}) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <Alert variant="warning" className={cn("w-auto", className)}>
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>
        {description}
        <code className="bg-card text-foreground mt-1 block w-full rounded-md border p-2 font-mono text-xs break-all">
          {value}
        </code>
      </AlertDescription>
      <AlertAction>
        <Button size="sm" variant="outline" onClick={() => void copy(value, { withToast: true })}>
          {isCopied ? <CheckIcon /> : <ClipboardIcon />}
          {isCopied ? t("Copied") : t("Copy")}
        </Button>
      </AlertAction>
    </Alert>
  );
}
