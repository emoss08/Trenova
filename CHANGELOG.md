# Changelog

All notable changes to Trenova will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- GraphQL API.
- Conditionals in Table Change Alerts (https://github.com/emoss08/Trenova/issues/203).
- Worker on-boarding Client.
- Worker Scorecards.
- Color Accessibility Options to User Settings.
- Font Size Switcher to User Settings.

### Fixed

- N+1 queries in Notifications API (https://github.com/emoss08/Trenova/issues/196).

## [0.1.0] - 2024-02-21

### Added
- `equipment_class` field filter for `Tractor` and `Trailer` api (https://github.com/emoss08/Trenova/pull/227)
- Tooltip for `PasswordField` (https://github.com/emoss08/Trenova/pull/227)
- Date time picker field (https://github.com/emoss08/Trenova/pull/227)
- Async Select Input field (https://github.com/emoss08/Trenova/pull/227)
- `Add new shipment` page (https://github.com/emoss08/Trenova/pull/227)
- Pyinstrument middleware for development (https://github.com/emoss08/Trenova/pull/227)
- Beam Integration (https://github.com/emoss08/TrenovaPrivate/pull/10)
- Anomaly Detection (https://github.com/emoss08/TrenovaPrivate/pull/13)
- API for next shipment `pro_number` (https://github.com/emoss08/Trenova/pull/227)
- `Severity` level to comment type (https://github.com/emoss08/Trenova/pull/227)

### Removed
- Python `Kafka` listener implementation (https://github.com/emoss08/Trenova/pull/227)
- `CommentType` field from `StopComment` model (https://github.com/emoss08/Trenova/pull/227)

### Changed
- Changed `Kafka` listener implementation from Python to Java (https://github.com/emoss08/Trenova/pull/227)
- `Lucide React Icons` to `FontAwesomeIcons` (https://github.com/emoss08/Trenova/pull/227)
- `Stop Comment` api queryset to include all fields (https://github.com/emoss08/Trenova/pull/227)
- `Comment` field to `Value` on `StopComment` model (https://github.com/emoss08/Trenova/pull/227)
- `Watch` from `React-hook-form` to subscriptions (https://github.com/emoss08/Trenova/pull/227)
- `Intregation Auth Types` to be more concise (https://github.com/emoss08/Trenova/pull/227)
- `useClickOutside` hook to react-aria `useInteractOutside` (https://github.com/emoss08/Trenova/pull/227)
- Styling of the `DeliverySlot` component (https://github.com/emoss08/Trenova/pull/227)

### Fixed
- Reduced bundle size by 50% (https://github.com/emoss08/Trenova/pull/227)
- Spacing on Filter Options (https://github.com/emoss08/Trenova/pull/227)
- Badge styling on `light mode` (https://github.com/emoss08/Trenova/pull/227)
- Password bubble being to big (https://github.com/emoss08/Trenova/pull/227)
- `useCustomMutation` hook to better handle file uploads (https://github.com/emoss08/Trenova/pull/227)
- `AuditLog` api returning incorrect results based on `content_type_id` (https://github.com/emoss08/Trenova/pull/227)
- `ColorFIeld` styles to be consistent with other fields (https://github.com/emoss08/Trenova/pull/227/commits/396638e1879fb85a1097f3b0254318b933751f6d)

### Security
- Fix security vulnerabilities where exceptions are returned in the response (https://github.com/emoss08/Trenova/pull/227)
- Fix permissive regular expression when validating vin numbers (https://github.com/emoss08/Trenova/pull/227)


## [0.0.6] - 2024-01-28

### Added
- Add Sheet to `Table Change Alert` page (https://github.com/emoss08/Trenova/pull/224)

## [0.0.5] - 2024-01-27

### Added
- Add `Applicaiton Favorites` which allows users to favorite pages (https://github.com/emoss08/Trenova/pull/222)
- Add idempotency middleware to DRF API (https://github.com/emoss08/Trenova/pull/220)

## [0.0.4] - 2024-01-25

### Added
- Add `conditionals` to `Table Change Alerts` (https://github.com/emoss08/Trenova/pull/218)

## [0.0.3] - 2024-01-19

### Added
- Add `en-US` translation to `Google API page` (https://github.com/emoss08/Trenova/pull/216).
- Add default profile field to `Email Profile` model (https://github.com/emoss08/Trenova/pull/216).

### Changed
- Change `inter` font to `geist` font (https://github.com/emoss08/Trenova/pull/215).
- Change Admin Page Sidebar icons to Font Awesome DuoTone icons (https://github.com/emoss08/Trenova/pull/215).

### Fixed
- Fix typo on General Ledger Account sub table component. (https://github.com/emoss08/Trenova/pull/215)

## [0.0.2] - 2024-01-18

### Changed
- Disabled nested backdrop-filter on Chrome due to Chromium bug (https://github.com/emoss08/Trenova/pull/213).
- Add toast to notify users of unsupported browsers (https://github.com/emoss08/Trenova/pull/213).

## [0.0.1] - 2024-01-16

### Changed
- Initial release.
    - Changelog will be updated moving forward from this release. If you'd like to see the changes made prior to this release, refer to the [commit history](https://github.com/emoss08/Trenova/commits/master/) or [issue tracking system](https://github.com/emoss08/Trenova/issues).

---

For more details on each change, refer to the [Trenova](https://github.com/emoss08/trenova) or issue tracking system.
