---
id: SPEC-QA
slug: claims-example
type: functional
summary: Example spec demonstrating the acceptance criteria claims format
---

# Password Reset Flow

## Overview

Users who forget their password need a way to securely reset it via email verification.

## Requirements

- Generate a time-limited reset token on request
- Send the token via email to the registered address
- Validate the token before allowing a new password to be set
- Invalidate all existing sessions after a successful reset

## Acceptance Criteria

- The system must generate a unique reset token with a 15-minute expiry
- The system must send the reset link to the user's verified email address
- The system should reject expired or already-used tokens with a clear error
- The system must invalidate all active sessions after a successful password change
- Users may request a new token if the previous one expired
- The system might rate-limit reset requests to 3 per hour per email address
