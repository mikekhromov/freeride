## ADDED Requirements

### Requirement: Status for unknown user

The system SHALL tell the user to use `/start` and request access when `/status` is invoked and no user row exists for the sender.

#### Scenario: No database row

- **WHEN** a user sends `/status` and the store returns no row
- **THEN** the bot sends a message directing them to `/start`

### Requirement: Status without active secret

The system SHALL report the stored status and state that there is no active secret when the user is not `active` or has an empty Hiddify UUID.

#### Scenario: Pending or inactive

- **WHEN** `/status` is sent and the user row exists but status is not active or UUID is empty
- **THEN** the bot responds with status text and indicates no active secret

### Requirement: Active user receives links

The system SHALL resolve VPN profile URL and MTProxy link from Hiddify for an `active` user with UUID and SHALL send both to the user in one message.

#### Scenario: Fully active user

- **WHEN** `/status` is sent for an `active` user with non-empty UUID and Hiddify succeeds
- **THEN** the message contains VPN and MTProxy sections with URLs

### Requirement: Hiddify failure on status

The system SHALL return a user-visible error if Hiddify link resolution fails for an otherwise active user.

#### Scenario: Hiddify error

- **WHEN** the user is `active` with UUID but Hiddify returns an error fetching links
- **THEN** the bot tells the user that the link could not be retrieved and to contact an admin
