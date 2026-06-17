# Auth Specification With Code

## Purpose
Authentication examples sometimes contain Markdown snippets that should not be
parsed as real OpenSpec headings.

```markdown
## Requirements

### Requirement: Ignored From Code Fence
The system SHALL not parse this fenced example.

#### Scenario: Ignored
- THEN this scenario is ignored
```

## Requirements

### Requirement: Parsed Outside Code Fence
The system SHALL parse headings that are outside fenced examples.

#### Scenario: Parsed scenario
- GIVEN a real requirements section
- WHEN the parser reads the document
- THEN exactly one requirement is parsed
