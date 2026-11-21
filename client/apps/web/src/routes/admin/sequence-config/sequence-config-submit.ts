import type { SequenceConfigDocument } from "@/types/sequence-config";
import { drawsRecordCodes } from "./_components/sequence-config-constants";

/**
 * The document as it is saved. A number drawn without a location or a business unit at
 * hand (every one but a PRO number) cannot carry their codes, and the server refuses to
 * issue it while it asks for them, so the page does not offer them there and sends them off.
 */
export function prepareSequenceConfigs(values: SequenceConfigDocument): SequenceConfigDocument {
  return {
    ...values,
    configs: values.configs.map((config) =>
      drawsRecordCodes(config.sequenceType)
        ? config
        : { ...config, includeLocationCode: false, includeBusinessUnitCode: false },
    ),
  };
}
