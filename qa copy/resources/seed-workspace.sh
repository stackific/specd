#!/usr/bin/env bash
# seed-workspace.sh — Populates a specd workspace with specs, tasks, links,
# dependencies, criteria, KB docs, and citations for comprehensive QA testing.
#
# Usage:
#   cd /tmp/specd-qa   # (an already-initialized workspace)
#   bash /path/to/specd/qa/resources/seed-workspace.sh /path/to/specd/qa/resources
#
# The first argument is the path to the qa/resources directory.

set -euo pipefail

RESOURCES="${1:?Usage: seed-workspace.sh <path-to-qa/resources>}"

echo "=== Seeding specd workspace ==="

# ── Config ──────────────────────────────────────────────────────
specd config user.name "QA Tester"

# ── Specs (3 types, 5 total) ───────────────────────────────────
echo "Creating specs..."

specd new-spec \
  --title "User Authentication" \
  --type functional \
  --summary "Core authentication system including login, registration, and session management" \
  --body "# User Authentication

Implement a complete authentication system supporting email/password login,
OAuth 2.0 social login, and session management with JWT tokens.

## Requirements

- Email/password registration with email verification
- Login with rate limiting and brute-force protection
- OAuth 2.0 integration with GitHub and Google providers
- JWT-based session tokens with refresh token rotation
- Password reset flow with secure time-limited tokens
- Account lockout after 5 failed login attempts"

specd new-spec \
  --title "Payment Processing" \
  --type business \
  --summary "Stripe integration for subscription billing and one-time payments" \
  --body "# Payment Processing

Integrate Stripe for handling subscription billing, one-time payments,
refunds, and invoice generation.

## Requirements

- Subscription plan management (monthly/annual)
- One-time payment support for add-ons
- Automated invoice generation
- Refund processing with audit trail
- Webhook handling for payment events
- PCI DSS compliance via Stripe Elements"

specd new-spec \
  --title "API Rate Limiting" \
  --type non-functional \
  --summary "Protect API endpoints with configurable per-client rate limits" \
  --body "# API Rate Limiting

Implement rate limiting middleware to protect API endpoints from abuse
and ensure fair resource allocation across clients.

## Requirements

- Token bucket algorithm with configurable burst and sustained rates
- Per-API-key and per-IP rate limit tracking
- Redis-backed distributed counter for multi-instance deployments
- Standard rate limit response headers (X-RateLimit-Limit, Remaining, Reset)
- Configurable limits per endpoint and per plan tier
- Graceful degradation when Redis is unavailable"

specd new-spec \
  --title "Data Export Pipeline" \
  --type functional \
  --summary "Allow users to export their data in CSV and JSON formats" \
  --body "# Data Export Pipeline

Build an asynchronous data export system that allows users to request
exports of their data and download them when ready.

## Requirements

- Export request queue with background processing
- Support CSV and JSON output formats
- Large dataset pagination to avoid memory issues
- Email notification when export is ready
- Automatic cleanup of expired export files after 7 days
- Progress tracking via polling endpoint"

specd new-spec \
  --title "Observability Stack" \
  --type non-functional \
  --summary "Structured logging, metrics, and distributed tracing across all services" \
  --body "# Observability Stack

Instrument all services with structured logging, Prometheus metrics,
and OpenTelemetry distributed tracing.

## Requirements

- JSON structured logging with correlation IDs
- Prometheus metrics for latency, throughput, and error rates
- OpenTelemetry traces propagated across service boundaries
- Grafana dashboards for key business and operational metrics
- Alerting rules for SLA violations
- Log aggregation via Fluentd to Elasticsearch"

# ── Spec Links (bidirectional) ──────────────────────────────────
echo "Linking specs..."

# Auth and Payment share business logic (user identity for billing)
specd link SPEC-1 SPEC-2

# Auth and Rate Limiting (auth tokens feed into rate limit keys)
specd link SPEC-1 SPEC-3

# Rate Limiting and Observability (rate limit events need monitoring)
specd link SPEC-3 SPEC-5

# Data Export and Observability (export jobs need tracing)
specd link SPEC-4 SPEC-5

# ── Tasks under SPEC-1 (Auth) ──────────────────────────────────
echo "Creating tasks for SPEC-1 (Auth)..."

specd new-task \
  --spec-id SPEC-1 \
  --title "Design auth database schema" \
  --summary "Design the users, sessions, and OAuth tokens tables" \
  --status todo \
  --body "# Design auth database schema

Design and document the database schema for the authentication system.

## Acceptance criteria

- [ ] Users table with email, password_hash, verified, locked_until columns
- [ ] Sessions table with user_id, token_hash, expires_at, refresh_token columns
- [ ] OAuth tokens table with provider, provider_user_id, access_token, refresh_token
- [ ] Migration scripts for all tables
- [ ] ERD diagram added to KB"

specd new-task \
  --spec-id SPEC-1 \
  --title "Implement JWT token service" \
  --summary "Service for issuing, validating, and refreshing JWT tokens" \
  --status todo \
  --body "# Implement JWT token service

Build the core JWT service that handles token lifecycle.

## Acceptance criteria

- [ ] Issue access tokens with configurable expiration (default 15 min)
- [ ] Issue refresh tokens with 7-day expiration
- [ ] Validate token signature and claims
- [ ] Refresh token rotation (old refresh token invalidated on use)
- [ ] Token blacklist for logout/revocation"

specd new-task \
  --spec-id SPEC-1 \
  --title "Build login and registration endpoints" \
  --summary "REST endpoints for user registration and login" \
  --status backlog \
  --body "# Build login and registration endpoints

Create the HTTP handlers for user registration and login flows.

## Acceptance criteria

- [ ] POST /auth/register with email validation
- [ ] POST /auth/login with rate limiting
- [ ] POST /auth/refresh for token refresh
- [ ] POST /auth/logout for session invalidation
- [ ] Input validation and error responses"

specd new-task \
  --spec-id SPEC-1 \
  --title "Implement OAuth 2.0 GitHub provider" \
  --summary "OAuth flow for GitHub login" \
  --status backlog \
  --body "# Implement OAuth 2.0 GitHub provider

Integrate GitHub as an OAuth 2.0 identity provider.

## Acceptance criteria

- [ ] GET /auth/github redirects to GitHub authorization
- [ ] GET /auth/github/callback handles the authorization code exchange
- [ ] Creates or links user account on first OAuth login
- [ ] Stores OAuth tokens securely
- [ ] Handles authorization denied gracefully"

specd new-task \
  --spec-id SPEC-1 \
  --title "Add password reset flow" \
  --summary "Secure password reset with email verification" \
  --status backlog \
  --body "# Add password reset flow

Implement the forgot password / reset password flow.

## Acceptance criteria

- [ ] POST /auth/forgot-password sends reset email
- [ ] Reset token expires after 1 hour
- [ ] POST /auth/reset-password validates token and updates password
- [ ] Old sessions invalidated after password change"

# ── Tasks under SPEC-2 (Payment) ───────────────────────────────
echo "Creating tasks for SPEC-2 (Payment)..."

specd new-task \
  --spec-id SPEC-2 \
  --title "Set up Stripe integration" \
  --summary "Configure Stripe SDK, API keys, and webhook endpoint" \
  --status in_progress \
  --body "# Set up Stripe integration

Initialize the Stripe integration with proper configuration.

## Acceptance criteria

- [x] Stripe Go SDK added as dependency
- [x] API key management via environment variables
- [ ] Webhook endpoint with signature verification
- [ ] Test mode and live mode switching"

specd new-task \
  --spec-id SPEC-2 \
  --title "Implement subscription management" \
  --summary "Create, upgrade, downgrade, and cancel subscription plans" \
  --status backlog \
  --body "# Implement subscription management

Build the subscription lifecycle management system.

## Acceptance criteria

- [ ] Create subscription for new customers
- [ ] Upgrade and downgrade between plans
- [ ] Cancel subscription with end-of-period handling
- [ ] Sync subscription status from Stripe webhooks
- [ ] Handle payment failures and retry logic"

# ── Tasks under SPEC-3 (Rate Limiting) ─────────────────────────
echo "Creating tasks for SPEC-3 (Rate Limiting)..."

specd new-task \
  --spec-id SPEC-3 \
  --title "Implement token bucket algorithm" \
  --summary "Core rate limiting algorithm with Redis backend" \
  --status done \
  --body "# Implement token bucket algorithm

Build the token bucket rate limiter with Redis for distributed state.

## Acceptance criteria

- [x] Token bucket implementation with burst and refill rate
- [x] Redis-backed counter with atomic operations
- [x] Fallback to in-memory counter when Redis unavailable
- [x] Unit tests for edge cases (empty bucket, refill timing)"

specd new-task \
  --spec-id SPEC-3 \
  --title "Add rate limit middleware" \
  --summary "HTTP middleware that enforces rate limits on incoming requests" \
  --status todo \
  --body "# Add rate limit middleware

Create HTTP middleware that checks rate limits before handlers execute.

## Acceptance criteria

- [ ] Middleware extracts API key or client IP for limit tracking
- [ ] Returns 429 with Retry-After header when limit exceeded
- [ ] Sets X-RateLimit-Limit, Remaining, Reset headers on all responses
- [ ] Configurable per-route overrides
- [ ] Bypass for internal service-to-service calls"

# ── Tasks under SPEC-4 (Data Export) ───────────────────────────
echo "Creating tasks for SPEC-4 (Data Export)..."

specd new-task \
  --spec-id SPEC-4 \
  --title "Build export queue worker" \
  --summary "Background worker that processes export requests" \
  --status todo \
  --body "# Build export queue worker

Implement the background job processor for data export requests.

## Acceptance criteria

- [ ] Queue consumer reads export requests
- [ ] Generates CSV or JSON based on requested format
- [ ] Streams large datasets to avoid memory pressure
- [ ] Uploads completed export to object storage
- [ ] Updates export status to 'ready' when complete"

specd new-task \
  --spec-id SPEC-4 \
  --title "Create export API endpoints" \
  --summary "REST endpoints for requesting and downloading exports" \
  --status backlog \
  --body "# Create export API endpoints

Build the HTTP handlers for the export feature.

## Acceptance criteria

- [ ] POST /exports to create an export request
- [ ] GET /exports/:id for status polling
- [ ] GET /exports/:id/download for file download
- [ ] Export requests expire after 7 days"

# ── Task Links (task-to-task, bidirectional) ────────────────────
echo "Linking tasks..."

# JWT service and login endpoints are related
specd link TASK-2 TASK-3

# Login endpoints and OAuth provider are related
specd link TASK-3 TASK-4

# Stripe setup and subscription management are related
specd link TASK-6 TASK-7

# Export queue and export API are related
specd link TASK-10 TASK-11

# ── Task Dependencies (directed: blocker -> blocked) ───────────
echo "Adding dependencies..."

# Must design schema before implementing JWT service
specd depend TASK-2 --on TASK-1

# Must have JWT service before building login endpoints
specd depend TASK-3 --on TASK-2

# Must have login endpoints before OAuth integration
specd depend TASK-4 --on TASK-3

# Password reset depends on login endpoints existing
specd depend TASK-5 --on TASK-3

# Subscription management depends on Stripe setup
specd depend TASK-7 --on TASK-6

# Rate limit middleware depends on the token bucket being done
specd depend TASK-9 --on TASK-8

# Export API depends on the queue worker
specd depend TASK-11 --on TASK-10

# ── KB Documents ────────────────────────────────────────────────
echo "Adding KB documents..."

specd kb add "$RESOURCES/sample-kb-article.md" --title "OAuth 2.0 RFC Guide" --note "Reference for SPEC-1 OAuth implementation"
specd kb add "$RESOURCES/sample-kb-guide.md" --title "JWT Best Practices" --note "Security guidelines for TASK-2 JWT service"
specd kb add "$RESOURCES/sample-kb-notes.txt" --title "Deployment Runbook" --note "Operational procedures reference"
specd kb add "$RESOURCES/sample-kb-page.html" --title "API Rate Limiting Docs" --note "Reference for SPEC-3 rate limiting"
specd kb add "$RESOURCES/sample-kb-long.txt" --title "Architecture Overview" --note "System architecture reference"

# ── Citations ───────────────────────────────────────────────────
echo "Adding citations..."

# Auth spec cites OAuth and JWT docs
specd cite SPEC-1 KB-1:0 KB-1:2 KB-2:0

# Rate limiting spec cites API rate limiting HTML doc
specd cite SPEC-3 KB-4:0 KB-4:1

# JWT task cites JWT best practices
specd cite TASK-2 KB-2:0 KB-2:1

# OAuth task cites OAuth RFC
specd cite TASK-4 KB-1:0 KB-1:3

# ── Check some criteria (partially done tasks) ─────────────────
echo "Checking some criteria..."

# TASK-1 (schema design): check 2 of 5
specd criteria check TASK-1 1
specd criteria check TASK-1 2

# TASK-9 (rate limit middleware): check nothing (fresh todo)

# ── Move one task to blocked status ─────────────────────────────
# TASK-4 (OAuth) is blocked because TASK-3 (login) isn't done
# It's already in backlog with a dependency, but let's explicitly block one
# to test the blocked column
specd move TASK-5 --status blocked

echo ""
echo "=== Workspace seeded ==="
echo ""
echo "Summary:"
specd status
echo ""
echo "Next tasks:"
specd next --limit 5
