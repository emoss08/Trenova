export function controlNumberText(message: {
  interchangeControlNumber: string;
  groupControlNumber: string;
  transactionControlNumber: string;
}) {
  return [
    // i18n-ignore: X12 segment tag and control number copied to the clipboard
    `ISA: ${message.interchangeControlNumber}`,
    // i18n-ignore: X12 segment tag and control number copied to the clipboard
    `GS: ${message.groupControlNumber}`,
    // i18n-ignore: X12 segment tag and control number copied to the clipboard
    `ST: ${message.transactionControlNumber}`,
  ].join("\n");
}
