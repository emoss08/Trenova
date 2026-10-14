import {
  ediInboundFileSchema,
  ediInboundFileInspectionSchema,
  type EDIInboundFile,
  type EDIInboundFileInspection,
} from "@trenova/shared/types/edi";
import batchFile from "./batch-file.json";
import duplicateFile from "./duplicate-file.json";
import notX12Inspection from "./not-x12.inspection.json";
import partnerBatchInspection from "./partner-batch.inspection.json";
import quarantinedFile from "./quarantined-file.json";
import simulator204Inspection from "./simulator-204.inspection.json";
import simulatorFile from "./simulator-file.json";

/**
 * The wire fixtures were written by the Go service itself: each file is what
 * `GET /edi/inbound-files/:id/` returns for an `EDIInboundFile` the inbound
 * parser built from a real interchange (a nil id as `null`, an unset string as
 * `""`), and each inspection what `GET /edi/inbound-files/:id/inspect/` returns
 * for that interchange.
 *
 * - simulator: one 204 the EDI partner simulator sent over AS2, processed.
 * - batch: an SFTP drop of two 204s in an SM group and a 997 in an FA group;
 *   the second 204's SE01 miscounts its segments and its tender was refused, so
 *   the file is partly processed.
 * - quarantined: a plain-text email body dropped into the mailbox, from no
 *   partner the ISA could name.
 * - duplicate: the simulator's interchange sent again under the same control
 *   number.
 */
const WIRE_FILES = {
  simulator: simulatorFile,
  batch: batchFile,
  quarantined: quarantinedFile,
  duplicate: duplicateFile,
} as const;

const WIRE_INSPECTIONS = {
  simulator: simulator204Inspection,
  batch: partnerBatchInspection,
  quarantined: notX12Inspection,
} as const;

export type WireFileName = keyof typeof WIRE_FILES;
export type WireInspectionName = keyof typeof WIRE_INSPECTIONS;

export const NOW = 1_791_720_000;

/** A fresh copy of a wire file, with any top-level field replaced. */
export function wireInboundFile(
  name: WireFileName,
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  return { ...structuredClone(WIRE_FILES[name]), ...overrides };
}

/** A fresh copy of a wire inspection. */
export function wireFileInspection(name: WireInspectionName): Record<string, unknown> {
  return structuredClone(WIRE_INSPECTIONS[name]);
}

export function inboundFile(
  name: WireFileName,
  overrides: Record<string, unknown> = {},
): EDIInboundFile {
  return ediInboundFileSchema.parse(wireInboundFile(name, overrides));
}

export function fileInspection(name: WireInspectionName): EDIInboundFileInspection {
  return ediInboundFileInspectionSchema.parse(wireFileInspection(name));
}
