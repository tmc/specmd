# Auth Specification

## Purpose
Authentication and session management for browser and API users.

## Requirements

### Requirement: User Authentication
The system SHALL issue a session token after successful login.

#### Scenario: Valid credentials
- GIVEN a user with valid credentials
- WHEN the user submits the login form
- THEN a session token is returned
- AND the user is redirected to the dashboard

#### Scenario: Invalid credentials
- GIVEN invalid credentials
- WHEN the user submits the login form
- THEN an authentication error is displayed
- AND no session token is issued

### Requirement: Session Expiration
The system MUST expire sessions after 30 minutes of inactivity.

#### Scenario: Idle timeout
- GIVEN an authenticated session
- WHEN 30 minutes pass without activity
- THEN the session is invalidated
- AND the user must authenticate again
