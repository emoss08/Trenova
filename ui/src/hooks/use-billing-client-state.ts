import { parseAsBoolean } from "nuqs";

export const billingClientSearchParams = {
  transferModalOpen: parseAsBoolean.withDefault(false).withOptions({
    shallow: true,
  }),
};
