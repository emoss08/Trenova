/**
 * A stored phone number as a person reads it. North American numbers (`+1` and ten digits, with
 * or without the `+1`) read `(815) 555-0142`; anything else is shown as it is stored, since
 * guessing another country's grouping would be worse than not grouping it.
 */
export function formatPhoneForDisplay(phone: string): string {
  const digits = phone.replace(/\D/g, "");
  const national = digits.length === 11 && digits.startsWith("1") ? digits.slice(1) : digits;
  if (national.length !== 10 || (phone.trim().startsWith("+") && !phone.trim().startsWith("+1"))) {
    return phone;
  }
  return `(${national.slice(0, 3)}) ${national.slice(3, 6)}-${national.slice(6)}`;
}
