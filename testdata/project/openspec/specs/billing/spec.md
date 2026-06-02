# Billing Specification

## Purpose
Billing behavior for invoices, payments, and account balances.

## Requirements

### Requirement: Invoice Payment
The system MUST record successful invoice payments.

#### Scenario: Payment succeeds
- GIVEN an unpaid invoice
- WHEN payment succeeds
- THEN the invoice is marked paid
