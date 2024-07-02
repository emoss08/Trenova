import type {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
  UnitOfMeasureChoiceProps,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types/index";
import { type BaseModel } from "@/types/organization";

export interface HazardousMaterial extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  description?: string;
  hazardClass: HazardousClassChoiceProps;
  packingGroup?: PackingGroupChoiceProps;
  ergNumber?: string;
  properShippingName?: string;
}

export type HazardousMaterialFormValues = Omit<
  HazardousMaterial,
  "id" | "organizationId" | "createdAt" | "updatedAt" | "version"
>;

export interface Commodity extends BaseModel {
  id: string;
  status: StatusChoiceProps;
  name: string;
  description?: string;
  minTemp?: number;
  maxTemp?: number;
  unitOfMeasure?: UnitOfMeasureChoiceProps;
  hazardousMaterialId?: string | null;
  isHazmat: boolean;
}

export type CommodityFormValues = Omit<
  Commodity,
  "id" | "createdAt" | "updatedAt" | "organizationId" | "version"
>;
