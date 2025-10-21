import * as z from "zod/v4";
import { additionalChargeSchema } from "./additional-charge-schema";
import { customerSchema } from "./customer-schema";
import { equipmentTypeSchema } from "./equipment-type-schema";
import { formulaTemplateSchema } from "./formula-template-schema";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { moveSchema } from "./move-schema";
import { serviceTypeSchema } from "./service-type-schema";
import { shipmentCommentSchema } from "./shipment-comment-schema";
import { shipmentCommoditySchema } from "./shipment-commodity-schema";
import { shipmentHoldSchema } from "./shipment-hold-schema";
import { shipmentTypeSchema } from "./shipment-type-schema";
import { userSchema } from "./user-schema";

export const ShipmentStatus = z.enum([
  "New",
  "PartiallyAssigned",
  "Assigned",
  "InTransit",
  "Delayed",
  "PartiallyCompleted",
  "Completed",
  "Billed",
  "ReadyToBill",
  "Canceled",
]);

export const RatingMethod = z.enum([
  "FlatRate",
  "PerMile",
  "PerStop",
  "PerPound",
  "PerPallet",
  "PerLinearFoot",
  "Other",
  "FormulaTemplate",
]);

// Temperature validation helper
const temperatureSchema = z.number().int().min(-100).max(200);

const chargeAmountSchema = decimalStringSchema.default(0);

export const shipmentSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,
    proNumber: optionalStringSchema, // * Will not be available on create, but will be added on update
    bol: z
      .string()
      .min(1, {
        error: "BOL is required",
      })
      .max(100, {
        error: "BOL must be between 1 and 100 characters",
      }),

    serviceTypeId: z.string().min(1, {
      error: "Service Type is required",
    }),
    shipmentTypeId: z.string().min(1, {
      error: "Shipment Type is required",
    }),
    customerId: z.string().min(1, {
      error: "Customer is required",
    }),
    tractorTypeId: nullableStringSchema,
    trailerTypeId: nullableStringSchema,
    ownerId: nullableStringSchema,
    enteredById: nullableStringSchema,
    canceledById: nullableStringSchema,
    formulaTemplateId: nullableStringSchema,
    status: ShipmentStatus,
    cancelReason: nullableStringSchema,
    ratingMethod: RatingMethod,
    otherChargeAmount: chargeAmountSchema,
    freightChargeAmount: chargeAmountSchema,
    totalChargeAmount: chargeAmountSchema,
    temperatureMin: temperatureSchema.nullish(),
    temperatureMax: temperatureSchema.nullish(),
    pieces: nullableIntegerSchema,
    weight: nullableIntegerSchema,
    actualDeliveryDate: nullableIntegerSchema,
    actualShipDate: nullableIntegerSchema,
    canceledAt: nullableIntegerSchema,
    ratingUnit: nullableIntegerSchema,

    // * Relationships
    shipmentType: shipmentTypeSchema.nullish(),
    serviceType: serviceTypeSchema.nullish(),
    customer: customerSchema.nullish(),
    formulaTemplate: formulaTemplateSchema.nullish(),
    tractorType: equipmentTypeSchema.nullish(),
    trailerType: equipmentTypeSchema.nullish(),
    owner: userSchema.nullish(),
    canceledBy: userSchema.nullish(),

    // * Collections
    moves: z.array(moveSchema),
    commodities: z.array(shipmentCommoditySchema).nullish(),
    additionalCharges: z.array(additionalChargeSchema).nullish(),
    comments: z.array(shipmentCommentSchema).nullish(),
    holds: z.array(shipmentHoldSchema).nullish(),
  })
  .refine(
    (data) => {
      // Freight Charge Amount is required when rating method is FlatRate
      if (
        data.ratingMethod === "FlatRate" &&
        (data.freightChargeAmount === null ||
          data.freightChargeAmount === undefined)
      ) {
        return false;
      }
      return true;
    },
    {
      message: "Freight Charge Amount is required when rating method is Flat",
      path: ["freightChargeAmount"],
    },
  )
  .refine(
    (data) => {
      // Weight is required when rating method is per pound
      if (
        data.ratingMethod === "PerPound" &&
        (data.weight === null || data.weight === undefined)
      ) {
        return false;
      }
      return true;
    },
    {
      message: "Weight is required when rating method is Per Pound",
      path: ["weight"],
    },
  )
  .refine(
    (data) => {
      // Rating Unit is required and must be > 0 when rating method is Per Mile
      if (
        data.ratingMethod === "PerMile" &&
        (data.ratingUnit === null ||
          data.ratingUnit === undefined ||
          data.ratingUnit < 1)
      ) {
        return false;
      }
      return true;
    },
    {
      message:
        "Rating Unit is required when rating method is Per Mile and must be greater than 0",
      path: ["ratingUnit"],
    },
  )
  .refine(
    (data) => {
      if (data.ratingMethod === "FormulaTemplate") {
        if (
          data.formulaTemplateId === null ||
          data.formulaTemplateId === undefined ||
          data.formulaTemplateId === ""
        ) {
          return false;
        }
      }
      return true;
    },
    {
      message:
        "Formula Template is required when rating method is Formula Template",
      path: ["formulaTemplateId"],
    },
  )
  .refine(
    (data) => {
      // Temperature Max cannot be less than Temperature Min
      if (
        data.temperatureMin !== null &&
        data.temperatureMax !== null &&
        data.temperatureMax !== undefined &&
        data.temperatureMin !== undefined &&
        data.temperatureMax < data.temperatureMin
      ) {
        return false;
      }
      return true;
    },
    {
      message: "Temperature Max must be greater than Temperature Min",
      path: ["temperatureMax"],
    },
  )
  .refine(
    (data) => {
      // Temperature Min cannot be greater than Temperature Max
      if (
        data.temperatureMin !== null &&
        data.temperatureMax !== null &&
        data.temperatureMin !== undefined &&
        data.temperatureMax !== undefined &&
        data.temperatureMin > data.temperatureMax
      ) {
        return false;
      }
      return true;
    },
    {
      message: "Temperature Min must be less than Temperature Max",
      path: ["temperatureMin"],
    },
  );

export type ShipmentSchema = z.infer<typeof shipmentSchema>;

// For API requests (without relationships)
export const shipmentRequestSchema = shipmentSchema.omit({
  shipmentType: true,
  serviceType: true,
  customer: true,
  tractorType: true,
  trailerType: true,
});
