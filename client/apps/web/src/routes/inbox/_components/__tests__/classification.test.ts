import { goEnumValues } from "@/test/go-source";
import { describe, expect, it } from "vitest";
import {
  CLASSIFICATION_ICON,
  CLASSIFICATION_VARIANT,
  INBOUND_CLASSIFICATIONS,
} from "../classification";

/**
 * The kinds rail is drawn from INBOUND_CLASSIFICATIONS, and the counts come
 * back keyed by the server's kinds. A kind added in Go and missing here is a
 * folder nobody can open, holding mail nobody sees — so the list is checked
 * against the server's own, not against a second hand-written copy.
 */
describe("inbound classifications", () => {
  const serverKinds = goEnumValues({
    file: "services/tms/internal/core/domain/inboundmessage/enums.go",
    typeName: "Classification",
  });

  it("lists every kind the server can read a message as, in the server's order", () => {
    expect([...INBOUND_CLASSIFICATIONS]).toEqual(serverKinds);
  });

  it("gives every kind a colour and an icon", () => {
    for (const kind of serverKinds) {
      expect(CLASSIFICATION_VARIANT).toHaveProperty(kind);
      expect(CLASSIFICATION_ICON).toHaveProperty(kind);
    }
  });
});
