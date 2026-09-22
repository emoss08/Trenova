import { ARTIFACT_KINDS } from "@/components/assistant/voice/artifact-chrome";
import { goEnumValues } from "@/test/go-source";
import { artifactKindSchema } from "@/types/assistant";
import { describe, expect, it } from "vitest";

/**
 * Every kind of artifact the Desk can be handed has to be nameable here.
 *
 * A thread's artifacts arrive as one array and are parsed as one: a kind the
 * client has not heard of does not degrade to an unknown card, it throws the
 * whole list away. The pane then shows nothing at all — not just the new
 * kind, nothing — for every thread that happens to contain one.
 *
 * ARTIFACT_KINDS is checked alongside the schema because it is indexed
 * directly, so a kind that parses but has no entry renders as a crash rather
 * than as a missing label.
 *
 * This reads the server's own list. A kind added in Go without being added
 * here fails this test rather than the Desk.
 */
function serverKinds(): string[] {
  return goEnumValues({
    file: "services/tms/internal/core/domain/assistantartifact/enums.go",
    typeName: "Kind",
  });
}

describe("artifactKindSchema", () => {
  const kinds = serverKinds();

  it("finds the kinds the server stores", () => {
    expect(kinds.length).toBeGreaterThan(5);
  });

  it.each(kinds)("accepts the %s artifact the server stores", (kind) => {
    expect(artifactKindSchema.safeParse(kind).success).toBe(true);
  });

  it("names no kind the server does not store", () => {
    expect([...artifactKindSchema.options].sort()).toEqual([...kinds].sort());
  });

  it.each(kinds)("has a label and an icon for the %s artifact", (kind) => {
    expect(ARTIFACT_KINDS).toHaveProperty(kind);
  });

  it("still refuses a kind the server does not store", () => {
    expect(artifactKindSchema.safeParse("hologram").success).toBe(false);
  });
});
