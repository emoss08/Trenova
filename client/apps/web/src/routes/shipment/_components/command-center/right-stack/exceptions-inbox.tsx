import { ExceptionsList } from "@/components/work-modules/exceptions-list";
import { Button } from "@trenova/shared/components/ui/button";
import { useCallback, useState } from "react";
import { useCommandCenterUrl } from "../url-state";
import { ModuleCard } from "./module-card";

export function ExceptionsInbox({ enabled = true }: { enabled?: boolean }) {
  const [, setUrl] = useCommandCenterUrl();
  const [count, setCount] = useState(0);

  const handleSelect = useCallback(
    (shipmentId: string) => setUrl({ expanded: shipmentId }),
    [setUrl],
  );

  return (
    <ModuleCard
      id="exceptions"
      title="Exceptions"
      count={count}
      countTone="danger"
      rightSlot={
        <Button variant="ghost" size="xxs" className="text-muted-foreground">
          Mute · 1h
        </Button>
      }
    >
      <ExceptionsList enabled={enabled} onSelect={handleSelect} onCount={setCount} />
    </ModuleCard>
  );
}
