# Delta for Billing

## MODIFIED Requirements
Clarifies invoice payment behavior.

### Requirement: Invoice Payment
The system MUST record successful invoice payments with a receipt identifier.

#### Scenario: Payment succeeds
- GIVEN an unpaid invoice
- WHEN payment succeeds
- THEN the invoice is marked paid
- AND a receipt identifier is recorded
