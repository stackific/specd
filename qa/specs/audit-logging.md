---
title: Audit Logging
summary: Record security-relevant events for compliance and debugging
type: nonfunctional
---

## Overview

All security-relevant actions must be logged for audit trail and compliance purposes.

## Requirements

- Log authentication events (login, logout, failed attempts)
- Log authorization failures (403 responses)
- Log role changes and permission modifications
- Log rate limiting events
- Structured JSON log format with timestamp, user, action, resource
- Retain logs for 90 days minimum

## Acceptance Criteria

- Failed login attempts are logged with IP and timestamp
- Role changes include before/after values
- Logs are queryable by user, action, and time range
- Rate limit violations are logged
