import {
  SequenceConfigsDocument,
  UpdateSequenceConfigsDocument,
  type SequenceConfigDocumentInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { sequenceConfigDocumentSchema, type SequenceConfigDocument } from "@/types/sequence-config";

export function toSequenceConfigDocumentInput(
  data: SequenceConfigDocument,
): SequenceConfigDocumentInput {
  return {
    configs: data.configs.map((config) => ({
      id: config.id || null,
      sequenceType: config.sequenceType,
      prefix: config.prefix,
      includeYear: config.includeYear,
      yearDigits: config.yearDigits,
      includeMonth: config.includeMonth,
      includeWeekNumber: config.includeWeekNumber,
      includeDay: config.includeDay,
      sequenceDigits: config.sequenceDigits,
      includeLocationCode: config.includeLocationCode,
      includeRandomDigits: config.includeRandomDigits,
      randomDigitsCount: config.randomDigitsCount,
      includeCheckDigit: config.includeCheckDigit,
      includeBusinessUnitCode: config.includeBusinessUnitCode,
      useSeparators: config.useSeparators,
      separatorChar: config.separatorChar,
      allowCustomFormat: config.allowCustomFormat,
      customFormat: config.customFormat,
      locationCodeStrategy: config.locationCodeStrategy
        ? {
            components: config.locationCodeStrategy.components,
            componentWidth: config.locationCodeStrategy.componentWidth,
            sequenceDigits: config.locationCodeStrategy.sequenceDigits,
            separator: config.locationCodeStrategy.separator,
            casing: config.locationCodeStrategy.casing,
            fallbackPrefix: config.locationCodeStrategy.fallbackPrefix,
          }
        : null,
      version: config.version,
    })),
  };
}

export class SequenceConfigService {
  public async get(options?: { signal?: AbortSignal }) {
    const response = await requestGraphQL({
      document: SequenceConfigsDocument,
      operationName: "SequenceConfigs",
      signal: options?.signal,
    });

    return safeParse(
      sequenceConfigDocumentSchema,
      response.sequenceConfigs,
      "SequenceConfigDocument",
    );
  }

  public async update(data: SequenceConfigDocument) {
    const response = await requestGraphQL({
      document: UpdateSequenceConfigsDocument,
      operationName: "UpdateSequenceConfigs",
      variables: { input: toSequenceConfigDocumentInput(data) },
    });

    return safeParse(
      sequenceConfigDocumentSchema,
      response.updateSequenceConfigs,
      "SequenceConfigDocument",
    );
  }
}
