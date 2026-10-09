import type { WatchedRecordChange } from "@/types/assistant";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { DeskWorldChange } from "../desk-world-change";

afterEach(cleanup);

function change(overrides: Partial<WatchedRecordChange>): WatchedRecordChange {
  return {
    recordId: "shp_1",
    resource: "audited:shipment",
    label: "shipment S1",
    action: "updated",
    fields: [],
    actorType: "session_user",
    actorUserId: "usr_other",
    at: 0,
    ...overrides,
  };
}

describe("DeskWorldChange", () => {
  beforeEach(() => {
    useAuthStore.setState({ user: { id: "usr_me" } as never });
  });

  it("says what changed on each record and who changed it", () => {
    render(
      <DeskWorldChange
        changes={[
          change({ fields: ["status", "appointment"] }),
          change({ recordId: "shp_2", label: "", action: "deleted", actorType: "agent" }),
          change({ recordId: "shp_3", label: "shipment S3", actorUserId: "usr_me" }),
        ]}
      />,
    );

    const items = screen.getAllByRole("listitem").map((item) => item.textContent);
    expect(items).toEqual([
      "shipment S1 Changed: status, appointment · By someone else",
      "shp_2 Deleted · By an agent",
      "shipment S3 Changed · By you, elsewhere",
    ]);
  });

  it("draws nothing for a notice with no records", () => {
    const { container } = render(<DeskWorldChange changes={[]} />);

    expect(container).toBeEmptyDOMElement();
  });
});
