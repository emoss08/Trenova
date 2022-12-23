function isNumber(value: string): boolean {
  return new RegExp('^(?=.*[0-9]).+$').test(value);
}

function isLowercaseChar(value: string): boolean {
  return new RegExp('^(?=.*[a-z]).+$').test(value);
}

function isUppercaseChar(value: string): boolean {
  return new RegExp('^(?=.*[A-Z]).+$').test(value);
}

function isSpecialChar(value: string): boolean {
  return new RegExp('^(?=.*[-+_!@#$%^&*.,?]).+$').test(value);
}

function minLength(value: string): boolean {
  return value.length > 7;
}

export { isNumber, isLowercaseChar, isUppercaseChar, isSpecialChar, minLength };
