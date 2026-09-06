import type { TrainingCourseRow } from "@/lib/graphql/worker-training";
import { trainingCourseFormSchema } from "@trenova/shared/types/worker-training";
import { describe, expect, it } from "vitest";
import { buildTrainingCourseDefaults, toTrainingCourseInput } from "../training-course-panel";

const row: TrainingCourseRow = {
  id: "trnc_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  code: "HAZMAT-AWARE",
  name: "Hazmat Awareness",
  description: "49 CFR 172 Subpart H",
  category: "HazardousMaterials",
  status: "Active",
  delivery: "Classroom",
  contentUrl: null,
  durationMinutes: 180,
  passingScore: "80.00",
  validityMonths: 36,
  renewalWindowDays: 30,
  isRequired: true,
  requiredForDriverTypes: ["OTR", "Regional"],
  dueDaysAfterAssignment: 30,
  requiresAcknowledgement: true,
  sortOrder: 40,
  openRecordCount: 3,
  version: 2,
  createdAt: 1,
  updatedAt: 2,
};

describe("training course form mapping", () => {
  it("round-trips a server row through the form and back into a mutation input", () => {
    const defaults = buildTrainingCourseDefaults(row);
    expect(defaults.passingScore).toBe("80.00");
    expect(defaults.validityMonths).toBe(36);
    expect(defaults.requiredForDriverTypes).toEqual(["OTR", "Regional"]);

    const parsed = trainingCourseFormSchema.safeParse(defaults);
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);

    const input = toTrainingCourseInput(defaults, row.version);
    expect(input).toMatchObject({
      code: "HAZMAT-AWARE",
      passingScore: "80.00",
      validityMonths: 36,
      requiredForDriverTypes: ["OTR", "Regional"],
      version: 2,
      contentUrl: undefined,
    });
  });

  it("uppercases the code and drops the driver-type filter when the course is optional", () => {
    const defaults = buildTrainingCourseDefaults(null);
    defaults.code = "winter";
    defaults.name = "Winter Driving";
    defaults.isRequired = false;
    defaults.requiredForDriverTypes = ["OTR"];
    const input = toTrainingCourseInput(defaults);
    expect(input.code).toBe("WINTER");
    expect(input.requiredForDriverTypes).toEqual([]);
    expect(input.passingScore).toBeUndefined();
  });

  it("insists on a link for online courses and a sane passing score", () => {
    const base = { ...buildTrainingCourseDefaults(null), code: "ONLINE", name: "Online" };

    const noLink = trainingCourseFormSchema.safeParse({ ...base, delivery: "Online" });
    expect(noLink.success).toBe(false);
    expect(noLink.error?.issues[0]?.path).toEqual(["contentUrl"]);

    const badLink = trainingCourseFormSchema.safeParse({
      ...base,
      delivery: "Online",
      contentUrl: "lms.example.com/course",
    });
    expect(badLink.success).toBe(false);

    const tooHigh = trainingCourseFormSchema.safeParse({ ...base, passingScore: "120" });
    expect(tooHigh.success).toBe(false);
    expect(tooHigh.error?.issues[0]?.path).toEqual(["passingScore"]);

    const fine = trainingCourseFormSchema.safeParse({
      ...base,
      delivery: "Online",
      contentUrl: "https://lms.example.com/course",
      passingScore: "70",
    });
    expect(fine.success, JSON.stringify(fine.error?.issues)).toBe(true);
  });
});
