import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { useReviewApproval } from "../use-permission";

const ME = "usr_me";

afterEach(() => {
  cleanup();
  useAuthStore.setState({ user: null } as never);
});

function Probe({ submittedById }: { submittedById: string | null }) {
  const approval = useReviewApproval(Resource.FormulaTemplate, submittedById);
  return (
    <p>
      {approval.allowed ? "allowed" : "not allowed"},{" "}
      {approval.needsSecondReviewer ? "needs a second reviewer" : "no second reviewer"}
    </p>
  );
}

function renderAs(operations: number, submittedById: string | null) {
  usePermissionStore.setState({
    manifest: { permissions: { [Resource.FormulaTemplate]: operations } },
  } as never);
  useAuthStore.setState({ user: { id: ME } } as never);
  render(<Probe submittedById={submittedById} />);
}

describe("useReviewApproval", () => {
  it("lets an approver approve someone else's submission", () => {
    renderAs(Operation.Read | Operation.Approve, "usr_someone");

    expect(screen.getByText("allowed, no second reviewer")).toBeInTheDocument();
  });

  it("holds back an approver's own submission for a second reviewer", () => {
    renderAs(Operation.Read | Operation.Approve, ME);

    expect(screen.getByText("not allowed, needs a second reviewer")).toBeInTheDocument();
  });

  it("lets an approver who may approve their own changes approve their own submission", () => {
    renderAs(Operation.Read | Operation.Approve | Operation.ApproveOwn, ME);

    expect(screen.getByText("allowed, no second reviewer")).toBeInTheDocument();
  });

  it("offers nothing to someone who cannot approve", () => {
    renderAs(Operation.Read, ME);

    expect(screen.getByText("not allowed, no second reviewer")).toBeInTheDocument();
  });
});
