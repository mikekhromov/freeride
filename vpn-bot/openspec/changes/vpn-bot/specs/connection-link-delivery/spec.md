## ADDED Requirements

### Requirement: Telegram Proxy link MUST use users subdomain

The system SHALL provide Telegram Proxy links with a host that starts with `users.` and SHALL NOT send direct IP-based proxy links to end users.

#### Scenario: Proxy link is prepared for delivery

- **WHEN** the bot formats a Telegram Proxy link for a user
- **THEN** the host component starts with `users.` and the full link is sent in that domain form

### Requirement: VPN links SHALL be separated by connection type

The system SHALL provide three distinct VPN links to the user: WireGuard configuration link, Full Xray configuration link, and a link to all available configurations.

#### Scenario: User gets connection options

- **WHEN** access is approved or configs are reissued
- **THEN** the first delivery message includes all three links labeled as WireGuard, Full Xray, and All configs

### Requirement: VPN message SHALL include download actions

The system SHALL include user-facing actions to download WireGuard and Full Xray configs as files.

#### Scenario: User sees VPN actions

- **WHEN** VPN payload is sent
- **THEN** the message contains two actions: download WireGuard and download Full Xray

### Requirement: Download action SHALL fetch and return config file

When a user presses a download action, the system SHALL perform server-side fetch of the source config URL and SHALL return the fetched content as a text file document.

#### Scenario: User downloads WireGuard

- **WHEN** the user presses the WireGuard download action
- **THEN** the bot fetches WireGuard config content from the source link and sends it as a document

### Requirement: Downloaded file name SHALL follow mask

The system SHALL name downloaded files with the mask `<user>_<protocol>.txt`, where `<protocol>` is `wireguard` or `xray`, and `<user>` is normalized from username or telegram id.

#### Scenario: Xray file is generated

- **WHEN** the user downloads Full Xray config
- **THEN** the returned file name matches `<user>_xray.txt`

### Requirement: Delivery sequence SHALL be deterministic

The system SHALL send connection details in this order: (1) VPN payload, (2) Telegram Proxy payload, (3) support contact message.

#### Scenario: Initial configuration delivery

- **WHEN** the user receives fresh connection data
- **THEN** messages are delivered in the required three-step sequence without reordering
