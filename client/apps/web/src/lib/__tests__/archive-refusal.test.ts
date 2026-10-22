import { ApiRequestError } from "@trenova/shared/lib/api";
import { describe, expect, it } from "vitest";
import { archiveRefusal } from "../archive-refusal";

const usageMessage =
  "Archiving stops these templates rating anything: Per Mile (12 shipments, 1 billing fallback). Confirm to archive them anyway";

describe("archiveRefusal", () => {
  it("asks for confirmation with the server's list of what the templates price", () => {
    const error = new ApiRequestError(422, {
      type: "validation-error",
      title: "Validation failed",
      status: 422,
      errors: [
        { field: "name", message: "Name is required" },
        { field: "confirmInUse", message: usageMessage, code: "INVALID" },
      ],
    });

    expect(archiveRefusal(error)).toEqual({ kind: "needsConfirmation", message: usageMessage });
  });

  it("says when the person may not archive a template in use", () => {
    const error = new ApiRequestError(403, {
      type: "authorization-error",
      title: "Forbidden",
      status: 403,
      detail: "Archiving a formula template that is still in use needs approve permission",
    });

    expect(archiveRefusal(error)).toEqual({
      kind: "forbidden",
      message: "Archiving a formula template that is still in use needs approve permission",
    });
  });

  it("leaves every other failure to the generic error", () => {
    const otherField = new ApiRequestError(422, {
      type: "validation-error",
      title: "Validation failed",
      status: 422,
      errors: [{ field: "status", message: "Invalid status" }],
    });

    expect(archiveRefusal(otherField)).toBeNull();
    expect(
      archiveRefusal(
        new ApiRequestError(500, { type: "about:blank", title: "Error", status: 500 }),
      ),
    ).toBeNull();
    expect(archiveRefusal(new Error("network down"))).toBeNull();
  });
});
