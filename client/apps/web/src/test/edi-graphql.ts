/**
 * The EDI record reads the Desk cards and EDI screens make over GraphQL, keyed and
 * shaped for tests: a test answers each operation from a table keyed by
 * `ediGraphQLKey`, holding the record as the server's domain struct marshals it,
 * and `ediGraphQLResponse` wraps it the way the operation's response carries it.
 */
type Variables = Record<string, unknown>;
type Wire = Record<string, unknown>;

/** `Operation:id`, or `EdiInboundFileTransfers:fileId:after` for one page of a file's tenders. */
export function ediGraphQLKey(operationName: string, variables: Variables = {}): string {
  if (operationName === "EdiInboundFileTransfers") {
    const input = (variables.input ?? {}) as { after?: string | null };
    return `${operationName}:${String(variables.inboundFileId)}:${input.after ?? ""}`;
  }
  return `${operationName}:${String(variables.id)}`;
}

export function ediGraphQLResponse(operationName: string, answer: unknown): unknown {
  const wire = answer as Wire;
  switch (operationName) {
    case "EdiMessageRecord": {
      const diagnostics = (wire.validationErrors as unknown[] | undefined) ?? [];
      return {
        ediMessage: {
          message: wire,
          diagnosticCount: (wire.diagnosticCount as number | undefined) ?? diagnostics.length,
          diagnostics,
        },
      };
    }
    case "EdiMessageInspection":
      return { ediMessageInspection: wire };
    case "EdiInboundFileRecord": {
      const messages = (wire.messages as unknown[] | null | undefined) ?? [];
      const profile = wire.communicationProfile as { name?: string } | null | undefined;
      return {
        ediInboundFile: {
          file: wire,
          communicationProfileName: profile?.name ?? null,
          messageCount: messages.length,
          messages,
        },
      };
    }
    case "EdiInboundFileInspection":
      return { ediInboundFileInspection: wire };
    case "EdiTransferRecord":
      return { ediTransfer: { transfer: wire } };
    case "EdiTransferMappingPreview":
      return { ediTransferMappingPreview: wire };
    case "EdiInboundFileTransfers": {
      const page = wire as { results: unknown[]; hasNextPage?: boolean; endCursor?: string };
      return {
        ediTransfers: {
          edges: page.results.map((node) => ({ node })),
          pageInfo: {
            hasNextPage: page.hasNextPage ?? false,
            endCursor: page.endCursor || null,
          },
        },
      };
    }
    default:
      throw new Error(`No EDI test answer for ${operationName}`);
  }
}
