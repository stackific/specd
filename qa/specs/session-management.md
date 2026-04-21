---
title: Session Management
summary: Handle user sessions with secure token storage and refresh rotation
type: functional
---

## Overview

After authentication, the system must maintain user sessions securely.

## Requirements

- Store session tokens in HTTP-only secure cookies
- Implement JWT with short-lived access tokens (15 min)
- Implement refresh token rotation
- Invalidate all sessions on password change
- Support concurrent sessions across devices

## Acceptance Criteria

- Access tokens expire after 15 minutes
- Refresh tokens rotate on each use
- Stolen refresh token invalidates the entire chain
- Password change kills all active sessions
