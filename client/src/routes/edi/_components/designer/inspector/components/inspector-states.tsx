import { DatabaseIcon } from "lucide-react";

export default function InspectorState({
  state,
  message = "Message detail unavailable.",
}: {
  state: "loading" | "empty";
  message?: string;
}) {
  return (
    <div className="flex h-48 flex-col items-center justify-center gap-2 p-4 text-sm text-muted-foreground">
      <DatabaseIcon className="size-5" />
      {state === "loading" ? "Loading message detail." : message}
    </div>
  );
}
