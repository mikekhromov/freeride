## ADDED Requirements

### Requirement: Start command shows access request entry point

The system SHALL respond to the `/start` command with a short greeting and an inline button whose callback data is `req` and whose label invites the user to request access.

#### Scenario: User opens bot

- **WHEN** a user sends `/start`
- **THEN** the bot sends a message containing an inline keyboard with one button to request access

### Requirement: Banned user cannot request access

The system SHALL NOT allow a user whose stored status is `banned` to complete a new access request via the request callback; the user SHALL receive a message that access is blocked.

#### Scenario: Banned user taps request

- **WHEN** a user with status `banned` triggers the access request callback
- **THEN** the bot responds indicating access is blocked and does not notify admins of a new pending request

### Requirement: Active user with provisioned access is directed to status

The system SHALL NOT treat an access request as a new pending flow when the user is already `active` and has a non-empty Hiddify UUID; the user SHALL be told they already have access and SHALL be pointed to `/status`.

#### Scenario: Active user with UUID requests again

- **WHEN** a user with status `active` and a stored Hiddify UUID triggers the access request callback
- **THEN** the bot informs the user that access is already active and references `/status`

### Requirement: Pending request notifies all admins

The system SHALL upsert the user record toward `pending` (without overriding `active` or `banned`), SHALL send each configured admin a message describing the user, and SHALL attach inline approve/reject buttons keyed to the internal user row id.

#### Scenario: New or updated pending request

- **WHEN** an eligible user completes the access request callback successfully
- **THEN** the user receives confirmation that the request was sent and each admin receives one message with approve and reject actions
