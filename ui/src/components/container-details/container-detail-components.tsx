import { cn } from "@/lib/utils";
import { useContainerLogStore } from "@/stores/docker-store";
import { ContainerInspect } from "@/types/docker";
import { Check, Clipboard } from "lucide-react";
import { useState } from "react";
import { Button, CopyButton } from "../ui/button";
import { ScrollArea } from "../ui/scroll-area";

export function CommandDetails({
  details,
  handleCopy,
  copiedKey,
}: {
  details?: ContainerInspect;
  handleCopy: (text: string, key: string) => void;
  copiedKey: string | null;
}) {
  const selectedContainer = useContainerLogStore.get("selectedContainer");

  return (
    <div className="p-4">
      <ContainerDetailHeader title="Command" description="Entrypoint + args" />
      <div className="rounded-md border p-3">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs text-muted-foreground">Shell-friendly</span>
          <CopyButton
            label="Copy"
            onClick={() =>
              handleCopy(
                details?.Config?.Cmd?.join(" ") ||
                  selectedContainer?.Command ||
                  "",
                "cmd",
              )
            }
            active={copiedKey === "cmd"}
          />
        </div>
        <div className="rounded-md border p-3 bg-muted">
          <code className="text-xs bg-muted p-2 rounded-md">
            {details?.Config?.Cmd?.join(" ") ||
              selectedContainer?.Command ||
              "—"}
          </code>
        </div>
      </div>
    </div>
  );
}

export function EnvironmentVariables({
  details,
  handleCopy,
  copiedKey,
}: {
  details?: ContainerInspect;
  handleCopy: (text: string, key: string) => void;
  copiedKey: string | null;
}) {
  const [isRevealed, setIsRevealed] = useState(false);

  return (
    <div className="p-4">
      <ContainerDetailHeader
        title="Environment Variables"
        description="From docker inspect"
      />
      <div className="rounded-md border p-3 relative">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs text-muted-foreground">
            {details?.Config?.Env?.length ?? 0} variables
          </span>
          {details?.Config?.Env?.length ? (
            <CopyButton
              label="Copy all"
              onClick={() => handleCopy(details.Config.Env.join("\n"), "env")}
              active={copiedKey === "env"}
            />
          ) : null}
        </div>
        <div className="relative">
          <ScrollArea className="h-[200px] rounded-md border p-3">
            <pre className="text-xs leading-5">
              {details?.Config?.Env?.join("\n") || "No environment variables"}
            </pre>
          </ScrollArea>
          {!isRevealed && (
            <div className="absolute inset-0 bg-background/10 backdrop-blur-sm rounded-md border flex items-center justify-center">
              <Button size="sm" onClick={() => setIsRevealed(true)}>
                View Environment Variables
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export function ContainerLabels({
  handleCopy,
  copiedKey,
}: {
  handleCopy: (text: string, key: string) => void;
  copiedKey: string | null;
}) {
  const selectedContainer = useContainerLogStore.get("selectedContainer");
  return (
    <div className="p-4">
      <ContainerDetailHeader
        title="Labels"
        description="Arbitrary key/value metadata"
      />
      <ScrollArea className="h-[150px] rounded-md border p-3">
        {Object.keys(selectedContainer?.Labels || {}).length ? (
          <div className="space-y-1">
            {Object.entries(selectedContainer?.Labels || {}).map(([k, v]) => (
              <div
                key={k}
                className="text-xs flex items-center justify-between gap-2"
              >
                <div className="space-x-2">
                  <span className="font-semibold">{k}:</span>
                  <span className="text-muted-foreground max-w-[200px]">
                    {v}
                  </span>
                </div>
                <CopyIcon
                  ariaLabel="Copy label"
                  onClick={() => handleCopy(`${k}=${v}`, `label-${k}`)}
                  active={copiedKey === `label-${k}`}
                />
              </div>
            ))}
          </div>
        ) : (
          <div className="text-xs text-muted-foreground">No labels</div>
        )}
      </ScrollArea>
    </div>
  );
}

export function CopyIcon({
  onClick,
  ariaLabel,
  active,
}: {
  onClick: () => void;
  ariaLabel: string;
  active?: boolean;
}) {
  return (
    <Button
      size="icon"
      variant="ghost"
      aria-label={ariaLabel}
      onClick={onClick}
      className="h-7 w-7 shrink-0"
    >
      {active ? (
        <Check className="h-4 w-4" />
      ) : (
        <Clipboard className="h-4 w-4" />
      )}
    </Button>
  );
}

export function KV({
  label,
  children,
}: {
  label: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-3 py-1">
      <span className="text-muted-foreground">{label}:</span>
      <div className="min-w-0 text-right">{children ?? "—"}</div>
    </div>
  );
}

export function Mono({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return <span className={cn("font-mono text-xs", className)}>{children}</span>;
}

export function ContainerDetailHeader({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="pb-2">
      <h3 className="text-sm">{title}</h3>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  );
}
