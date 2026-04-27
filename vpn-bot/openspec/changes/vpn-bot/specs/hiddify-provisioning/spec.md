## ADDED Requirements

### Requirement: Create Hiddify user on approval

The system SHALL create a Hiddify user when an approval is allowed for a user in `pending` or a recovering `active` path without an existing UUID, using a display name derived from Telegram username or `tg-<telegram_id>`.

#### Scenario: First-time approval

- **WHEN** `ApproveUser` runs for a user in `pending`
- **THEN** the system calls Hiddify to create a user with package duration and usage limit taken from configuration

### Requirement: Persist activation in database

The system SHALL store the returned Hiddify UUID and mark the user `active` with approving admin metadata after successful Hiddify user creation and link resolution.

#### Scenario: Approve completes successfully

- **WHEN** Hiddify returns a new user UUID and profile links are obtained
- **THEN** the database row is updated to `active` with the UUID set

### Requirement: Idempotent active user with links

The system SHALL treat an already `active` user with a stored UUID as already provisioned and SHALL return existing profile and MTProxy links without creating a duplicate Hiddify user when both link fetches succeed.

#### Scenario: Re-approve active user

- **WHEN** `ApproveUser` is invoked for a user already `active` with non-empty UUID and Hiddify returns links
- **THEN** the function indicates «already active» and no new Hiddify user is created

### Requirement: Blocked statuses cannot be approved

The system SHALL refuse approval for users in `banned` and SHALL refuse when status is neither `pending` nor the approved re-entry path defined by implementation.

#### Scenario: Approve banned user

- **WHEN** approval is attempted for a `banned` user
- **THEN** the operation fails with an explanatory error and no Hiddify user is created
