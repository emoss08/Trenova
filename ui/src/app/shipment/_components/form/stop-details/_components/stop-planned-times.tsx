export function StopPlannedTimes({
  plannedArrival,
}: {
  plannedArrival: { date: string; time: string };
}) {
  return (
    <StopPlannedTimesOuter>
      <div className="text-primary text-xs">{plannedArrival.date}</div>
      <div className="text-muted-foreground text-2xs">
        {plannedArrival.time}
      </div>
    </StopPlannedTimesOuter>
  );
}

function StopPlannedTimesOuter({ children }: { children: React.ReactNode }) {
  return <div className="w-24 text-right text-sm">{children}</div>;
}
