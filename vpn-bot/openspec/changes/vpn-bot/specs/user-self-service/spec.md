## ADDED Requirements

### Requirement: User self reissue command SHALL be available

The system SHALL provide a user-level command to forget/reissue personal configs using `/revoke` as primary command and `/revok` as alias.

#### Scenario: User requests reissue

- **WHEN** an active user invokes `/revoke` or `/revok`
- **THEN** the system invalidates previous personal configs, issues fresh configs, and sends a new connection delivery payload

### Requirement: Self reissue SHALL be scoped to caller identity

The self reissue command SHALL only affect the caller's own account and SHALL NOT accept arbitrary usernames for user-level flow.

#### Scenario: User triggers self command

- **WHEN** a user invokes self reissue
- **THEN** only records related to the sender's Telegram id are modified

### Requirement: User traffic stats command SHALL provide minimal summary

The system SHALL provide a user-level `stats` command that returns minimal traffic statistics including used traffic and remaining/limit figures.

#### Scenario: User checks personal traffic

- **WHEN** an active user invokes `/stats`
- **THEN** the bot returns concise personal traffic stats in a readable format

### Requirement: Support contact message SHALL conclude delivery

After sending VPN and Telegram Proxy payloads, the system SHALL send a final message with support contact tag and instruction to write in case of issues.

#### Scenario: End of onboarding flow

- **WHEN** connection links are delivered successfully
- **THEN** the final message includes configured support tag and troubleshooting call to action
