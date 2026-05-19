export function controlNumberText(message: {
  interchangeControlNumber: string;
  groupControlNumber: string;
  transactionControlNumber: string;
}) {
  return [
    `ISA: ${message.interchangeControlNumber}`,
    `GS: ${message.groupControlNumber}`,
    `ST: ${message.transactionControlNumber}`,
  ].join("\n");
}
