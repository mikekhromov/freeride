## ADDED Requirements

### Requirement: VPN and Telegram Proxy cards SHALL be image-based

The system SHALL send two rendered image cards during connection delivery: one for VPN options and one for Telegram Proxy. The cards SHALL use project branding background assets.

#### Scenario: User receives onboarding payload

- **WHEN** the bot starts sending connection options
- **THEN** the first message includes a VPN card image and the second message includes a Telegram Proxy card image

### Requirement: Card text rendering SHALL use golang.org/x/image

The system SHALL render card labels and overlay text using `golang.org/x/image`.

#### Scenario: Card is generated at runtime

- **WHEN** card generation is executed
- **THEN** text rendering uses `golang.org/x/image` and produces readable labels

### Requirement: Card labels SHALL match required captions

The VPN card SHALL include the caption `VPN`. The proxy card SHALL include the caption `Telegram Proxy`.

#### Scenario: Captions are drawn

- **WHEN** each card image is generated
- **THEN** the rendered text is exactly `VPN` for the first card and `Telegram Proxy` for the second card

### Requirement: Delivery MUST degrade gracefully on render failure

If image generation fails, the system SHALL still deliver the required links in text form and SHALL include a generic warning message without exposing internal errors.

#### Scenario: Renderer fails

- **WHEN** the image generator returns an error
- **THEN** the bot sends text-only link messages so the user can still connect
