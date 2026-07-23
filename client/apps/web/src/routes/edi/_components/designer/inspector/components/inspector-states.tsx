import { DatabaseIcon } from "lucide-react";

export default function InspectorState({
  state,
  message,
}: {
  state: "loading" | "empty" | "error";
  message?: string;
}) {
  const displayMessage =
    message ??
    (state === "loading"
      ? "Loading message detail."
      : state === "error"
        ? "Unable to load inspection."
        : "Message detail unavailable.");

  return (
    <div
      aria-live={state === "loading" ? "polite" : undefined}
      className="flex h-48 flex-col items-center justify-center gap-2 p-4 text-sm text-muted-foreground"
    >
      <DatabaseIcon className="size-5" />
      {displayMessage}
    </div>
  );
}
