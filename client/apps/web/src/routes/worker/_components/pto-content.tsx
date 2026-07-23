import { ApprovedPTOOverview } from "./pto/approved/approved-pto-overview";
import { RequestedPTOOverview } from "./pto/requested/requested-pto-overview";

export function PTOContent() {
  return (
    <ContentInner>
      <ApprovedPTOOverview />
      <RequestedPTOOverview />
    </ContentInner>
  );
}

function ContentInner({ children }: { children: React.ReactNode }) {
  return <div className="flex h-125 flex-col gap-4 lg:flex-row">{children}</div>;
}
