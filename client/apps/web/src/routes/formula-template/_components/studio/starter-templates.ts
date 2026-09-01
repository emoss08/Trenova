import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";

export type StarterTemplate = {
  id: string;
  name: string;
  description: string;
  values: Pick<
    FormulaTemplateFormValues,
    "expression" | "variableDefinitions" | "breakdownDefinitions" | "minCharge" | "maxCharge"
  >;
};

export const STARTER_TEMPLATES: StarterTemplate[] = [
  {
    id: "per-mile-fuel",
    name: "Per mile with fuel surcharge",
    description: "Rate per mile times distance, with a percentage fuel surcharge on top.",
    values: {
      expression: "round((baseRate * totalDistance) * (1 + fuelSurchargePercent / 100), 2)",
      variableDefinitions: [
        {
          name: "fuelSurchargePercent",
          type: "Number",
          description: "Fuel surcharge percentage applied to the linehaul",
          required: true,
          defaultValue: 18,
        },
      ],
      breakdownDefinitions: [
        {
          name: "linehaul",
          label: "Linehaul",
          expression: "round(baseRate * totalDistance, 2)",
        },
        {
          name: "fuel",
          label: "Fuel Surcharge",
          expression: "round((baseRate * totalDistance) * (fuelSurchargePercent / 100), 2)",
        },
      ],
      minCharge: null,
      maxCharge: null,
    },
  },
  {
    id: "weight-breaks",
    name: "Weight breaks",
    description: "Per-hundredweight pricing with cheaper rates at higher weights.",
    values: {
      expression:
        "ceil(totalWeight / 100) * (totalWeight < 5000 ? rateUnder5k : totalWeight < 10000 ? rate5kTo10k : rateOver10k)",
      variableDefinitions: [
        {
          name: "rateUnder5k",
          type: "Number",
          description: "Rate per CWT below 5,000 lbs",
          required: true,
          defaultValue: 18.5,
        },
        {
          name: "rate5kTo10k",
          type: "Number",
          description: "Rate per CWT from 5,000 to 10,000 lbs",
          required: true,
          defaultValue: 14.25,
        },
        {
          name: "rateOver10k",
          type: "Number",
          description: "Rate per CWT above 10,000 lbs",
          required: true,
          defaultValue: 11.75,
        },
      ],
      breakdownDefinitions: [],
      minCharge: null,
      maxCharge: null,
    },
  },
  {
    id: "flat-with-minimum",
    name: "Flat rate with minimum",
    description: "Uses the shipment's base rate, but never bills below a floor.",
    values: {
      expression: "baseRate",
      variableDefinitions: [],
      breakdownDefinitions: [],
      minCharge: 250,
      maxCharge: null,
    },
  },
  {
    id: "hazmat-surcharge",
    name: "Hazmat surcharge",
    description: "Adds a hazardous-materials fee only when the shipment carries hazmat.",
    values: {
      expression: "hasHazmat ? hazmatFee : 0",
      variableDefinitions: [
        {
          name: "hazmatFee",
          type: "Number",
          description: "Hazardous materials handling fee",
          required: true,
          defaultValue: 175,
        },
      ],
      breakdownDefinitions: [],
      minCharge: null,
      maxCharge: null,
    },
  },
];
