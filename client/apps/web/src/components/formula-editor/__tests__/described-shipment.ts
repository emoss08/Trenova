import { formulaSchemaResponseSchema } from "@trenova/shared/types/formula-template";
import described from "../../../../../../../services/tms/internal/modules/rating/app/formula/testdata/shipment.describe.json";

/**
 * The engine's own description of the shipment schema, as `GET
 * /formula-templates/schema` serves it. The Go test that writes the file keeps
 * it equal to what the engine publishes, so editor tests check against the
 * real schema and functions rather than a copy kept by hand.
 */
export const describedShipment = formulaSchemaResponseSchema.parse(described);
