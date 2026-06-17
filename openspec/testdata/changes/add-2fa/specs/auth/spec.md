# Delta for Auth

## ADDED Requirements
Adds the user-facing behavior for two-factor authentication during login.

### Requirement: Two-Factor Authentication
The system MUST require a one-time code after password authentication when
two-factor authentication is enabled for the user.

#### Scenario: OTP challenge
- GIVEN a user with two-factor authentication enabled
- WHEN password authentication succeeds
- THEN an OTP challenge is shown
- AND login completes only after a valid OTP code is submitted

#### Scenario: Invalid OTP
- GIVEN a user is viewing an OTP challenge
- WHEN the user submits an invalid OTP code
- THEN login remains incomplete
- AND an error is displayed
