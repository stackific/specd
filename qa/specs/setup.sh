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
$SPECD init --folder specd --username qa-tester --skip-skills

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

echo ""
echo "=== Verification ==="
echo ""
echo "--- Specs in DB ---"
sqlite3 specd/cache.db "SELECT id, type, title FROM specs ORDER BY id;"

echo ""
echo "--- Links ---"
sqlite3 specd/cache.db "SELECT from_spec, to_spec FROM spec_links ORDER BY from_spec, to_spec;"

echo ""
echo "--- Spec files ---"
find specd/specs -name 'spec.md' | sort

echo ""
echo "--- FTS search test: 'authentication session' ---"
sqlite3 specd/cache.db "SELECT id, title FROM specs_fts WHERE specs_fts MATCH 'authentication OR session' ORDER BY bm25(specs_fts);"

echo ""
echo "--- FTS search test: 'rate limit abuse' ---"
sqlite3 specd/cache.db "SELECT id, title FROM specs_fts WHERE specs_fts MATCH 'rate OR limit OR abuse' ORDER BY bm25(specs_fts);"

echo ""
echo "--- Trigram search test: 'auth' (substring) ---"
sqlite3 specd/cache.db "SELECT kind, ref_id FROM search_trigram WHERE search_trigram MATCH 'auth' AND kind = 'spec';"

echo ""
echo "--- NEGATIVE: 'invoice PDF tax' should NOT match auth/session specs ---"
sqlite3 specd/cache.db "SELECT id, title FROM specs_fts WHERE specs_fts MATCH 'invoice OR PDF OR tax' ORDER BY bm25(specs_fts);"

echo ""
echo "--- NEGATIVE: 'dark mode theme toggle' should NOT match auth/billing specs ---"
sqlite3 specd/cache.db "SELECT id, title FROM specs_fts WHERE specs_fts MATCH 'dark OR mode OR theme OR toggle' ORDER BY bm25(specs_fts);"

echo ""
echo "--- NEGATIVE: 'CSV import encoding' should NOT match auth/UI specs ---"
sqlite3 specd/cache.db "SELECT id, title FROM specs_fts WHERE specs_fts MATCH 'CSV OR import OR encoding' ORDER BY bm25(specs_fts);"

echo ""
echo "=== QA project created at $QA_DIR ==="
echo "Inspect with: cd $QA_DIR && sqlite3 specd/cache.db"
