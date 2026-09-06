import { describe, expect, it } from "vitest";
import {
  dqfItemBlocks,
  dqfItemTone,
  dqfSectionLabel,
  verificationSettled,
  verificationTone,
} from "../dqf";

describe("dqfItemBlocks", () => {
  // Expiring soon warns. The document on file is still valid today, and
  // treating it as a gap would make every file incomplete for a month before
  // every renewal.
  it("stops short of expiring-soon", () => {
    expect(dqfItemBlocks("Missing")).toBe(true);
    expect(dqfItemBlocks("Expired")).toBe(true);
    expect(dqfItemBlocks("Outstanding")).toBe(true);
    expect(dqfItemBlocks("ExpiringSoon")).toBe(false);
    expect(dqfItemBlocks("Satisfied")).toBe(false);
    expect(dqfItemBlocks("NotApplicable")).toBe(false);
  });
});

describe("verificationSettled", () => {
  // An employer who never answers still settles the obligation: the rule asks
  // for a good-faith effort and a record of it, not an answer nobody can
  // compel.
  it("counts a silent employer as settled", () => {
    expect(verificationSettled("NoResponse")).toBe(true);
    expect(verificationSettled("Received")).toBe(true);
    expect(verificationSettled("NotApplicable")).toBe(true);
    expect(verificationSettled("Pending")).toBe(false);
    expect(verificationSettled("Requested")).toBe(false);
  });
});

describe("tones", () => {
  it("grades items and investigations", () => {
    expect(dqfItemTone("Satisfied")).toBe("active");
    expect(dqfItemTone("ExpiringSoon")).toBe("warning");
    expect(dqfItemTone("Missing")).toBe("inactive");
    expect(verificationTone("Received")).toBe("active");
    expect(verificationTone("Requested")).toBe("warning");
    expect(verificationTone("NoResponse")).toBe("secondary");
  });
});

describe("dqfSectionLabel", () => {
  it("names each section", () => {
    expect(dqfSectionLabel("SafetyHistory")).toBe("Previous employers");
  });

  // A section the client does not recognise still has to render as something.
  it("falls back to the raw value", () => {
    expect(dqfSectionLabel("SomethingNew")).toBe("SomethingNew");
  });
});
