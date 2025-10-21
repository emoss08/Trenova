import { StopBasicInformationForm } from "./stop-basic-information-form";
import { StopScheduleInformationForm } from "./stop-schedule-form";

export function StopDialogForm({
  moveIdx,
  stopIdx,
}: {
  moveIdx: number;
  stopIdx: number;
}) {
  return (
    <StopDialogFormOuter>
      <StopBasicInformationForm moveIdx={moveIdx} stopIdx={stopIdx} />
      <StopScheduleInformationForm moveIdx={moveIdx} stopIdx={stopIdx} />
    </StopDialogFormOuter>
  );
}

function StopDialogFormOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-2">{children}</div>;
}
