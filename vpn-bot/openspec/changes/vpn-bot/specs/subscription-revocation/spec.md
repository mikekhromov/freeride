## ADDED Requirements

### Requirement: Admin-only revoke command

The system SHALL expose `/revoke` with a username argument only to admins; non-admins SHALL receive a denial message.

#### Scenario: Non-admin revoke

- **WHEN** a non-admin sends `/revoke @user`
- **THEN** the bot responds that the command is not allowed

### Requirement: Revoke deletes Hiddify user when present

The system SHALL look up the user by case-insensitive username, SHALL call Hiddify delete when `hiddify_uuid` is non-empty before updating the database, and SHALL fail the operation if Hiddify deletion fails.

#### Scenario: Revoke active provisioned user

- **WHEN** an admin revokes a user that has a non-empty Hiddify UUID
- **THEN** Hiddify user deletion is attempted first and the database row is set to `banned` with UUID cleared on success

### Requirement: Revoke without UUID still bans locally

The system SHALL still set the user to `banned` and clear UUID when there was no Hiddify UUID, and SHALL report that no active UUID existed.

#### Scenario: Revoke user without Hiddify UUID

- **WHEN** an admin revokes a user with empty `hiddify_uuid`
- **THEN** the row is updated to `banned` and the admin message indicates no active UUID was present

### Requirement: Notify revoked user

The system SHALL send the affected Telegram user a direct message stating that access was revoked.

#### Scenario: Successful revoke

- **WHEN** revoke completes successfully
- **THEN** the target user receives a revocation notice in addition to the admin confirmation
