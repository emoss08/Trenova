import "yup";

declare module "yup" {
  // eslint-disable-next-line no-shadow
  interface StringSchema {
    decimal(message?: string): StringSchema;
  }
}
