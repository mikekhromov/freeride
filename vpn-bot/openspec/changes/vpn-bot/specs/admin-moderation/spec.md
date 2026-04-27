## ADDED Requirements

### Requirement: Admin-only commands

The system SHALL restrict `/stats`, `/users`, `/approve`, `/revoke`, `/test`, and approve/reject callbacks to Telegram senders whose id is listed in `ADMIN_IDS`; non-admins SHALL receive a denial message or empty callback response as implemented.

#### Scenario: Non-admin invokes admin command

- **WHEN** a non-admin user sends `/stats` or `/approve`
- **THEN** the bot responds that access is denied

### Requirement: Admin test delivery preview

The system SHALL provide an admin-only `/test` command that sends the same connection delivery sequence as end users receive, using the admin caller’s own active user record and Hiddify links when available.

#### Scenario: Admin previews delivery

- **WHEN** an admin sends `/test` and their Telegram id has an `active` user row with a Hiddify UUID
- **THEN** the bot sends the standard VPN and Telegram Proxy delivery payload to the admin

#### Scenario: Admin test without eligible user row

- **WHEN** an admin sends `/test` but has no suitable active record
- **THEN** the bot explains what is missing instead of sending partial secrets

### Requirement: Aggregate user statistics

The system SHALL provide an admin command that returns counts of users grouped by status from the database and SHALL attempt to include total Hiddify user count when the API is reachable.

#### Scenario: Admin runs stats without username argument

- **WHEN** an admin sends `/stats` with no payload
- **THEN** the bot sends a summary including per-status counts and Hiddify total or an explicit failure note for Hiddify

### Requirement: Admin lookup by username

The system SHALL allow an admin to pass a Telegram username to `/stats` and SHALL return that user’s status, active access count, and Telegram id when found.

#### Scenario: Admin queries single user

- **WHEN** an admin sends `/stats` with a username payload
- **THEN** the bot returns the user’s status and identifiers or states that the user was not found

### Requirement: Recent users list

The system SHALL list a bounded number of recent users with username (or placeholder), Telegram id, and status for admins via `/users`.

#### Scenario: Admin lists users

- **WHEN** an admin sends `/users`
- **THEN** the bot returns up to the configured limit of recent users or an error message if the query fails

### Requirement: Approve by username

The system SHALL allow an admin to approve a pending or eligible user by `/approve` with username, delegating to provisioning logic and notifying the target user on success or idempotent «already active» paths.

#### Scenario: Admin approves via command

- **WHEN** an admin sends `/approve` with a valid username
- **THEN** the provisioning workflow runs and the admin receives confirmation or an error message

### Requirement: Inline approve and reject

The system SHALL process callback data prefixed with `a:` as approve and `x:` as reject for the encoded internal user id, SHALL enforce admin-only execution, and on reject SHALL move a `pending` user to `banned` and notify the user.

#### Scenario: Admin rejects from inline button

- **WHEN** an admin taps reject for a user still in `pending`
- **THEN** the user becomes `banned`, receives a rejection notice, and the admin gets confirmation
