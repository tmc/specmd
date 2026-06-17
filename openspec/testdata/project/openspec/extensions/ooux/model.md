# OOUX Model: Authentication

## Objects

### Object: User
The person authenticating with the system.

#### Attributes
- email
- two-factor status

#### Relationships
- has many Sessions

#### Calls to Action
- log in
- enroll two-factor authentication

### Object: Session
An authenticated period of access.

#### Attributes
- issued at
- expires at

#### Relationships
- belongs to User

#### Calls to Action
- revoke
