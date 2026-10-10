# Samsara Simulator

`samsara-sim` is a local simulator for Samsara API behavior used by TMS integration flows.

## Run

```bash
cd services/samsara-sim
task run
```

Server defaults:

- `http://localhost:8091`
- bearer token: `dev-samsara-token`
- health: `GET /_sim/health`
- live map: `GET /_sim/map` (tractors as circles, trailers as squares with
  reefer temperatures; toggle trailers from the toolbar)

## Configure TMS

Point TMS to the simulator:

```yaml
samsara:
  token: dev-samsara-token
  baseURL: http://localhost:8091
```

## Fleet API Surface

Samsara-shaped endpoints (bearer auth required):

- Fleet resources (tags, drivers, driver-vehicle assignments, vehicles and
  vehicle stats, trailers, equipment, assets) are described under
  [Fleet Resources](#fleet-resources).
- `GET /fleet/hos/clocks`, `GET /fleet/hos/logs`
- `GET /fleet/hos/daily-logs` — per-day driver log summaries derived from the
  same deterministic duty timeline as `/fleet/hos/logs` (duty-status durations,
  drive distance, and certification metadata stay mutually consistent with
  clocks and logs). Supports `driverIds`, `startDate`/`endDate` (`YYYY-MM-DD`,
  default: last 7 sim days ending today, max 30-day range), and `after`.
  Records are ordered newest day first per driver.
- `GET /fleet/hos/violations` — derived from deterministic HOS sim events.
  Supports `driverIds`, `types`, `startTime`/`endTime` (default: last 24h of
  sim time), and `after`. Violation types use the real Samsara enum:
  `restbreakMissed`, `shiftHours`, `shiftDrivingHours`, `cycleHoursOn`.
- `GET /fleet/dvirs/history` — deterministic DVIRs derived from the same duty
  timeline as HOS: each driver files a `preTrip` DVIR at their sim workday
  start and a `postTrip` DVIR at workday end, tied to their assigned vehicle.
  Requires `startTime`/`endTime` (RFC3339, max 30-day range, filtered on DVIR
  `endTime`); supports `driverIds`, `vehicleIds`, and `after`/`limit`. Records
  use the real `Dvir` shape: `id`, `type` (`preTrip`/`postTrip`),
  `safetyStatus` (`safe`/`unsafe`/`resolved`, ~10% flagged), `startTime`,
  `endTime`, `odometerMeters` (odometer machinery), `location`,
  `licensePlate`, `driver{id,name}`, `vehicle{id,name}`,
  `authorSignature{signatoryUser{id,name},signedAtTime,type}`, and
  `vehicleDefects[]` (`id`, `defectType`, `comment`, `createdAtTime`,
  `isResolved`, `resolvedAtTime`, `vehicle`) on flagged DVIRs. Defects from
  DVIRs at least 2 sim-days old resolve deterministically
  (`safetyStatus: resolved` + `resolvedAtTime`).
- `GET /form-templates` — five fixture templates with real field shapes:
  Fuel Receipt (`number`/`text`), Incident Report (`text`/`multiple_choice`),
  Trip Inspection Checklist (`check_boxes`), Bill of Lading (Shipper)
  (`formCategory: routing`; `text`/`number`/`multiple_choice`/`signature`
  fields: Seal Number, Pieces Loaded, Gross Weight, Trailer Temperature, Load
  Secured, Shipper Signature, Pickup Notes) and Proof of Delivery (Consignee)
  (`formCategory: routing`; Pieces Delivered, Delivery Temperature, Condition
  on Arrival, Seal Intact, Receiver Name, Receiver Signature, Delivery Notes).
- `GET /form-submissions` — fixture records plus generated submissions from
  the trailing 24h of sim time (`ids` lookups search the trailing 7 sim days).
- `GET /form-submissions/stream` — deterministic submissions generated per
  driver/day (1-2 generic per driver per sim-day, timestamps inside the
  workday; fuel gallons/amounts hashed, checklists mostly passing). Each driver
  also files a Bill of Lading near their workday START (pickup) and a Proof of
  Delivery near their workday END (delivery), tied to that driver. Shipment
  field values are deterministic from `hashFraction`: seal numbers `SL-######`,
  pieces 4-26, gross weight 8000-44000 lbs, reefer temperature 34-38°F, Load
  Secured / Condition on Arrival biased to Yes/Good, and signatures rendered as
  media stubs (`signatureValue.media{id,processingStatus,url,urlExpiresAt}`
  where `urlExpiresAt` is sim-now + 1h). Requires `startTime` (RFC3339,
  `endTime` defaults to now, max 30-day range, filtered on `updatedAtTime`);
  supports `formTemplateIds`, `driverIds`, `userIds`, and `after`.
  Records use the real form-submission shape (`id`, `title`, `status`,
  `isRequired`, `createdAtTime`, `updatedAtTime`, `submittedAtTime`,
  `submittedBy{id,type}`, `formTemplate{id,revisionId}`, `externalIds`,
  `location{latitude,longitude}` from the driver's route position at submit
  time, and — when the driver's route resolves — `routeId` plus `routeStopId`
  (first stop for the BOL pickup, last stop for the POD delivery), with
  `fields[]` carrying
  `numberValue`/`textValue`/`multipleChoiceValue`/`checkBoxesValue`/`signatureValue`).
- `GET /fleet/routes`, `GET /addresses`, `GET /webhooks`

Fixture drivers carry real-shape `eldSettings`
(`{"rulesets":[{"cycle","shift","restart","break","jurisdiction"}]}`) exposed
through `GET /fleet/drivers`: most run `USA 70 hour / 8 day` /
`US Interstate Property`, two run `USA 60 hour / 7 day`, and one runs the
`Texas Intrastate` shift — all with `jurisdiction: TX`.

## Fleet Resources

Every endpoint below implements the query parameters, path parameters, body
fields and response fields of the bundled OpenAPI spec (2025-10-23), with
validation failures returning `400` and unknown objects `404` in the standard
error envelope. Path IDs accept either the Samsara ID or an external ID in
`key:value` form wherever the spec says so (`/tags/{id}`, `/fleet/drivers/{id}`,
`/fleet/vehicles/{id}`, `GET`/`PATCH /fleet/trailers/{id}`, `PATCH /assets?id=`).
Vehicles also resolve the automatically populated `samsara.vin:<VIN>` and
`samsara.serial:<gateway serial>` external IDs, which vehicle responses include
in `externalIds`; client-supplied keys may not use the reserved `samsara.`
prefix. Request fields the spec does not define are ignored and never stored.

### Tags

- `GET /tags` (`limit` 1-512, `after`), `POST /tags`, `GET`/`PATCH`/`PUT`/`DELETE
  /tags/{id}`. Tags carry `parentTagId` + `parentTag`, `externalIds`, and
  member lists (`addresses`, `assets`, `drivers`, `machines`, `sensors`,
  `vehicles`) as `{id, name}` objects. Membership is stored on the tag and is
  also written through the entities' `tagIds` (drivers, vehicles, trailers,
  assets). Vehicles are members through `vehicles`; trailers, equipment,
  unpowered and uncategorized assets through `assets`.
- Validation: `name` 1-191 characters and unique among siblings; the parent
  must exist and may not be the tag itself or one of its descendants; every
  member must exist with the right kind (the organization has no machines or
  sensors, so those lists only accept an empty array); external IDs unique.
- `PATCH` changes only the fields sent (`parentTagId: null` moves a tag to the
  root); `PUT` replaces the parent and every member list (omitted lists become
  empty) and keeps `externalIds`, which its body cannot carry. `DELETE` removes
  the tag and its descendants and clears drivers' group tags that pointed at
  them.
- Tag filters (`tagIds` exact, `parentTagIds` including every descendant, the
  two combined as a union) apply to drivers, vehicles, vehicle stats and
  locations, trailers and trailer stats, equipment and equipment stats and
  locations, assets, the driver-vehicle assignment endpoints and driver
  efficiency/tachograph activity. Unknown tag IDs match nothing.

### Drivers

- `GET /fleet/drivers` — `driverActivationStatus` (default `active`),
  `tagIds`/`parentTagIds`, `attributeValueIds` (every value must match),
  `attributes` (`name:value` or numeric `name:range(min,max)`, repeated or `;`
  separated, all must match), `updatedAfterTime`/`createdAfterTime` (RFC 3339
  with milliseconds and offsets), `limit` 1-512 and `after`.
- `POST /fleet/drivers` — `name`, `username` and `password` are required;
  every UpdateDriverRequest field is validated (lengths, `locale`, `timezone`
  as an IANA zone, `email`, `eldDayStartHour` 0 or 12, `licenseState`,
  `dateOfBirth`, `usDriverRulesetOverride` enums, `carrierSettings`,
  `profileImageBase64` must decode to a JPEG or PNG and becomes
  `profileImageUrl`, referenced tags and `staticAssignedVehicleId` must exist).
  Usernames are unique case-insensitively, `licenseNumber` + `licenseState`
  and every external ID key/value are unique; violations return `400` (the
  spec declares no `409` for these endpoints). The password is validated and
  discarded: it is never stored, returned or sent in a webhook. `eldSettings`
  is derived from `usDriverRulesetOverride` (for example `Texas (7/70)` becomes
  `TX 70 hour / 7 day` with the `Texas Intrastate` shift).
- `GET /fleet/drivers/{id}` and `PATCH /fleet/drivers/{id}` (the full
  UpdateDriverRequest, including `externalIds`, `tagIds`, `password`, and
  `null` to clear optional fields such as `usDriverRulesetOverride`).
  `driverActivationStatus: deactivated` records `deactivatedAtTime` (now, or
  the RFC 3339 value sent, which may not be in the future; sending it for an
  active driver is a `400`). Deactivated drivers drop out of the default
  listing, end their assignments at that time and stop being any vehicle's
  current driver; reactivation clears the time. Responses add the deprecated
  `isDeactivated`, `tags`, the group tags and `staticAssignedVehicle`.
- `GET /beta/fleet/drivers/efficiency` — hour-truncated window (default the
  24 hours before the current hour, at most 31 days, never in the future,
  empty when it starts within the last hour), `driverIds` (exclusive with the
  tag filters and `driverActivationStatus`), `driverTagIds`,
  `driverParentTagIds`, `driverActivationStatus`, `after`. Drive time comes
  from the HOS timeline; distance, fuel, idle, coasting, cruise, green-band,
  torque, over-speed, PTO and brake counts are derived from it per vehicle.
- `GET /fleet/drivers/tachograph-activity/history` — required
  `startTime`/`endTime` (30 days max), `driverIds`, tag filters; contiguous
  `DRIVING`/`WORK`/`BREAK/REST` activity from the duty timeline for the three
  drivers holding a `tachographCardNumber`.
- `POST /fleet/drivers/auth-token` (`code` 12+ characters and one or more of
  `driverId`/`externalId`/`username`, which must agree; deactivated drivers are
  refused; tokens expire after 10 minutes), `POST /fleet/drivers/remote-sign-out`
  (returns `{driverName}` and ends the driver's current driverApp assignment),
  `POST /fleet/drivers/voice-sign-in/resolve-assignment` (matches the spoken
  name against active drivers, `201`, starts a `voiceSignIn` assignment),
  `GET /fleet/drivers/workflows` (`workflowType`, `limit`, `after`) and
  `POST /fleet/drivers/workflow-assignments` (publish/unpublish lists;
  unknown workflows `404`, unknown drivers `400`).
- `DriverCreated` and `DriverUpdated` webhooks carry `{"driver": <Driver>}`.

### Driver-vehicle assignments

Assignments are derived from the live simulation: each driver's paired
vehicle (the same pairing HOS clocks and vehicle events use) gets a `driverApp`
assignment for every on-duty block of their HOS timeline (rest gaps under two
hours stay inside the block), ongoing while the block is in progress.

- `GET /fleet/driver-vehicle-assignments` — `filterBy` (`drivers`/`vehicles`,
  required), `driverIds`/`driverTagIds` only with `filterBy=drivers`,
  `vehicleIds`/`vehicleTagIds` only with `filterBy=vehicles` (IDs or external
  IDs), `startTime`/`endTime` (default now), `assignmentType`, `sourceName`
  (drivers only, exclusive with the other filters, unbounded in time without a
  window), `after`.
- `POST` (`201`), `PATCH` (`202`; identify by `vehicleId` + `driverId` +
  `startTime` or by `metadata.sourceName` alone; `endTime: null` makes it
  ongoing) and `DELETE` (`204`; by vehicle and optional time window,
  `isPassenger` and `assignedAtTime`) manage `external` assignments. An active
  non-passenger API, voice or static assignment re-pairs the live simulation:
  the vehicle's current driver, the driver's HOS `currentVehicle` and vehicle
  events follow it, and other drivers' derived assignments on that vehicle end
  when it starts.
- Legacy `GET /fleet/vehicles/driver-assignments` and
  `GET /fleet/drivers/vehicle-assignments` (7-day window max, tag filters,
  `driverActivationStatus`) return every selected vehicle or driver with its
  `driverApp` assignments.

### Vehicles

- `GET /fleet/vehicles` (`limit`, `after`, tag filters, `attributeValueIds`,
  `attributes` with numeric and date ranges, `updatedAfterTime`,
  `createdAfterTime`), `GET`/`PATCH /fleet/vehicles/{id}`. Vehicles report
  `gateway`, `serial`, `esn`, `cameraSerial`, aux input types,
  `harshAccelerationSettingType`, attributes, tags and `staticAssignedDriver`
  (the vehicle's current driver). `PATCH` validates every UpdateVehicleRequest
  field (VIN 11-17 characters and unique, `licensePlate` 12, `notes` 255, enums,
  gateway serial format); `odometerMeters` and `engineHours` set the manual
  readings behind `gpsOdometerMeters` and `syntheticEngineSeconds`;
  `staticAssignedDriverId` (`null` clears) creates a `static` assignment.
  `VehicleUpdated` carries `{"vehicle": <Vehicle>}`.
- `GET /fleet/vehicles/locations` (`time`, `vehicleIds`, tags), `/feed` and
  `/history` with `reverseGeo.formattedLocation` and speed in mph.
- `GET /fleet/vehicles/immobilizer/stream` — required `vehicleIds` (IDs or
  external IDs) and `startTime`; the three fitted tractors (1001, 1005, 1009)
  report their installation and occasional connection blips with both relays
  closed.
- `GET /fleet/vehicles/stats` (`time`), `/stats/feed` and `/stats/history`
  (`decorations`, at most 2) take a required `types` list of at most 3 types,
  where `auxInput3`-`auxInput10` count as one; every value is checked against
  the spec's 63-value enum and only the requested types are returned. Rows
  carry `id`, `name` and `externalIds`. Diesel tractors report every engine,
  fuel, position, door, seatbelt, fault-code (J1939), NFC card scan
  (`nfcCardScan`/`nfcCardScans`) and immobilizer stat; aux inputs report only
  where an `auxInputTypeN` is configured; EV, spreader and tell-tale types
  return no data, like a diesel truck without that hardware. `gps` includes
  `reverseGeo.formattedLocation` and, inside an address geofence,
  `address {id, name}`. Event-style stats (fault codes, card scans,
  immobilizer) report when they change; the others sample on the 2-minute
  grid. The feed keeps its cursor semantics: the first call returns the
  latest sample per vehicle plus `endCursor`; `after=<cursor>` returns samples
  strictly newer than the cursor; `hasNextPage: true` means page again;
  `false` means caught up and the cursor stays stable. Each response carries at
  most 512 records. History clips the window at the current sim time.

### Trailers

- `GET /fleet/trailers` (tags, `limit`, `after`), `POST`, `GET`/`PATCH`
  (`odometerMeters` sets the manual odometer) and `DELETE /fleet/trailers/{id}`.
  Trailers are assets of type `trailer`, so they also appear in `/assets`
  (`trailerSerialNumber` is the asset's `serialNumber`). No webhook fires: the
  spec's event-type enum has no trailer events.
- `GET /fleet/trailers/stats`, `/feed`, `/history` — `types` (at most 3) from
  the documented list (`gps`, `gpsOdometerMeters`, `carrierReeferState`,
  `reeferAlarms`, `reeferAmbientAirTemperatureMilliC`, `reeferDoorStateZone1-3`,
  `reeferFuelPercent`, `reeferObdEngineSeconds`, `reeferReturnAirTemperatureMilliCZone1-3`,
  `reeferRunMode`, `reeferSetPointTemperatureMilliCZone1-3`,
  `reeferStateZone1-3`, `reeferSupplyAirTemperatureMilliCZone1-3`),
  `decorations` (at most 2), `trailerIds` (IDs or external IDs), tag filters,
  `time` on the snapshot. Every trailer appears on each feed page, with empty
  arrays when a stat has no data. Coupled trailers follow their tractor (a
  hitch length behind it); reefer trailers report their unit's temperatures,
  defrost cycles, fuel and engine hours; multi-zone Carrier units add zone 2
  and `carrierReeferState`.

### Equipment

- `GET /fleet/equipment`, `GET /fleet/equipment/{id}`,
  `GET /fleet/equipment/locations` (+ `/feed`, `/history`),
  `GET /fleet/equipment/stats` (+ `/feed`, `/history`; `types` at most 4 from
  `gatewayEngineStates`, `obdEngineStates`, `fuelPercents`, `engineRpm`,
  `gatewayEngineSeconds`, `obdEngineSeconds`, `gatewayJ1939EngineSeconds`
  (snapshot key `engineSeconds`, AG26 gateways only), `gpsOdometerMeters`,
  `gps`, `engineTotalIdleTimeMinutes`).
- `PATCH /fleet/equipment/{id}/digital-output` — integer ID, `pinId` 1 or 2,
  `state`, `durationSeconds` 0-604800; only the AG53-connected yard spotter
  accepts commands.
- Reefer units ride on their trailers and share the trailer's reefer engine
  hours and fuel; the yard spotter circles the Austin yard Monday to Saturday from 6am
  to 6pm Central and is parked otherwise.

### Assets

- `GET /assets` — `type`, `ids` (IDs and external IDs), `externalIds`
  (`key:value` list), `updatedAfterTime`, tag filters, `attributeValueIds`,
  `attributes` (range queries only, at least one bound), and
  `includeExternalIds`/`includeTags`/`includeAttributes` (omitted unless
  requested). Pages hold 300 assets.
- `POST /assets`, `PATCH /assets?id=` (Samsara or external ID) and
  `DELETE /assets?id=`. `make`, `model` and `year` cannot change on an asset
  with an installed gateway; VINs and external IDs are unique. Changing `type`
  moves the asset between a tag's `vehicles` and `assets`. Deleting a tractor
  leaves its trailer (and a trailer's reefer unit) parked where it was.
- Webhooks follow the spec's event-type enum, which has `VehicleCreated` and
  `VehicleUpdated` but no trailer, asset, tag or delete events:
  `POST /assets` emits `VehicleCreated` only for `type: vehicle`, `PATCH`
  emits `VehicleUpdated` only when the asset is a vehicle afterwards, and
  `DELETE` emits nothing.
- `GET /assets/location-and-speed/stream` includes trailers and equipment
  alongside vehicles.

### Fleet fixtures

- Tag tree (`config/fixtures/default.json`): `Texas Operations` (342401) >
  `Central Texas` (342412: `Austin Terminal` 342423, `San Antonio Terminal`
  342434), `North Texas` (342445: `Dallas-Fort Worth Terminal` 342456),
  `Gulf Coast` (342467: `Houston Terminal` 342478, `Rio Grande Valley Terminal`
  342489), `West Texas` (342500: `El Paso Terminal` 342511, `Permian Basin
  Terminal` 342522, `Panhandle Terminal` 342533); `Fleet Equipment` (342544:
  `Tractors`, `Reefer Trailers`, `Dry Van Trailers`, `Reefer Units`,
  `Yard Equipment`); `Driver Programs` (342610: `Hazmat Certified`,
  `Long Haul`, `Regional`, `Driver Mentors`). Terminals tag their trucks,
  trailers, equipment, drivers and addresses.
- Drivers carry usernames (`arivera`, `jlee`, ...), phones, emails, Texas CDLs,
  time zones (El Paso drivers on `America/Denver`), ELD flags, group tags,
  attributes (CDL class, hazmat endorsement, years of experience, medical card
  expiration), ID card codes for four drivers and tachograph cards for three.
- Tractors gained model years, VG34/VG54NA gateways, engine serials, camera
  serials, aux inputs on three trucks, immobilizers on three, manual odometer
  readings on four and engine-hour readings on two.
- Trailers `Trailer 2041`-`Trailer 2054` (IDs `281474979348595` +
  17 x n, so `Trailer 2042` keeps `281474979348612`): 2041-2052 are coupled to
  Trucks 1001-1012, 2053 is parked at the Austin yard and 2054 at the Dallas
  hub. 2042, 2045, 2049 (two-zone Carrier) and 2053 are reefers; the rest are
  Wabash, Great Dane and Utility dry vans. VINs: 2041 `1JJV532D8KL222089`,
  2042 `1UYVS2534LU829506`, 2043 `1UYVS2533MU197101`, 2044 `1JJV532D9NL717616`,
  2045 `1UYVS2531PU392179`, 2046 `1UYVS2534KU846398`, 2047 `1JJV532D9LL767073`,
  2048 `1GRAA0622MW322884`, 2049 `1GRAA9627NW204626`, 2050 `1JJV532D6PL486872`,
  2051 `1GRAA0620KW547608`, 2052 `1UYVS2535LU909624`, 2053 `1UYVS2534MU246807`,
  2054 `1GRAA0621NW421729`.
- Equipment: `Reefer Unit 2042`/`2045`/`2053` (Thermo King, AG26) and
  `Reefer Unit 2049` (Carrier, AG26) mounted on their trailers
  (`281474980315201`-`281474980315252`), and `Yard Spotter 01`
  (`281474980315269`, Kalmar Ottawa T2, AG53) at the Austin yard.
- Six driver workflows (`Pre-Trip Inspection`, `End of Day Checklist`,
  `Trailer Hook-Up Verification`, `Reefer Pre-Cool Check`,
  `Drop Trailer Checklist`, `Stop Arrival Confirmation`).
- Internal simulation fields (`sim*`: couplings, parked spots, reefer profiles,
  manual readings, digital outputs) are stored with the record but never
  rendered.

### Simulation time and determinism

Vehicle positions, HOS timelines and every derived stat are anchored to the
fixed epoch 2026-01-01T00:00:00Z rather than the process start, so the same sim
time yields the same answer across restarts and between instances (jitter and
speed variance are keyed on the sample time). Odometers, engine hours, fuel
consumed and idle time grow monotonically from that epoch; fuel and DEF levels
follow refill cycles driven by distance. Reverse geocoding uses the address
book inside geofences and a Texas gazetteer elsewhere ("8.2 mi W of Dallas,
TX").

### Reusing the tag filter

Endpoints in later phases filter with
`tagIDs, parentTagIDs := standardTagParams(values)` (or `tagFilterParams` for
differently named parameters), `filter := view.snap.tags.filter(tagIDs,
parentTagIDs)` and `filter.matches(view.snap.tags, tagMembersAddresses, id)`;
`view.snap.tags.tinyTags(kind, id)` renders an entity's `tags`.

## Resource IDs

IDs follow the formats the real API returns, so nothing in TMS has to special-case
the simulator:

| Resource | Format | Fixture example |
| --- | --- | --- |
| Drivers | numeric string | `1654973` (Alex Rivera) |
| Vehicles, trailers, equipment, assets (one shared ID space) | numeric string in the device range | `281474977075805` (Truck 1001), `281474979348612` (Trailer 2042), `281474980315269` (Yard Spotter 01) |
| Tags | numeric string | `342423` (Austin Terminal) |
| Addresses | numeric string | `41226316` (Austin Main Yard) |
| Routes | numeric string | `4129806431` (Texas Route 1) |
| Route stops, DVIRs, DVIR defects | numeric string, derived deterministically | — |
| Webhooks | numeric string | `523918` |
| Users | numeric string | `524871` |
| Form templates, form submissions, tachograph files, driver workflows, attributes and attribute values | UUID | `3f9a6c2e-7b41-4d8a-9e15-c2b7d04f8a61` |
| Live shares | 19 lowercase alphanumerics | `kq3m8vz2pt7xw4nb6rd` |
| Legacy `/v1/fleet/messages` `driverId` | integer (the driver's ID) | `1654973` |

The server always assigns IDs: like the real API, create request bodies carry no
`id`, and one sent anyway is ignored. IDs created through the API are
deterministic: each ID space continues above the highest ID it has ever issued
(deleted records included), so they never collide with fixture records or get
reused, and they repeat after `POST /_sim/state/reset`.

External IDs (`externalIds.tmsVehicleId`, `externalIds.workerId`, …) follow
[Samsara's rules](https://developers.samsara.com/docs/external-ids):

- keys are 1-32 ASCII letters or digits (no `_`, `-` or other punctuation) and
  may not use the reserved `samsara.` prefix;
- values contain only letters, digits and `@ . _ % + -`; numbers and booleans
  are converted to strings, and an empty string removes the key;
- a `key:value` pair is unique across every object of every class (drivers,
  vehicles, trailers, assets, tags, addresses, routes and route stops share one
  namespace), so `maintenance:1234` on a driver blocks `maintenance:1234` on a
  vehicle while `payroll:1234` stays available;
- at most 30 distinct keys are in use per object type across the organization;
  the automatic read-only keys (`samsara.vin`, `samsara.serial` on vehicles and
  trailers, `samsara.name` on tags) do not count and resolve in `key:value`
  path IDs.

Violations return `400`. Attribute IDs are deterministic UUIDs per entity type
and name, and attribute value IDs (used by `attributeValueIds`) are
deterministic UUIDs of the attribute ID and string value. API driver-vehicle
assignments and remote sign-outs get internal numeric IDs that are never
rendered.

Resources whose spec objects carry server timestamps get them from the sim
clock: drivers, assets and form submissions set `createdAtTime` and
`updatedAtTime` on create and bump `updatedAtTime` on every PATCH; addresses set
`createdAtTime` only. Client-supplied values for these fields are ignored.
Routes, webhooks and live shares carry no such fields, matching the spec.

## Errors

Every error uses the Samsara envelope, and nothing else:

```json
{ "message": "Invalid token.", "requestId": "8916e1c1" }
```

| Status | When |
| --- | --- |
| 400 | invalid parameter (`limit` outside the endpoint's range, unknown `after` cursor, malformed or reversed time — `Invalid value for parameter startTime: must be an RFC 3339 timestamp.`, bad body), a referenced tag/driver/vehicle that does not exist, or a uniqueness violation (username, external ID, VIN, license, sibling tag name) — the spec declares no `409` for these endpoints |
| 401 | missing or malformed `Authorization` header, or unknown token (`Invalid token.`) |
| 403 | a read-only token (`dev-samsara-token-readonly`) attempts a write |
| 404 | unknown path (`Not found: no endpoint matches GET /x.`) or unknown object (`Object not found.`) |
| 405 | known path, wrong method (`PUT not allowed on /addresses.`) with an `Allow` header |
| 429 | rate limit exceeded, with a decimal `Retry-After` |

Unknown query parameters are ignored, as the real API does (including
`sortBy`/`sortOrder`, which Samsara does not define: list order is the
endpoint's natural order). Fault-injected `429`s (`/_sim/faults`) carry the
same envelope and decimal `Retry-After` as a real limit: one token's refill
time for the endpoint's level (for example `0.02000` on Legacy Tier 2). The `/_sim/*`
control plane uses the same JSON 404/405 responses.

## Rate Limits

Limits follow [Samsara's documented model](https://developers.samsara.com/docs/rate-limits),
enforced with token buckets on wall-clock time:

- 150 requests/sec per API token;
- 200 requests/sec for the organization, shared by every token (the simulator
  is one organization);
- a per-endpoint limit shared by the organization, keyed by method and route
  (path parameters match any value):

| Level | Limit | Examples |
| --- | --- | --- |
| Level One | 100 requests/min | `POST /addresses`, `PATCH /fleet/routes/{id}`, `POST /v1/fleet/messages` |
| Level Two | 5 requests/sec | `GET /addresses`, `GET /fleet/drivers`, `GET /fleet/routes`, `GET /fleet/hos/logs` |
| Level Three | 10 requests/sec | `GET /fleet/equipment/stats/feed`, `GET /readings/latest` |
| Legacy Tier 1 | 25 requests/sec | `GET /fleet/vehicles`, `GET /fleet/hos/clocks`, `GET /fleet/routes/{id}` |
| Legacy Tier 2 | 50 requests/sec | `GET /fleet/vehicles/stats`, `/stats/feed`, `/stats/history`, `GET /fleet/dvirs/history` |

Endpoints without a documented level fall back to Level Two for reads and
Level One for writes. A limited request gets `429`, the error envelope naming
the limit it hit, and `Retry-After` in decimal seconds until the bucket refills
(for example `0.20000`). No `X-RateLimit-*` headers are sent. `/_sim/*` and
the live map are never limited.

```yaml
rateLimits:
  enabled: true  # default
  multiplier: 1  # scale every limit, e.g. 10 for heavy local development
```

## Pagination

List endpoints return `pagination: {endCursor, hasNextPage}`. `endCursor` is an
opaque token: when `hasNextPage` is `true` it is non-empty and `after=<endCursor>`
resumes exactly after the last record returned, including for records that have
no top-level `id` (HOS clocks, HOS logs, daily logs, asset locations). On the
last page `hasNextPage` is `false` and `endCursor` is `""`. An `after` value the
endpoint did not issue returns `400`. Feed endpoints
(`/fleet/{vehicles,trailers,equipment}/stats/feed`,
`/fleet/{vehicles,equipment}/locations/feed`) keep their own cursor, which
stays valid when caught up.

Page sizes come from the spec:

| Endpoint | `limit` | Page size |
| --- | --- | --- |
| `GET /addresses`, `/fleet/drivers`, `/fleet/drivers/workflows`, `/fleet/equipment`, `/fleet/routes`, `/fleet/trailers`, `/fleet/vehicles`, `/tags`, `/fleet/dvirs/history`, `/fleet/hos/clocks`, `/webhooks`, `/assets/location-and-speed/stream` | 1-512, default 512 | `limit` |
| `GET /live-shares` | 1-100, default 100 | `limit` |
| `GET /assets` | not accepted | 300 |
| `GET /fleet/hos/logs`, `/fleet/hos/daily-logs`, `/fleet/hos/violations`, `/fleet/{vehicles,trailers,equipment}/stats` (+ `/feed`, `/history`), `/fleet/{vehicles,equipment}/locations` (+ `/feed`, `/history`), `/fleet/vehicles/immobilizer/stream`, `/fleet/driver-vehicle-assignments`, `/fleet/{vehicles,drivers}/{driver,vehicle}-assignments`, `/beta/fleet/drivers/efficiency`, `/fleet/drivers/tachograph-activity/history`, `/fleet/{drivers,vehicles}/tachograph-files/history`, `/form-templates`, `/form-submissions/stream` | not accepted | 512 |

Where an endpoint declares `limit`, a value outside its range returns `400`;
where it does not, `limit` is ignored.

## State Persistence

API changes (drivers, addresses, routes, assets including trailers and
equipment, tags, driver-vehicle assignments, remote sign-outs, workflow
publications, webhooks and their URLs, live shares, form submissions, messages)
and the ID counters can survive restarts:

```yaml
state:
  path: ./data/state.json  # empty disables persistence
  flushDelay: 250ms        # bursts of writes inside this window become one write
```

Every successful mutation schedules a write; the file is replaced atomically
(temporary file, fsync, rename, directory fsync) and flushed again on shutdown.
At startup the file, when present, replaces the fixture records; asset
locations are always rebuilt from the fixture and route dataset. A corrupt or
unreadable file stops the simulator with an error naming the file rather than
silently starting from fixtures; move it aside to start fresh. When the
fixture has changed since the file was written, a warning says so and the
persisted records still win; collections the file does not contain at all
(for example `tags` in a file written before tags existed) start from the
fixture. Writes are transactional: a request that fails validation part-way
leaves no partial change and consumes no ID. `POST /_sim/state/reset` restores the fixtures and
deletes the file. `GET /_sim/state/summary` reports the path and write count.

`config.example.yaml` enables persistence at `./data/state.json`, which is
`/app/data/state.json` in the container; `docker-compose-local.yml` mounts the
`samsara_sim_data` volume there, so rebuilding the container keeps the IDs TMS
has stored. To start over, call `POST /_sim/state/reset`, or stop the
container and remove only that volume (`docker volume rm trenova_samsara_sim_data`
with the default project name).

## Geofences

Fixture addresses carry real-shape geofence data
(`{"geofence":{"circle":{"latitude":..,"longitude":..,"radiusMeters":..}}}`),
including circles placed directly on the Texas OSM route geometry so vehicles
deterministically pass through them. Entry/exit transitions are evaluated at
the 2-minute sample cadence during stats/feed polls and emit `GeofenceEntry` /
`GeofenceExit` webhooks with the real address + vehicle payload (including
`vin` and `licensePlate`). `GET /_sim/state/summary` reports dispatched
geofence transition counts.

## Webhooks

All outbound webhooks use the real Samsara Webhooks 2.0 envelope:

```json
{
  "eventId": "<deterministic uuid>",
  "eventTime": "2026-03-01T14:00:00.000Z",
  "eventType": "GeofenceEntry",
  "orgId": 20936,
  "webhookId": "523918",
  "data": {}
}
```

Emitted event types are real Samsara types only: `SpeedingEventStarted` /
`SpeedingEventEnded`, `SevereSpeedingStarted` / `SevereSpeedingEnded`,
`AlertIncident` (HOS violations), `RouteStopEtaUpdated` (traffic delays),
`GeofenceEntry` / `GeofenceExit`, `RouteStopArrival` / `RouteStopDeparture`
(route-stop tracking), `DvirSubmitted` (DVIR completions:
`{driver, vehicle, dvir{...defects}}`), `FormSubmitted` / `FormUpdated`
(`{form: {...}}`), and resource CRUD events (`AddressCreated`, ...;
`DriverCreated`/`DriverUpdated` as `{driver: <Driver>}` and
`VehicleCreated`/`VehicleUpdated` as `{vehicle: <Vehicle>}` for vehicle assets
only — see [Fleet Resources](#fleet-resources)).
`RouteStopArrival` / `RouteStopDeparture`, `DvirSubmitted` and `FormSubmitted`
fire lazily during stats polls when a route-stop crossing, DVIR `endTime`, or
submission `submittedAtTime` falls inside the dispatch window, with the same
deduplication as geofence events.

`RouteStopArrival` / `RouteStopDeparture` reuse the geofence machinery: when a
vehicle that is assigned to a route crosses an address geofence, the entry
emits a `RouteStopArrival` (aligned with `GeofenceEntry`) and the exit emits a
`RouteStopDeparture` (aligned with `GeofenceExit`). The `data` payload follows
the real route-tracking shape: `{operation ("stop arrived"/"stop departed"),
type: "route tracking", time, assignedToRoute, driver{id,name,externalIds},
vehicle{id,name,assetType,licensePlate,vin,externalIds},
route{id,name,externalIds}, routeStopDetails{id, state ("arrived"/"departed"),
eta, enRouteTime, actualArrivalTime, actualDepartureTime, externalIds,
orders[]}}`.

Signing follows the real Samsara scheme:

- headers: `X-Samsara-Timestamp` (RFC3339) and `X-Samsara-Signature: v1=<hex>`
- signed string: `"v1:" + timestamp + ":" + rawBody`
- key: the webhook `secretKey` (or global `webhooks.signingSecret`) is treated
  as base64; it is base64-STD-decoded to raw key bytes before HMAC-SHA256, and
  used as raw bytes when it is not valid base64.

Webhook records expose their `secretKey` in `GET /webhooks` responses, and a
deterministic base64 `secretKey` is generated when a webhook is created
without one. The simulator also keeps its `X-Samsara-Sim-Delivery-*` headers
for duplicate/reorder/attempt introspection.

## Scenarios

Default scenario profile:

- `default`: full payload fidelity
- `sparse`: omit optional fields
- `partial`: partial collections and optional field omissions
- `degraded`: partial + omissions + deterministic 503 responses

Webhook event omission behavior:

- `default`: no event omission
- `sparse`: omits a meaningful portion of webhook events
- `partial`: omits many webhook events
- `degraded`: omits most webhook events

Override per request:

- header: `X-Samsara-Sim-Profile: sparse`

Control plane:

- `GET /_sim/scenarios`
- `GET /_sim/scenarios/active`
- `PUT /_sim/scenarios/active` with `{"profile":"degraded"}`
- `GET /_sim/time`
- `PUT /_sim/time`
- `POST /_sim/time/step`
- `GET /_sim/scripts/status`
- `GET /_sim/faults`
- `PUT /_sim/faults`
- `POST /_sim/faults/rules`
- `DELETE /_sim/faults/rules/{id}`
- `POST /_sim/faults/reset`
- `GET /_sim/state/summary`
- `POST /_sim/state/reset`
- `POST /_sim/events/trigger`
- `GET /_sim/events/active`
- `GET /_sim/events/window`

## Docker

Start simulator container from repository root:

```bash
docker compose -f docker-compose-local.yml --profile samsara-sim up -d samsara-sim
```

## Route Dataset

The simulator ships with a Texas route dataset at:

- `config/datasets/texas_osm_routes.geojson`

This dataset is derived from OpenStreetMap road geometry via the OSRM demo server and is used to seed realistic asset waypoints.
It now includes 12 long-haul corridors for multi-hour travel simulation.

Refresh the dataset:

```bash
cd services/samsara-sim
task routes-fetch
```

Disable or override route dataset loading in config:

```yaml
seed:
  routeDatasetPath: ./config/datasets/texas_osm_routes.geojson
```

## Simulation Controls

Tune long-haul behavior and event realism:

```yaml
simulation:
  fleetSize: 12
  tripHoursMin: 8  # minimum simulated loop duration for moving vehicles
  tripHoursMax: 12 # deterministic target band used for short-route stretching
  eventIntensity: balanced # balanced|compliance|driving
  violationRate: 0.08
  speedingRate: 0.14
  scriptPath: ./config/scenarios/default.yaml
  scriptMode: merge # merge|override
  scriptTimezone: UTC
```

Operational events are deterministic by seed and include:

- duty transitions (`offDuty`, `sleeperBerth`)
- stop/delay periods
- speeding bursts
- HOS violation windows

Events affect API state (`/fleet/vehicles/stats`, `/fleet/hos/clocks`, `/fleet/hos/logs`)
and are emitted as webhooks when webhooks are enabled.

Movement loop timing uses `tripHoursMin`/`tripHoursMax` to stretch short routes so
vehicles do not complete unrealistically short loops.

Simulation time is virtual and controllable via `/_sim/time`:

- pause and resume
- set explicit simulation timestamp
- step simulation forward deterministically

Scenario scripts are loaded from YAML and merged or overridden per `simulation.scriptMode`.
Rule-based fault injection for endpoints and webhook event delivery is available through `/_sim/faults`.

## Smoke Test

Run HOS and asset-location smoke checks:

```bash
cd services/samsara-sim
task smoke-sim
```

Run event and correlation checks:

```bash
cd services/samsara-sim
task smoke-events
```

Run time-control checks:

```bash
cd services/samsara-sim
task smoke-time
```

Run fault-injection checks:

```bash
cd services/samsara-sim
task smoke-faults
```

Run CI smoke suite (health, route lifecycle, moving GPS, HOS delta, webhook inbox, rate-limit 429 envelope and decimal `Retry-After`):

```bash
cd services/samsara-sim
task smoke-ci
```

Optional overrides:

- `SIM_BASE_URL` (default: `http://localhost:8091`)
- `SIM_TOKEN` (default: `dev-samsara-token`)
