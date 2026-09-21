import { useDesk } from "./_components/desk-layout";
import { DeskHome } from "./_components/desk-home";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";

export function DeskHomePage() {
  const desk = useDesk();

  return (
    <ScrollArea className="min-h-0 flex-1">
      <DeskHome
        agents={desk.agents}
        threads={desk.threads}
        isLoading={desk.isLoading}
        isStarting={desk.isStarting}
        onStart={desk.start}
      />
    </ScrollArea>
  );
}
