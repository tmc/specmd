# Delta for Auth

## MODIFIED Requirements
Updates the idle timeout behavior for authenticated sessions.

### Requirement: Session Expiration
The system MUST expire sessions after 15 minutes of inactivity.

#### Scenario: Idle timeout
- GIVEN an authenticated session
- WHEN 15 minutes pass without activity
- THEN the session is invalidated
- AND the user must authenticate again
