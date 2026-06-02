# Delta for Auth

## REMOVED Requirements
Removes the old password-only login behavior.

### Requirement: Legacy Login
The system SHALL reject legacy password-only login requests.

#### Scenario: Legacy endpoint request
- GIVEN a client calls the legacy login endpoint
- WHEN the request is received
- THEN the request is rejected
- AND no session token is issued
