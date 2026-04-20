---
title: API Rate Limiting
summary: Throttle API requests to prevent abuse and ensure fair usage
type: nonfunctional
---

## Overview

Protect the API from abuse by implementing rate limiting per user and per IP.

## Requirements

- Sliding window rate limiter
- Configurable limits per endpoint
- Return 429 Too Many Requests with Retry-After header
- Exempt authenticated admin users
- Log rate-limited requests for monitoring

## Acceptance Criteria

- Anonymous requests limited to 60/minute per IP
- Authenticated requests limited to 300/minute per user
- 429 response includes Retry-After header
- Admin users are not rate limited
