import { Link } from "@trenova/shared/components/link";
import { FactoringSubmissionStatusBadge } from "@/components/factoring/factoring-badges";
import { recordPath } from "@/config/record-links";
import type { FactoringSubmissionStatus } from "@trenova/graphql/generated/graphql";

/** The schedule an invoice is on while it is with the factor, and how its delivery stands. */
export function FactoringSubmissionLink({
  submission,
}: {
  submission: {
    submissionId: string;
    submissionNumber: string;
    submissionStatus: FactoringSubmissionStatus;
  };
}) {
  return (
    <span className="inline-flex items-center gap-2">
      <Link
        to={recordPath("factoring_submission", submission.submissionId)}
        className="text-brand font-mono hover:underline"
      >
        {submission.submissionNumber}
      </Link>
      <FactoringSubmissionStatusBadge status={submission.submissionStatus} />
    </span>
  );
}
