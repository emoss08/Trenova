import { DeskHome } from "./_components/desk-home";
import { useDesk } from "./_components/desk-layout";

export function DeskHomePage() {
  const desk = useDesk();

  return (
    <DeskHome
      agents={desk.agents}
      threads={desk.threads}
      isLoading={desk.isLoading}
      isStarting={desk.isStarting}
      onStart={desk.start}
    />
  );
}
