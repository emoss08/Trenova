import { useDesk } from "./_components/desk-layout";
import { DeskToday } from "./_components/today/desk-today";

export function DeskHomePage() {
  const desk = useDesk();

  return (
    <DeskToday
      agents={desk.agents}
      threads={desk.threads}
      isLoading={desk.isLoading}
      isStarting={desk.isStarting}
      onStart={desk.startInMargin}
    />
  );
}
