#!/bin/sh
set -eu

# QA test script for spec creation and linking.
# Run from the repo root after building: task build && qa/specs/setup.sh
# Creates a temporary project in /tmp/specd-qa with several related specs.

# Resolve the binary path relative to the repo root before changing directory.
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SPECD="$REPO_ROOT/bin/specd"
QA_DIR="/tmp/specd-qa"

echo "=== Cleaning previous QA run ==="
rm -rf "$QA_DIR"
mkdir -p "$QA_DIR"
cd "$QA_DIR"

echo ""
echo "=== Initializing project ==="
$SPECD init --dir specd --username qa-tester --skip-skills

echo ""
echo "=== Creating SPEC-1: User Authentication ==="
$SPECD new-spec \
  --title "User Authentication" \
  --summary "Implement OAuth2 login with Google and GitHub providers" \
  --body "## Overview

Users must be able to sign in using their Google or GitHub accounts via OAuth2.

## Requirements

- Redirect to provider's consent screen
- Exchange authorization code for access token
- Create or update user record on successful login
- Issue a session token (JWT) after authentication
- Support remember-me functionality

## Acceptance Criteria

- Google OAuth2 login works end-to-end
- GitHub OAuth2 login works end-to-end
- New users are created on first login
- Existing users are matched by email"

echo ""
echo "=== Creating SPEC-2: Session Management ==="
$SPECD new-spec \
  --title "Session Management" \
  --summary "Handle user sessions with secure token storage and refresh rotation" \
  --body "## Overview

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
- Password change kills all active sessions"

echo ""
echo "=== Creating SPEC-3: Role-Based Access Control ==="
$SPECD new-spec \
  --title "Role-Based Access Control" \
  --summary "Define user roles and permissions for authorization" \
  --body "## Overview

The system needs a role-based access control (RBAC) layer to restrict actions based on user roles.

## Requirements

- Predefined roles: admin, editor, viewer
- Permissions mapped to API endpoints
- Role assignment during user creation or by admin
- Middleware to enforce authorization on protected routes

## Acceptance Criteria

- Admin can access all endpoints
- Editor cannot delete resources
- Viewer has read-only access
- Unauthorized access returns 403"

echo ""
echo "=== Creating SPEC-4: API Rate Limiting ==="
$SPECD new-spec \
  --title "API Rate Limiting" \
  --summary "Throttle API requests to prevent abuse and ensure fair usage" \
  --body "## Overview

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
- Admin users are not rate limited"

echo ""
echo "=== Creating SPEC-5: Audit Logging ==="
$SPECD new-spec \
  --title "Audit Logging" \
  --summary "Record security-relevant events for compliance and debugging" \
  --body "## Overview

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
- Rate limit violations are logged"

echo ""
echo "=== Creating SPEC-6: Invoice Generation (unrelated to auth) ==="
$SPECD new-spec \
  --title "Invoice Generation" \
  --summary "Generate PDF invoices from billing data with line items and tax calculations" \
  --body "## Overview

The billing module must generate downloadable PDF invoices for completed orders.

## Requirements

- Pull line items from the orders table
- Calculate subtotal, tax, and total
- Apply regional tax rules (GST, VAT, sales tax)
- Generate PDF with company logo, customer details, and itemized breakdown
- Store generated PDF in object storage with a unique URL
- Send invoice via email to the customer

## Acceptance Criteria

- Invoice PDF matches the approved template
- Tax calculations are correct for AU, US, and EU regions
- PDF is accessible via a signed URL for 30 days
- Email delivery succeeds with the PDF attachment"

echo ""
echo "=== Creating SPEC-7: Dark Mode Toggle (unrelated to auth) ==="
$SPECD new-spec \
  --title "Dark Mode Toggle" \
  --summary "Allow users to switch between light and dark color themes" \
  --body "## Overview

The UI must support a light/dark theme toggle that persists across sessions.

## Requirements

- Toggle switch in the navigation bar
- Persist preference in localStorage
- Respect OS-level prefers-color-scheme on first visit
- Transition smoothly without flash of unstyled content
- All components must support both themes

## Acceptance Criteria

- Toggle switches theme immediately without page reload
- Preference survives browser restart
- First visit respects OS setting
- No white flash on page load in dark mode"

echo ""
echo "=== Creating SPEC-8: CSV Data Import (unrelated to auth) ==="
$SPECD new-spec \
  --title "CSV Data Import" \
  --summary "Import bulk data from CSV files with validation and error reporting" \
  --body "## Overview

Users need to upload CSV files to bulk-import records into the system.

## Requirements

- Accept CSV files up to 50MB
- Validate headers against expected schema
- Report row-level errors without aborting the entire import
- Support UTF-8 and Latin-1 encodings
- Preview first 10 rows before committing
- Run import as a background job with progress tracking

## Acceptance Criteria

- Valid CSV imports all rows successfully
- Invalid rows are skipped with error details in the report
- File over 50MB is rejected with a clear error message
- Progress bar updates during import"

echo ""
echo "=== Creating SPEC-9: GraphQL Gateway (term in title) ==="
$SPECD new-spec \
  --title "GraphQL Gateway" \
  --summary "Unified API gateway for all microservices" \
  --body "## Overview

Build a GraphQL gateway that aggregates data from multiple backend microservices into a single query endpoint.

## Requirements

- Schema stitching across order, inventory, and user services
- Dataloader batching to avoid N+1 queries
- Rate limiting per client API key
- Authentication via JWT forwarded from the edge proxy"

echo ""
echo "=== Creating SPEC-10: Service Communication Layer (term in body only) ==="
$SPECD new-spec \
  --title "Service Communication Layer" \
  --summary "Internal RPC framework for backend services" \
  --body "## Overview

Design the inter-service communication layer using gRPC with protobuf schemas.

## Requirements

- All services register with the service mesh
- Health checks and circuit breakers on every connection
- The order service must expose a GraphQL gateway compatible schema for the frontend team
- Retry with exponential backoff on transient failures"

echo ""
echo "=== Creating SPEC-11: API Schema Registry (term in summary only) ==="
$SPECD new-spec \
  --title "API Schema Registry" \
  --summary "Central registry for GraphQL gateway schemas and versioning" \
  --body "## Overview

Maintain a central registry where all service schemas are published and versioned. The frontend team consumes the merged schema from this registry.

## Requirements

- Schema upload via CLI or CI pipeline
- Breaking change detection on PR
- Version history with rollback support
- Webhook notifications on schema changes"

echo ""
echo "=== Linking specs ==="

echo "-- Linking SPEC-1 (Auth) with SPEC-2 (Sessions) and SPEC-3 (RBAC)"
$SPECD update-spec --id "SPEC-1" --type "functional" --link-specs "SPEC-2,SPEC-3"

echo ""
echo "-- Linking SPEC-2 (Sessions) with SPEC-5 (Audit)"
$SPECD update-spec --id "SPEC-2" --type "functional" --link-specs "SPEC-5"

echo ""
echo "-- Linking SPEC-3 (RBAC) with SPEC-5 (Audit)"
$SPECD update-spec --id "SPEC-3" --type "functional" --link-specs "SPEC-5"

echo ""
echo "-- Setting SPEC-4 (Rate Limiting) as non-functional"
$SPECD update-spec --id "SPEC-4" --type "nonfunctional"

echo ""
echo "-- Setting SPEC-5 (Audit) as non-functional"
$SPECD update-spec --id "SPEC-5" --type "nonfunctional"

DB=".specd.cache"

echo ""
echo "=== Verification ==="
echo ""
echo "--- Specs in DB ---"
sqlite3 $DB "SELECT id, type, title FROM specs ORDER BY id;"

echo ""
echo "--- Links ---"
sqlite3 $DB "SELECT from_spec, to_spec FROM spec_links ORDER BY from_spec, to_spec;"

echo ""
echo "--- Spec files ---"
find specd/specs -name 'spec.md' | sort

echo ""
echo "--- FTS search: 'authentication session' ---"
sqlite3 $DB "SELECT id, title FROM specs_fts JOIN specs ON specs.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"authentication\" \"session\"' ORDER BY bm25(specs_fts, 10.0, 5.0, 1.0);"

echo ""
echo "--- FTS search: 'rate limit abuse' ---"
sqlite3 $DB "SELECT id, title FROM specs_fts JOIN specs ON specs.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"rate\" \"limit\"' ORDER BY bm25(specs_fts, 10.0, 5.0, 1.0);"

echo ""
echo "--- Trigram search: 'auth' (substring) ---"
sqlite3 $DB "SELECT kind, ref_id FROM search_trigram WHERE search_trigram MATCH '\"auth\"' AND kind = 'spec';"

echo ""
echo "--- NEGATIVE: 'invoice PDF tax' should NOT match auth/session specs ---"
sqlite3 $DB "SELECT id, title FROM specs_fts JOIN specs ON specs.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"invoice\" \"PDF\" \"tax\"' ORDER BY bm25(specs_fts, 10.0, 5.0, 1.0);"

echo ""
echo "--- NEGATIVE: 'dark mode theme toggle' should NOT match auth/billing specs ---"
sqlite3 $DB "SELECT id, title FROM specs_fts JOIN specs ON specs.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"dark\" \"mode\" \"theme\"' ORDER BY bm25(specs_fts, 10.0, 5.0, 1.0);"

echo ""
echo "--- NEGATIVE: 'CSV import encoding' should NOT match auth/UI specs ---"
sqlite3 $DB "SELECT id, title FROM specs_fts JOIN specs ON specs.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"CSV\" \"import\" \"encoding\"' ORDER BY bm25(specs_fts, 10.0, 5.0, 1.0);"

echo ""
echo "--- WEIGHT TEST: 'GraphQL' with title=100, summary=1, body=1 ---"
echo "    (SPEC-9 'GraphQL Gateway' should rank first — term is in title)"
sqlite3 $DB "SELECT s.id, s.title, bm25(specs_fts, 100.0, 1.0, 1.0) AS score FROM specs_fts JOIN specs s ON s.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"GraphQL\"' ORDER BY score LIMIT 5;"

echo ""
echo "--- WEIGHT TEST: 'GraphQL' with title=1, summary=1, body=100 ---"
echo "    (SPEC-10 'Service Communication Layer' should rank first — term is in body)"
sqlite3 $DB "SELECT s.id, s.title, bm25(specs_fts, 1.0, 1.0, 100.0) AS score FROM specs_fts JOIN specs s ON s.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"GraphQL\"' ORDER BY score LIMIT 5;"

echo ""
echo "--- WEIGHT TEST: 'GraphQL' with title=1, summary=100, body=1 ---"
echo "    (SPEC-11 'API Schema Registry' should rank first — term is in summary)"
sqlite3 $DB "SELECT s.id, s.title, bm25(specs_fts, 1.0, 100.0, 1.0) AS score FROM specs_fts JOIN specs s ON s.rowid = specs_fts.rowid WHERE specs_fts MATCH '\"GraphQL\"' ORDER BY score LIMIT 5;"

echo ""
echo "=== QA project created at $QA_DIR ==="
echo "Inspect with: cd $QA_DIR && sqlite3 $DB"
