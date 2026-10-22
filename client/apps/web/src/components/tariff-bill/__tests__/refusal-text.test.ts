import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { refusalText, tariffMessageText } from "../refusal-text";

describe("tariffMessageText", () => {
  it("shows text the server already wrote as it is", () => {
    expect(tariffMessageText("The shipment has no weight", translate)).toBe(
      "The shipment has no weight",
    );
  });

  it("translates a module message with its arguments", () => {
    expect(
      tariffMessageText({ message: "Nothing is named {0}", args: ["`nope`"] }, translate),
    ).toBe("Nothing is named `nope`");
  });
});

describe("refusalText", () => {
  it("shows an author's refusal as written and translates the module's", () => {
    expect(refusalText({ message: "No weight, no rate" } as never, translate)).toBe(
      "No weight, no rate",
    );
    expect(
      refusalText({ message: { message: "Set {0}", args: ["rate"] } } as never, translate),
    ).toBe("Set rate");
  });
});
