import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { faCheck, faClipboard } from "@fortawesome/pro-regular-svg-icons";
import React from "react";
import { Button } from "./button";
import { Icon } from "./icons";
import { Label } from "./label";

export function CopyText({ label, value }: { label: string; value: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="group flex flex-col gap-1">
      <Label>{label}</Label>
      <div className="flex items-center justify-between border-muted-foreground/20 bg-primary/5 h-7 w-full gap-1 text-sm text-muted-foreground font-mono truncate rounded-md border px-2 py-1">
        <span className="flex-grow">{value}</span>
        <CopyButton active={isCopied} onClick={() => copy(value)} />
      </div>
    </div>
  );
}

function CopyButton({
  active,
  onClick,
}: {
  active: boolean;
  onClick: () => void;
}) {
  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    e.stopPropagation();
    onClick();
  };

  return (
    <Button size="xs" variant="ghost" onClick={handleClick} className="size-5">
      {active ? (
        <Icon icon={faCheck} className="size-4" />
      ) : (
        <Icon icon={faClipboard} className="size-4" />
      )}
    </Button>
  );
}
