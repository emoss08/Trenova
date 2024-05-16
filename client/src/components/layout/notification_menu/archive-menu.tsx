import { InboxIcon } from "lucide-react";

export function ArchiveMenuContent() {
  return (
    <div className="flex h-80 w-full items-center justify-center p-4">
      <div className="flex flex-col items-center justify-center gap-y-3">
        <div className="bg-accent flex size-10 items-center justify-center rounded-full">
          <InboxIcon className="text-muted-foreground" />
        </div>
        <p className="text-muted-foreground select-none text-center text-sm">
          Nothing appears to be here
        </p>
      </div>
    </div>
  );
}
