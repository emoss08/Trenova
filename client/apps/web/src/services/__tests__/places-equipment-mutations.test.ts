import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Carrier } from "@trenova/shared/types/carrier";
import type { Location } from "@trenova/shared/types/location";
import type { RateZone } from "@trenova/shared/types/rate";
import type { Worker } from "@trenova/shared/types/worker";
import type { CustomFieldDefinition } from "@/types/custom-field";
import type { DistanceOverride } from "@/types/distance-override";
import type { SequenceConfigDocument } from "@/types/sequence-config";
import type { ServiceFailureReasonCode } from "@/types/service-failure-reason-code";
import { fetchLocationGeofences } from "@/lib/graphql/location-geofences";
import { CarrierService } from "../carrier";
import { CustomFieldService } from "../custom-field";
import { DistanceOverrideService } from "../distance-override";
import { LocationService } from "../location";
import { RateZoneService } from "../rate";
import { SequenceConfigService } from "../sequence-config";
import { ServiceFailureReasonCodeService } from "../service-failure-reason-code";
import { StoredMileageService } from "../stored-mileage";
import { TrailerService } from "../trailer";
import { WorkerService } from "../worker";

const requestGraphQL = vi.fn();

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: (...args: unknown[]) => requestGraphQL(...args),
}));

function sentRequest() {
  expect(requestGraphQL).toHaveBeenCalledTimes(1);
  return requestGraphQL.mock.calls[0]?.[0] as {
    operationName: string;
    variables: Record<string, unknown>;
    signal?: AbortSignal;
  };
}

beforeEach(() => {
  requestGraphQL.mockReset();
});

describe("LocationService", () => {
  const location = {
    status: "Active",
    code: "",
    name: "Columbus DC",
    locationCategoryId: "lc_1",
    description: null,
    addressLine1: "1 Main St",
    addressLine2: null,
    city: "Columbus",
    stateId: "us_oh",
    postalCode: "43004",
    timezone: "America/New_York",
    isGeocoded: true,
    longitude: -82.9,
    latitude: 39.9,
    placeId: null,
    geofenceType: "draw",
    geofenceRadiusMeters: null,
    geofenceVertices: [
      { latitude: 39.9, longitude: -82.9 },
      { latitude: 39.91, longitude: -82.9 },
      { latitude: 39.91, longitude: -82.91 },
    ],
    version: 4,
  } as Location;

  it("updates the location it names with every field, leaving a blank code for the sequence", async () => {
    requestGraphQL.mockResolvedValue({ updateLocation: { id: "loc_1", version: 5 } });

    await expect(new LocationService().update("loc_1", location)).resolves.toEqual({
      id: "loc_1",
      version: 5,
    });

    const request = sentRequest();
    expect(request.operationName).toBe("UpdateLocation");
    expect(request.variables).toEqual({
      id: "loc_1",
      input: {
        status: "Active",
        code: null,
        name: "Columbus DC",
        description: null,
        locationCategoryId: "lc_1",
        stateId: "us_oh",
        addressLine1: "1 Main St",
        addressLine2: null,
        city: "Columbus",
        postalCode: "43004",
        timezone: "America/New_York",
        placeId: null,
        isGeocoded: true,
        longitude: -82.9,
        latitude: 39.9,
        geofenceType: "draw",
        geofenceRadiusMeters: null,
        geofenceVertices: [
          { latitude: 39.9, longitude: -82.9 },
          { latitude: 39.91, longitude: -82.9 },
          { latitude: 39.91, longitude: -82.91 },
        ],
        version: 4,
      },
    });
  });

  it("patches only the fields it is given", async () => {
    requestGraphQL.mockResolvedValue({ patchLocation: { id: "loc_1", status: "Inactive" } });

    await new LocationService().patch("loc_1", { status: "Inactive" });

    expect(sentRequest()).toMatchObject({
      operationName: "PatchLocation",
      variables: { id: "loc_1", input: { status: "Inactive" } },
    });
  });
});

describe("fetchLocationGeofences", () => {
  it("asks for the first locations, forwards the signal and returns the nodes", async () => {
    const node = {
      id: "loc_1",
      code: "CMH",
      name: "Columbus DC",
      description: "",
      addressLine1: "1 Main St",
      addressLine2: "",
      city: "Columbus",
      postalCode: "43004",
      latitude: 39.9,
      longitude: -82.9,
      geofenceType: "circle",
      geofenceRadiusMeters: 250,
      geofenceVertices: [],
      state: { id: "us_oh", abbreviation: "OH" },
    };
    requestGraphQL.mockResolvedValue({ locations: { edges: [{ node }] } });
    const controller = new AbortController();

    await expect(fetchLocationGeofences(100, { signal: controller.signal })).resolves.toEqual([
      node,
    ]);

    const request = sentRequest();
    expect(request.operationName).toBe("LocationGeofences");
    expect(request.variables).toEqual({ first: 100 });
    expect(request.signal).toBe(controller.signal);
  });
});

describe("TrailerService.locate", () => {
  it("sends the trailer and the location it now sits at", async () => {
    requestGraphQL.mockResolvedValue({ locateTrailer: { id: "eqc_1" } });

    await new TrailerService().locate("tr_1", { newLocationId: "loc_2" });

    expect(sentRequest()).toMatchObject({
      operationName: "LocateTrailer",
      variables: { input: { trailerId: "tr_1", newLocationId: "loc_2" } },
    });
  });
});

describe("CarrierService", () => {
  const carrier = {
    status: "Active",
    code: "SWFT",
    name: "Swift Haulers",
    dbaName: null,
    carrierType: "Common",
    dotNumber: "1234567",
    mcNumber: null,
    scac: "SWFT",
    complianceStatus: "Qualified",
    safetyRating: "Satisfactory",
    qualifiedAt: 1_700_000_000,
    disqualifiedReason: null,
    taxId: "12-3456789",
    taxIdType: "EIN",
    w9OnFile: true,
    is1099Eligible: false,
    paymentMethod: "ACHManual",
    paymentTermDays: 15,
    remitToName: null,
    remitAddressLine1: null,
    remitAddressLine2: null,
    remitCity: null,
    remitStateId: null,
    remitPostalCode: null,
    addressLine1: "9 Dock Rd",
    addressLine2: null,
    city: "Dayton",
    stateId: "us_oh",
    postalCode: "45402",
    phone: null,
    email: "dispatch@swift.test",
    externalId: null,
    notes: null,
    version: 2,
    contacts: [
      {
        id: "carc_1",
        name: "Dana Reyes",
        title: "Dispatcher",
        email: "dana@swift.test",
        phone: null,
        isPrimary: true,
        receivesRateConfirmations: true,
        version: 1,
      },
      {
        name: "Lee Park",
        title: null,
        email: null,
        phone: "555-0100",
        isPrimary: false,
        receivesRateConfirmations: false,
      },
    ],
    insurancePolicies: [
      {
        id: "carins_1",
        policyType: "AutoLiability",
        policyNumber: "AL-1",
        providerName: "Shield Mutual",
        coverageAmount: 1000000,
        effectiveDate: 1_700_000_000,
        expirationDate: 1_731_536_000,
        isVerified: true,
        version: 3,
      },
    ],
  } as Carrier;

  it("sends the contacts and policies it keeps, with saved ones by id and decimals as text", async () => {
    requestGraphQL.mockResolvedValue({ updateCarrier: { id: "car_1", version: 3 } });

    await new CarrierService().update("car_1", carrier);

    const request = sentRequest();
    expect(request.operationName).toBe("UpdateCarrier");
    const input = request.variables.input as Record<string, unknown>;
    expect(input.contacts).toEqual([
      {
        id: "carc_1",
        name: "Dana Reyes",
        title: "Dispatcher",
        email: "dana@swift.test",
        phone: null,
        isPrimary: true,
        receivesRateConfirmations: true,
        version: 1,
      },
      {
        id: null,
        name: "Lee Park",
        title: null,
        email: null,
        phone: "555-0100",
        isPrimary: false,
        receivesRateConfirmations: false,
        version: undefined,
      },
    ]);
    expect(input.insurancePolicies).toEqual([
      {
        id: "carins_1",
        policyType: "AutoLiability",
        policyNumber: "AL-1",
        providerName: "Shield Mutual",
        coverageAmount: "1000000",
        effectiveDate: 1_700_000_000,
        expirationDate: 1_731_536_000,
        isVerified: true,
        version: 3,
      },
    ]);
    expect(input).toMatchObject({
      code: "SWFT",
      taxIdType: "EIN",
      paymentTermDays: 15,
      version: 2,
    });
    expect(input).not.toHaveProperty("ediChannels");
  });

  it("reads one carrier with its contacts and policies by id", async () => {
    requestGraphQL.mockResolvedValue({ carrier: { ...carrier, id: "car_1" } });

    const read = await new CarrierService().getById("car_1");

    expect(sentRequest()).toMatchObject({
      operationName: "CarrierRecord",
      variables: { id: "car_1" },
    });
    expect(read.contacts).toHaveLength(2);
    expect(read.insurancePolicies[0]?.policyNumber).toBe("AL-1");
  });

  it("refuses a carrier the server does not return", async () => {
    requestGraphQL.mockResolvedValue({ carrier: null });

    await expect(new CarrierService().getById("car_404")).rejects.toThrow("Carrier not found");
  });
});

describe("WorkerService", () => {
  const worker = {
    stateId: "us_oh",
    fleetCodeId: null,
    managerId: "usr_9",
    positionId: "jp_1",
    status: "Active",
    type: "Employee",
    driverType: "OTR",
    gender: "Female",
    firstName: "Ana",
    lastName: "Lopez",
    addressLine1: "2 Elm St",
    addressLine2: null,
    city: "Akron",
    postalCode: "44301",
    email: null,
    phoneNumber: "555-0101",
    emergencyContactName: null,
    emergencyContactPhone: "",
    externalId: null,
    canBeAssigned: true,
    availableForDispatch: true,
    customFields: { shift: "night" },
    version: 7,
    profile: {
      licenseStateId: "us_oh",
      dob: undefined,
      licenseNumber: undefined,
      cdlClass: "A",
      cdlRestrictions: null,
      endorsement: "N",
      hazmatExpiry: null,
      licenseExpiry: 1_900_000_000,
      medicalCardExpiry: 1_800_000_000,
      medicalExaminerName: null,
      medicalExaminerNpi: null,
      twicCardNumber: null,
      twicExpiry: null,
      hireDate: 1_600_000_000,
      terminationDate: null,
      physicalDueDate: null,
      mvrDueDate: null,
      complianceStatus: "Compliant",
      isQualified: true,
      disqualificationReason: null,
      eldExempt: false,
      shortHaulExempt: true,
    },
  } as unknown as Worker;

  it("leaves out the personal fields a caller cannot see so the saved values stay", async () => {
    requestGraphQL.mockResolvedValue({ updateWorker: { id: "wrk_1", version: 8 } });

    await new WorkerService().update("wrk_1", worker);

    const request = sentRequest();
    expect(request.operationName).toBe("UpdateWorker");
    const input = request.variables.input as { profile: Record<string, unknown> };
    expect(request.variables.id).toBe("wrk_1");
    expect(input).toMatchObject({
      managerId: "usr_9",
      positionId: "jp_1",
      phoneNumber: "555-0101",
      emergencyContactPhone: null,
      customFields: { shift: "night" },
      version: 7,
    });
    expect(input.profile).toMatchObject({
      dob: null,
      licenseNumber: null,
      licenseExpiry: 1_900_000_000,
      hireDate: 1_600_000_000,
      shortHaulExempt: true,
    });
    expect(input.profile).not.toHaveProperty("trainingHealth");
    expect(input.profile).not.toHaveProperty("lastComplianceCheck");
  });

  it("refuses an update that names no worker", async () => {
    await expect(new WorkerService().update(undefined, worker)).rejects.toThrow(
      "No worker selected",
    );
    expect(requestGraphQL).not.toHaveBeenCalled();
  });
});

describe("RateZoneService", () => {
  const zone = {
    code: "SE",
    name: "Southeast",
    description: "",
    kind: "Regional",
    status: "Active",
    version: 1,
    members: [
      { id: "rzm_1", scopeType: "State", scopeValue: "us_ga", city: "" },
      { scopeType: "CityState", scopeValue: "us_fl", city: "Miami" },
    ],
  } as RateZone;

  it("sends every member, keeping saved ones by id", async () => {
    requestGraphQL.mockResolvedValue({ updateRateZone: { id: "rzn_1", version: 2 } });

    await new RateZoneService().update("rzn_1", zone);

    expect(sentRequest()).toEqual(
      expect.objectContaining({
        operationName: "UpdateRateZone",
        variables: {
          id: "rzn_1",
          input: {
            code: "SE",
            name: "Southeast",
            description: null,
            kind: "Regional",
            status: "Active",
            version: 1,
            members: [
              { id: "rzm_1", scopeType: "State", scopeValue: "us_ga", city: null },
              { id: null, scopeType: "CityState", scopeValue: "us_fl", city: "Miami" },
            ],
          },
        },
      }),
    );
  });

  it("reads a zone with its members", async () => {
    requestGraphQL.mockResolvedValue({
      rateZone: { id: "rzn_1", ...zone, members: [{ ...zone.members[0], matchKey: "ST:us_ga" }] },
    });

    const read = await new RateZoneService().getById("rzn_1");

    expect(sentRequest()).toMatchObject({
      operationName: "RateZoneRecord",
      variables: { id: "rzn_1" },
    });
    expect(read.members).toEqual([
      { id: "rzm_1", scopeType: "State", scopeValue: "us_ga", city: "", matchKey: "ST:us_ga" },
    ]);
  });
});

describe("CustomFieldService", () => {
  it("sends the options, rules and attributes of a definition", async () => {
    requestGraphQL.mockResolvedValue({ createCustomFieldDefinition: { id: "cfd_1", version: 0 } });

    await new CustomFieldService().create({
      resourceType: "trailer",
      name: "door_type",
      label: "Door type",
      description: "",
      fieldType: "select",
      isRequired: true,
      isActive: true,
      displayOrder: 2,
      color: "",
      options: [{ value: "swing", label: "Swing", color: "", description: "Two doors" }],
      validationRules: { minLength: null, maxLength: 20, min: null, max: null, pattern: "" },
      defaultValue: "swing",
      uiAttributes: null,
    } as CustomFieldDefinition);

    expect(sentRequest()).toMatchObject({
      operationName: "CreateCustomFieldDefinition",
      variables: {
        input: {
          resourceType: "trailer",
          name: "door_type",
          description: null,
          color: null,
          options: [{ value: "swing", label: "Swing", color: null, description: "Two doors" }],
          validationRules: { minLength: null, maxLength: 20, min: null, max: null, pattern: null },
          defaultValue: "swing",
          uiAttributes: null,
        },
      },
    });
  });

  it("deletes a definition by id", async () => {
    requestGraphQL.mockResolvedValue({ deleteCustomFieldDefinition: true });

    await new CustomFieldService().delete("cfd_1");

    expect(sentRequest()).toMatchObject({
      operationName: "DeleteCustomFieldDefinition",
      variables: { id: "cfd_1" },
    });
  });
});

describe("DistanceOverrideService", () => {
  it("sends the intermediate stops in route order", async () => {
    requestGraphQL.mockResolvedValue({ createDistanceOverride: { id: "do_1", version: 0 } });

    await new DistanceOverrideService().create({
      originLocationId: "loc_1",
      destinationLocationId: "loc_9",
      customerId: null,
      distance: 412.5,
      intermediateStops: [{ locationId: "loc_4" }, { locationId: "loc_2" }],
    } as DistanceOverride);

    expect(sentRequest()).toMatchObject({
      operationName: "CreateDistanceOverride",
      variables: {
        input: {
          originLocationId: "loc_1",
          destinationLocationId: "loc_9",
          customerId: null,
          distance: 412.5,
          intermediateStops: [{ locationId: "loc_4" }, { locationId: "loc_2" }],
        },
      },
    });
  });
});

describe("ServiceFailureReasonCodeService", () => {
  it("archives and activates by id", async () => {
    requestGraphQL.mockResolvedValue({ archiveServiceFailureReasonCode: { id: "sfrc_1" } });
    await new ServiceFailureReasonCodeService().archive("sfrc_1");
    expect(sentRequest()).toMatchObject({
      operationName: "ArchiveServiceFailureReasonCode",
      variables: { id: "sfrc_1" },
    });

    requestGraphQL.mockReset();
    requestGraphQL.mockResolvedValue({ activateServiceFailureReasonCode: { id: "sfrc_1" } });
    await new ServiceFailureReasonCodeService().activate("sfrc_1");
    expect(sentRequest()).toMatchObject({
      operationName: "ActivateServiceFailureReasonCode",
      variables: { id: "sfrc_1" },
    });
  });

  it("sends blank defaults as null", async () => {
    requestGraphQL.mockResolvedValue({ updateServiceFailureReasonCode: { id: "sfrc_1" } });

    await new ServiceFailureReasonCodeService().update("sfrc_1", {
      code: "WX",
      label: "Weather",
      description: "",
      category: "Weather",
      appliesTo: "Both",
      defaultStatusCode: "SD",
      defaultReasonCode: "",
      defaultExceptionCode: "",
      defaultNote: "",
      active: true,
      sortOrder: 10,
      externalMap: null,
      version: 2,
    } as ServiceFailureReasonCode);

    expect(sentRequest().variables).toEqual({
      id: "sfrc_1",
      input: {
        code: "WX",
        label: "Weather",
        description: null,
        category: "Weather",
        appliesTo: "Both",
        defaultStatusCode: "SD",
        defaultReasonCode: null,
        defaultExceptionCode: null,
        defaultNote: null,
        active: true,
        sortOrder: 10,
        externalMap: null,
        version: 2,
      },
    });
  });
});

describe("StoredMileageService", () => {
  it("deactivates a stored mileage by id", async () => {
    requestGraphQL.mockResolvedValue({ deactivateStoredMileage: true });

    await new StoredMileageService().deactivate("sm_1");

    expect(sentRequest()).toMatchObject({
      operationName: "DeactivateStoredMileage",
      variables: { id: "sm_1" },
    });
  });
});

describe("SequenceConfigService", () => {
  it("reads the sequences with the caller's signal", async () => {
    requestGraphQL.mockRejectedValue(new Error("offline"));
    const controller = new AbortController();

    await expect(new SequenceConfigService().get({ signal: controller.signal })).rejects.toThrow(
      "offline",
    );

    const request = sentRequest();
    expect(request.operationName).toBe("SequenceConfigs");
    expect(request.signal).toBe(controller.signal);
  });

  it("sends each sequence with its location code strategy", async () => {
    requestGraphQL.mockRejectedValue(new Error("stop after send"));
    const doc = {
      configs: [
        {
          id: "sc_1",
          sequenceType: "location_code",
          prefix: "LOC",
          includeYear: false,
          yearDigits: 2,
          includeMonth: false,
          includeWeekNumber: false,
          includeDay: false,
          sequenceDigits: 3,
          includeLocationCode: false,
          includeRandomDigits: false,
          randomDigitsCount: 0,
          includeCheckDigit: false,
          includeBusinessUnitCode: false,
          useSeparators: true,
          separatorChar: "-",
          allowCustomFormat: false,
          customFormat: "",
          locationCodeStrategy: {
            components: ["name", "city"],
            componentWidth: 3,
            sequenceDigits: 3,
            separator: "-",
            casing: "upper",
            fallbackPrefix: "LOC",
          },
          version: 5,
        },
      ],
    } as SequenceConfigDocument;

    await expect(new SequenceConfigService().update(doc)).rejects.toThrow("stop after send");

    const request = sentRequest();
    expect(request.operationName).toBe("UpdateSequenceConfigs");
    expect(request.variables).toEqual({
      input: {
        configs: [
          {
            id: "sc_1",
            sequenceType: "location_code",
            prefix: "LOC",
            includeYear: false,
            yearDigits: 2,
            includeMonth: false,
            includeWeekNumber: false,
            includeDay: false,
            sequenceDigits: 3,
            includeLocationCode: false,
            includeRandomDigits: false,
            randomDigitsCount: 0,
            includeCheckDigit: false,
            includeBusinessUnitCode: false,
            useSeparators: true,
            separatorChar: "-",
            allowCustomFormat: false,
            customFormat: "",
            locationCodeStrategy: {
              components: ["name", "city"],
              componentWidth: 3,
              sequenceDigits: 3,
              separator: "-",
              casing: "upper",
              fallbackPrefix: "LOC",
            },
            version: 5,
          },
        ],
      },
    });
  });
});
