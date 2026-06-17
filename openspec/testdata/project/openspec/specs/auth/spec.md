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
