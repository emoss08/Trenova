import { ApprovedPTOOverview } from "./approved/approved-pto-overview";
import { RequestedPTOOverview } from "./requested/requested-pto-overview";

export function PTOContent() {
  return (
    <PTOContentInner>
      <ApprovedPTOOverview />
      <RequestedPTOOverview />
    </PTOContentInner>
  );
}

function PTOContentInner({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-row gap-4 h-[400px]">{children}</div>;
}
