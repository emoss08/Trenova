import { parseAsStringLiteral } from "nuqs";

export const searchParamsParser = {
  tab: parseAsStringLiteral(["general", "security", "billing-usage"]).withDefault("general"),
  securityTab: parseAsStringLiteral(["sign-in", "provisioning", "policies", "activity"])
    .withDefault("sign-in")
    .withOptions({ history: "push" }),
};
