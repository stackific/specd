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

DB=".specd.cache"

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
echo "=== Creating bulk specs (SPEC-12 onwards) ==="

# Helper: make a spec with consistent body shape so the script stays readable.
# Optional 5th arg is the type (business|functional|nonfunctional). When provided,
# the spec is retyped via update-spec since new-spec defaults to "business".
make_spec() {
  TITLE="$1"; SUMMARY="$2"; OVERVIEW="$3"; CRITERIA="$4"; TYPE="${5:-}"
  $SPECD new-spec --title "$TITLE" --summary "$SUMMARY" --body "## Overview

$OVERVIEW

## Acceptance Criteria

$CRITERIA"
  if [ -n "$TYPE" ] && [ "$TYPE" != "business" ]; then
    LAST_ID=$(sqlite3 "$DB" "SELECT id FROM specs ORDER BY CAST(SUBSTR(id, 6) AS INTEGER) DESC LIMIT 1;")
    $SPECD update-spec --id "$LAST_ID" --type "$TYPE" >/dev/null
  fi
}

make_spec "Email Delivery Pipeline" \
  "Transactional email service with retry and bounce handling" \
  "Send transactional emails (receipts, password resets, alerts) through a third-party provider with reliable retry and bounce tracking." \
  "- Failed sends must retry with exponential backoff
- Hard bounces must be recorded and excluded from future sends
- Delivery status must be queryable per message" \
  business

make_spec "Push Notification Service" \
  "Cross-platform mobile push notifications via FCM and APNs" \
  "Deliver push notifications to iOS and Android devices with per-user opt-in and quiet hours." \
  "- Users must be able to opt out per category
- Delivery must respect per-user quiet hours
- Failed pushes must be retried up to three times" \
  business

make_spec "Webhook Delivery" \
  "Outbound webhooks with signed payloads and retries" \
  "Allow customers to register webhook URLs and receive signed payloads for domain events." \
  "- Each payload must be signed with HMAC-SHA256
- Failed deliveries must retry with backoff for up to 24 hours
- Delivery attempts must be queryable per webhook" \
  functional

make_spec "Stripe Payment Integration" \
  "Card payments via Stripe Checkout and webhooks" \
  "Charge customers via Stripe Checkout, handle async webhooks for payment status, and reconcile with internal orders." \
  "- Successful payments must transition the order to paid
- Refunds must reverse the order to refunded
- Webhook signatures must be verified before processing" \
  business

make_spec "Subscription Billing" \
  "Recurring subscription plans with proration and dunning" \
  "Manage recurring subscriptions, plan changes with proration, and dunning for failed renewals." \
  "- Plan upgrades must prorate the current period
- Failed renewals must enter a 14-day dunning window
- Cancelled subscriptions must end at period end" \
  business

make_spec "Customer Portal" \
  "Self-serve account, billing, and subscription management" \
  "Logged-in customers can update payment methods, view invoices, and change plans without contacting support." \
  "- Users must be able to update their default payment method
- Past invoices must be downloadable as PDF
- Plan changes must take effect immediately" \
  business

make_spec "File Upload Service" \
  "Direct-to-S3 uploads with virus scanning and quotas" \
  "Allow users to upload files directly to S3 with pre-signed URLs, scan for malware, and enforce per-account quotas." \
  "- Pre-signed URLs must expire after 15 minutes
- Uploaded files must be virus-scanned before becoming available
- Per-account storage quota must be enforced" \
  functional

make_spec "Image Optimization" \
  "On-demand image resizing and format conversion" \
  "Serve resized images and modern formats (WebP, AVIF) on demand with edge caching." \
  "- Requests must be served from edge cache when possible
- Output must fall back to JPEG when WebP/AVIF is unsupported
- Image dimensions must be capped to prevent abuse" \
  nonfunctional

make_spec "Full-Text Search" \
  "Postgres trigram index with ranked results" \
  "Provide full-text search across products and articles using Postgres trigram and ranked results." \
  "- Results must be ranked by trigram similarity
- Search must complete within 200ms at p95
- Highlighted snippets must show the matched terms" \
  functional

make_spec "Recommendations Engine" \
  "Item-to-item collaborative filtering with cold-start fallback" \
  "Surface related products based on user behavior, with a popularity fallback for new users." \
  "- New users must see popularity-based recommendations
- Returning users must see personalized recommendations
- Recommendations must be regenerated nightly" \
  business

make_spec "Analytics Event Pipeline" \
  "Client and server event ingestion into the warehouse" \
  "Capture analytics events from web, mobile, and backend services into a unified event stream and warehouse." \
  "- Events must be batched on the client to reduce overhead
- Server-emitted events must include user and session context
- Lost events must be re-deliverable from the buffer" \
  nonfunctional

make_spec "Funnel Reports" \
  "Conversion funnel analytics with drop-off attribution" \
  "Build conversion funnel reports for product and growth teams with step-level drop-off attribution." \
  "- Reports must support multi-step funnels
- Drop-off must be attributable to last completed step
- Reports must filter by user cohort and date range" \
  business

make_spec "Feature Flags" \
  "Runtime feature toggles with targeting rules" \
  "Allow rollout of new features behind flags with targeting by user, account, and percentage." \
  "- Flags must evaluate in under 5ms locally
- Targeting rules must support user, account, and percentage
- Flag changes must propagate within one minute" \
  nonfunctional

make_spec "A/B Testing Framework" \
  "Bucket users into experiment variants and report lift" \
  "Run controlled experiments with deterministic bucketing and lift reporting against primary metrics." \
  "- Bucketing must be sticky per user
- Variant exposure must be logged for each user
- Lift reports must include confidence intervals" \
  business

make_spec "Dashboard Builder" \
  "Drag-and-drop dashboards with saved views" \
  "Let internal users build dashboards from a library of charts and save them per team." \
  "- Users must be able to drag, drop, and resize widgets
- Dashboards must be shareable within the team
- Saved views must persist across sessions" \
  functional

make_spec "Cross-Service Tracing" \
  "Distributed tracing with W3C tracecontext propagation" \
  "Adopt OpenTelemetry across services with W3C tracecontext propagation and sampling." \
  "- Tracecontext must propagate across HTTP and async boundaries
- Sampling rate must be configurable at runtime
- Traces must be queryable by user and request ID" \
  nonfunctional

make_spec "Backup and Restore" \
  "Daily encrypted backups with quarterly restore drills" \
  "Take daily encrypted backups of primary databases and validate restore procedures every quarter." \
  "- Backups must be encrypted at rest with a managed key
- Backups must be retained for 30 days
- A documented restore drill must run every quarter" \
  nonfunctional

make_spec "Disaster Recovery" \
  "Multi-region failover with documented RPO and RTO" \
  "Stand up a secondary region for failover with documented Recovery Point and Recovery Time objectives." \
  "- Replica lag must be monitored and alerted
- Failover must complete within the documented RTO
- Data loss must not exceed the documented RPO" \
  nonfunctional

make_spec "Service-Level Objectives" \
  "SLO definitions with error budgets and burn alerts" \
  "Define availability and latency SLOs per critical service with burn-rate alerting on error budgets." \
  "- Each critical service must have an availability and a latency SLO
- Error budgets must be tracked per quarter
- Burn-rate alerts must page on-call when budgets are at risk" \
  nonfunctional

make_spec "Localization Framework" \
  "i18n strings with locale-aware formatting" \
  "Externalize user-facing strings, add locale-aware date and number formatting, and support right-to-left layouts." \
  "- All user-facing strings must be externalized
- Dates and numbers must format per locale
- Right-to-left locales must render correctly" \
  functional

make_spec "Translation Workflow" \
  "Continuous translation pipeline with vendor sync" \
  "Sync source strings to a translation vendor and pull completed translations back automatically." \
  "- New strings must reach the vendor within one hour
- Completed translations must be merged automatically
- Missing translations must fall back to the source locale" \
  business

make_spec "Onboarding Tour" \
  "First-run product tour with progress tracking" \
  "Guide new users through key features with an interactive tour and persist progress per account." \
  "- Tour progress must persist across sessions
- Users must be able to dismiss and restart the tour
- Completion must be tracked for analytics" \
  business

make_spec "In-App Messaging" \
  "Targeted in-app messages and announcements" \
  "Show targeted product updates and announcements inside the app with rule-based targeting." \
  "- Messages must target by plan, role, and country
- Users must be able to dismiss messages permanently
- Engagement must be tracked per message" \
  business

make_spec "Help Center Integration" \
  "Searchable help articles and contextual links" \
  "Embed a searchable help center with contextual links from in-app screens." \
  "- Searches must surface results within 300ms
- Contextual links must open the relevant article
- Article views must be tracked for analytics" \
  functional

make_spec "Customer Support Tickets" \
  "Ticket creation with attachment support" \
  "Let customers raise support tickets directly with attachments and ticket history." \
  "- Customers must be able to attach files up to 25MB
- Ticket history must be visible per customer
- Status changes must trigger email notifications" \
  functional

make_spec "Mobile App Crash Reporting" \
  "Crash and ANR capture for iOS and Android" \
  "Capture crashes and ANRs from mobile apps with symbolicated stack traces and release tagging." \
  "- Stack traces must be symbolicated automatically
- Crashes must be grouped by signature
- Releases must be tagged for regression detection" \
  nonfunctional

make_spec "Performance Budget" \
  "Frontend performance budgets enforced in CI" \
  "Set performance budgets for bundle size and Core Web Vitals, enforced in CI on every PR." \
  "- Bundle size regressions must fail the build
- Core Web Vitals must be measured per route
- Budgets must be configurable per route" \
  nonfunctional

make_spec "Background Job Queue" \
  "Reliable background job processing with priorities" \
  "Run background jobs across services with priorities, retries, and dead-letter handling." \
  "- High-priority jobs must run before normal jobs
- Failed jobs must move to a dead-letter queue after retries
- Job status must be queryable per job ID" \
  nonfunctional

make_spec "Scheduled Reports" \
  "User-configurable scheduled exports via email" \
  "Let users schedule reports to be emailed on a recurring cadence." \
  "- Users must be able to schedule reports daily, weekly, or monthly
- Reports must be delivered as CSV or PDF
- Failed deliveries must alert the owner" \
  business

make_spec "Data Export API" \
  "Async export of large datasets with download URLs" \
  "Expose async export endpoints for large datasets, returning a signed download URL when ready." \
  "- Exports must run asynchronously and return a job ID
- Download URLs must expire after 24 hours
- Users must be able to cancel an in-flight export" \
  functional

make_spec "Mobile Deep Linking" \
  "Universal links and Android App Links routing" \
  "Route inbound URLs to the right in-app screen on iOS and Android with fallback to the web." \
  "- Universal links must open the correct screen on iOS
- App Links must open the correct screen on Android
- Unknown links must fall back to the web" \
  functional

make_spec "GDPR Data Export" \
  "User-initiated personal-data export within 30 days" \
  "Provide users a way to export all personal data we hold, fulfilled within the GDPR deadline." \
  "- Users must be able to request an export from settings
- Export must be delivered within 30 days
- Export must include all data referenced by the user's ID" \
  nonfunctional

make_spec "GDPR Data Deletion" \
  "User-initiated account deletion with data purge" \
  "Allow users to delete their account, purging personal data while retaining anonymized analytics." \
  "- Account deletion must purge personal data within 30 days
- Anonymized analytics may be retained
- Deletion must cascade to associated tenants" \
  nonfunctional

make_spec "Cookie Consent" \
  "Region-aware cookie consent banner with category opt-in" \
  "Show a region-aware cookie consent banner with per-category opt-in for analytics and marketing." \
  "- Consent must be remembered per user/browser
- Analytics must respect the user's consent
- Marketing pixels must respect the user's consent" \
  nonfunctional

# -- Additional specs to push each category over the default page size of 20 --

make_spec "Referral Program" \
  "Invite-a-friend rewards with attribution and fraud checks" \
  "Reward customers for inviting friends with code-based attribution and basic fraud detection." \
  "- Each customer must have a unique referral code
- Successful conversions must credit the referrer
- Self-referrals and obvious fraud must be blocked" \
  business

make_spec "Loyalty Points" \
  "Earn-and-burn points across orders and engagement" \
  "Award points for purchases and key actions, redeemable at checkout." \
  "- Points must accrue on order completion
- Points must be redeemable up to a configurable cap per order
- Points balance must be visible in the customer portal" \
  business

make_spec "Promo Codes" \
  "Discount codes with stacking rules and expiry" \
  "Issue and validate promo codes with stacking rules, usage caps, and expiry dates." \
  "- Codes must validate against active campaigns
- Usage caps must be enforced atomically
- Expired codes must reject with a clear message" \
  business

make_spec "Gift Cards" \
  "Stored-value gift cards with PIN-protected redemption" \
  "Sell and redeem stored-value gift cards with PIN-protected balance lookups." \
  "- Cards must be PIN-protected at lookup
- Redemption must atomically decrement the balance
- Balances must be queryable in the customer portal" \
  business

make_spec "Marketing Email Campaigns" \
  "Segmented broadcast emails with unsubscribe handling" \
  "Schedule broadcast emails to user segments with one-click unsubscribe and preference management." \
  "- Unsubscribe must take effect within 24 hours
- Each campaign must record open and click rates
- Suppressed addresses must be excluded automatically" \
  business

make_spec "Affiliate Tracking" \
  "Cookie-based affiliate attribution with payouts" \
  "Track affiliate-driven sign-ups and orders for periodic payout reconciliation." \
  "- Affiliate clicks must drop a cookie with a 30-day window
- Conversions must attribute to the most recent affiliate
- Payout reports must be exportable per affiliate" \
  business

make_spec "Pricing Plans" \
  "Define and version customer-facing pricing tiers" \
  "Maintain customer-facing pricing tiers with grandfathering for existing customers." \
  "- Plan changes must not affect grandfathered customers
- Pricing pages must reflect the active plan version
- Internal admins must preview changes before publishing"

make_spec "Notification Preferences" \
  "Per-channel and per-category user notification settings" \
  "Let users opt in or out of notifications per channel (email, push, SMS) and per category." \
  "- Settings must be respected by every notification sender
- Defaults must follow the documented policy
- Changes must take effect within one minute" \
  functional

make_spec "Saved Filters and Views" \
  "Reusable list filters and saved views for power users" \
  "Allow users to save list filters and reuse them as named views." \
  "- Views must be private by default with optional sharing
- Saved filters must include sort and column choices
- Default view must be selectable per list" \
  functional

make_spec "Bulk Actions" \
  "Multi-select toolbar for batch operations on lists" \
  "Provide a multi-select toolbar with batch actions (assign, archive, tag) on list views." \
  "- Selecting all must respect the active filter
- Bulk actions must show progress and per-item errors
- Each action must be undoable for at least one minute" \
  functional

make_spec "Two-Factor Authentication" \
  "TOTP and recovery-code based 2FA" \
  "Let users enroll TOTP authenticators and store one-time recovery codes." \
  "- Users must be able to enroll a TOTP authenticator
- Recovery codes must be one-time use
- Lost-device flow must require email verification" \
  functional

make_spec "Account Recovery" \
  "Self-serve password reset and account recovery flows" \
  "Provide secure password reset and account recovery without contacting support." \
  "- Reset links must expire within 30 minutes
- Recovery must require verification of an alternate channel
- Failed attempts must be rate-limited" \
  functional

make_spec "API Key Management" \
  "Self-serve API keys with scopes and rotation" \
  "Let customers manage personal and service API keys with scoped permissions." \
  "- Keys must be hashed at rest
- Each key must be revocable independently
- Scopes must be enforced on every request" \
  functional

make_spec "Org and Team Management" \
  "Multi-user organizations with team-level roles" \
  "Support multi-user organizations with team-level role assignments and invitations." \
  "- Owners must be able to invite members by email
- Roles must be assignable per team
- Removed members must lose access immediately" \
  functional

make_spec "Audit Trail UI" \
  "Searchable audit log viewer for admins" \
  "Surface audit events in a searchable, filterable UI for compliance reviewers." \
  "- Events must be filterable by user, action, and time range
- Pagination must support stable cursors
- Export must produce CSV within five minutes" \
  functional

make_spec "Tagging System" \
  "Free-form tags across major entities with autocomplete" \
  "Allow tagging of orders, customers, and tickets with autocomplete from existing tags." \
  "- Tags must be case-insensitive on lookup
- Autocomplete must surface the top 20 matches
- Bulk re-tag must be supported" \
  functional

make_spec "Comments and Mentions" \
  "Threaded comments with @mention notifications" \
  "Add threaded comments to records with @mention email and in-app notifications." \
  "- Mentions must notify the mentioned user
- Comment edits must show an edited indicator
- Soft-delete must hide content from non-authors" \
  functional

make_spec "Cache Invalidation" \
  "Tag-based cache invalidation across services" \
  "Adopt a tag-based cache invalidation strategy so dependent caches update together." \
  "- Writes must invalidate all tagged entries
- Stale reads must be bounded to the documented TTL
- Invalidation events must be observable" \
  nonfunctional

make_spec "Database Migrations" \
  "Zero-downtime, expand-contract migration pattern" \
  "Adopt expand-contract migrations so schema changes ship without downtime." \
  "- Each migration must be reversible
- Long-running migrations must run online
- CI must verify migrations on a representative dataset" \
  nonfunctional

make_spec "Secret Management" \
  "Centralized secrets with rotation and access logs" \
  "Move all secrets into a managed vault with rotation and per-access auditing." \
  "- Secrets must rotate on the documented schedule
- All access must be audited
- No secret may be committed to source control" \
  nonfunctional

make_spec "Container Image Hardening" \
  "Minimal, signed container images with vulnerability scans" \
  "Build minimal, signed container images and fail the pipeline on critical CVEs." \
  "- Images must be built from minimal base images
- Each image must be signed with cosign
- Critical CVEs must fail the CI pipeline" \
  nonfunctional

make_spec "WAF and Bot Mitigation" \
  "Web application firewall with bot scoring" \
  "Front user-facing endpoints with a WAF that includes managed rules and bot scoring." \
  "- Managed rule sets must be enabled at the edge
- Bot traffic must be scored and rate-limited
- Bypass paths must be allow-listed deliberately" \
  nonfunctional

make_spec "DDoS Protection" \
  "Edge-level DDoS mitigation with on-call runbooks" \
  "Use edge-level DDoS mitigation and document on-call response runbooks." \
  "- Layer 3/4 attacks must be absorbed at the edge
- Layer 7 attacks must trigger automatic mitigation
- Runbooks must cover incident escalation"

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
echo "-- Retyping infrastructure specs as nonfunctional"
$SPECD update-spec --id "SPEC-9"  --type "nonfunctional"
$SPECD update-spec --id "SPEC-10" --type "nonfunctional"

echo ""
echo "=== Creating tasks ==="

# Helper: tasks with a Description (derived from the summary) followed by a
# three-criterion checklist. Use make_task_n for tasks that need a different
# number of criteria or pre-checked state.
make_task() {
  SPEC_ID="$1"; TITLE="$2"; SUMMARY="$3"; A="$4"; B="$5"; C="$6"
  $SPECD new-task --spec-id "$SPEC_ID" \
    --title "$TITLE" \
    --summary "$SUMMARY" \
    --body "## Description

$SUMMARY. This task is part of the work captured by $SPEC_ID and should be
treated as done only when every checkbox below is ticked.

## Acceptance Criteria

- [ ] $A
- [ ] $B
- [ ] $C"
}

# Helper: tasks with raw Markdown criteria lines (e.g. \`- [x] done\`,
# \`- [ ] todo\`). Pass the criteria as one preformatted block in arg 4.
make_task_n() {
  SPEC_ID="$1"; TITLE="$2"; SUMMARY="$3"; CRITERIA_BLOCK="$4"
  $SPECD new-task --spec-id "$SPEC_ID" \
    --title "$TITLE" \
    --summary "$SUMMARY" \
    --body "## Description

$SUMMARY. This task is part of the work captured by $SPEC_ID and should be
treated as done only when every checkbox below is ticked.

## Acceptance Criteria

$CRITERIA_BLOCK"
}

# SPEC-1 — User Authentication: 3 tasks across stages
make_task_n SPEC-1 "Wire up Google OAuth2 provider" \
  "Add the Google identity provider with consent flow and token exchange" \
  "- [x] Register OAuth2 client in Google Cloud Console
- [x] Implement /auth/google/callback handler
- [ ] Persist Google profile fields on first login
- [ ] Add integration tests for the callback handler"

make_task_n SPEC-1 "Wire up GitHub OAuth2 provider" \
  "Add the GitHub identity provider with consent flow and token exchange" \
  "- [x] Register OAuth App in GitHub
- [ ] Implement /auth/github/callback handler
- [ ] Match existing users by primary email
- [ ] Add integration tests for the callback handler"

make_task_n SPEC-1 "Issue JWT session tokens after login" \
  "Sign and return a short-lived JWT plus refresh token on successful auth" \
  "- [ ] Generate signed JWT with 15-minute expiry
- [ ] Set HTTP-only secure cookie for access token
- [ ] Issue refresh token paired with access token
- [ ] Reject expired or tampered tokens"

# SPEC-2 — Session Management: 2 tasks
make_task_n SPEC-2 "Implement refresh token rotation" \
  "Rotate the refresh token on every use and invalidate the prior chain on theft" \
  "- [ ] Rotate refresh token on each /auth/refresh call
- [ ] Detect reuse of an already-rotated refresh token
- [ ] Invalidate the entire refresh chain on suspected theft
- [ ] Emit audit event when a chain is invalidated"

make_task_n SPEC-2 "Invalidate sessions on password change" \
  "Kill all active refresh tokens for a user when their password changes" \
  "- [x] Hook password-change endpoint to session-invalidate routine
- [ ] Confirm concurrent device sessions are killed
- [ ] Notify the user via email of the global logout"

# SPEC-3 — RBAC: 2 tasks
make_task_n SPEC-3 "Define role schema and seed default roles" \
  "Create roles table and seed admin, editor, viewer with permission mappings" \
  "- [x] Create roles and role_permissions tables
- [x] Seed admin, editor, viewer roles
- [ ] Document the permission matrix in /docs"

make_task_n SPEC-3 "Authorization middleware for protected routes" \
  "Block requests that lack the permission required by the route" \
  "- [ ] Build middleware that resolves the user's role from the JWT
- [ ] Return 403 with structured JSON when permission is missing
- [ ] Cover admin/editor/viewer happy paths in tests"

# SPEC-4 — Rate Limiting: 1 task
make_task_n SPEC-4 "Implement sliding-window rate limiter" \
  "Per-user and per-IP sliding window limiter backed by Redis" \
  "- [ ] Sliding window with 1-second resolution
- [ ] Configurable limits per route from .yaml
- [ ] Return 429 with Retry-After header
- [ ] Skip enforcement for admin tokens"

# SPEC-5 — Audit Logging: 2 tasks
make_task_n SPEC-5 "Emit structured audit logs for auth events" \
  "JSON-formatted audit events for login, logout, and failed attempts" \
  "- [x] Emit audit_event for successful login
- [x] Emit audit_event for failed login with IP
- [ ] Emit audit_event for logout
- [ ] Include trace ID in every event"

make_task_n SPEC-5 "Build audit log search UI" \
  "Filter audit events by user, action, and time range" \
  "- [ ] Server-side filtering by user, action, time range
- [ ] Pagination with page size 50
- [ ] CSV export of filtered results"

# SPEC-6 — Invoice Generation: 1 task
make_task_n SPEC-6 "Render PDF invoices from order data" \
  "Use the approved template to render a downloadable PDF per order" \
  "- [ ] Pull line items, tax, and total from the orders table
- [ ] Render PDF matching the approved template
- [ ] Upload PDF to object storage with a 30-day signed URL"

# SPEC-7 — Dark Mode Toggle: 1 task
make_task_n SPEC-7 "Persist theme preference across sessions" \
  "Store the dark/light selection in localStorage and respect prefers-color-scheme" \
  "- [x] Save selection in localStorage under specd-theme
- [x] Restore selection on page load
- [ ] Fall back to prefers-color-scheme on first visit"

# SPEC-9 — GraphQL Gateway: 1 task
make_task_n SPEC-9 "Set up dataloader batching" \
  "Batch downstream service calls per request to avoid N+1 queries" \
  "- [ ] DataLoader instance per request
- [ ] Cover order, inventory, and user services
- [ ] Verify N+1 queries are eliminated under load"

echo ""
echo "=== Creating bulk tasks (against newer specs) ==="

# SPEC-12 Email Delivery Pipeline
make_task SPEC-12 "Integrate provider SDK" "Add the provider SDK and verify a test send" \
  "SDK installed and credentials wired" "Test email delivers to a sandbox inbox" "Failures surface as actionable errors"
make_task SPEC-12 "Implement bounce handler" "Process bounce webhooks and flag addresses" \
  "Hard bounces flag the address" "Soft bounces allow retry" "Bounce events are logged"

# SPEC-13 Push Notification Service
make_task SPEC-13 "Wire FCM credentials" "Configure Firebase Cloud Messaging" \
  "FCM service account loaded from secrets" "Sample push delivers to a test device" "Failures are logged with reason"
make_task SPEC-13 "Wire APNs credentials" "Configure Apple Push Notification service" \
  "APNs key loaded from secrets" "Sample push delivers to a test device" "Tokens that fail are pruned"
make_task SPEC-13 "Quiet hours scheduler" "Defer pushes during user-configured quiet hours" \
  "Per-user quiet windows are stored" "Pushes during quiet hours are deferred" "Deferred pushes are released afterwards"

# SPEC-14 Webhook Delivery
make_task SPEC-14 "HMAC signing of payloads" "Sign every outbound payload with HMAC-SHA256" \
  "Signature header is included on every send" "Signatures verify against the documented algorithm" "Customers can rotate their signing secret"
make_task SPEC-14 "Retry with exponential backoff" "Retry failed deliveries up to 24 hours" \
  "Failed deliveries enqueue a retry" "Backoff doubles up to a cap" "Permanent failures move to dead letter"

# SPEC-15 Stripe Payment Integration
make_task SPEC-15 "Checkout session creation" "Create Stripe Checkout sessions for orders" \
  "Sessions are created with the order amount" "Session URL is returned to the client" "Cancelled sessions revert the order"
make_task SPEC-15 "Webhook signature verification" "Verify Stripe signatures before processing" \
  "Invalid signatures return 400" "Valid signatures are processed once" "Replays are rejected"
make_task SPEC-15 "Refund flow" "Issue refunds via Stripe and update orders" \
  "Refund call returns refund ID" "Order moves to refunded on success" "Failed refunds surface to support"

# SPEC-16 Subscription Billing
make_task SPEC-16 "Plan upgrade proration" "Prorate the current period on plan upgrades" \
  "Upgrades calculate prorated charge" "Charges are applied immediately" "Receipts list both line items"
make_task SPEC-16 "Dunning flow" "Run a 14-day dunning window for failed renewals" \
  "Failed renewals start dunning" "Reminders are sent on a schedule" "Cancellation occurs at end of window"

# SPEC-17 Customer Portal
make_task SPEC-17 "Update default payment method" "Let customers replace their default card" \
  "Form validates the new card" "Default card is updated atomically" "Old card remains attached but inactive"
make_task SPEC-17 "Download past invoices" "Expose downloadable PDF invoices" \
  "List of invoices is paginated" "Each invoice downloads as PDF" "Access is restricted to the customer"

# SPEC-18 File Upload Service
make_task SPEC-18 "Pre-signed URL endpoint" "Issue short-lived S3 pre-signed URLs" \
  "URLs expire in 15 minutes" "URLs encode content type and size limit" "Endpoint is authenticated"
make_task SPEC-18 "Virus scanning" "Scan uploads before they go live" \
  "Files are scanned on upload" "Infected files are quarantined" "Clean files are made available"

# SPEC-19 Image Optimization
make_task SPEC-19 "Resize and reformat handler" "Serve resized images on demand" \
  "Width and quality are bounded" "Output content-type matches request" "Cache headers are set per response"

# SPEC-20 Full-Text Search
make_task SPEC-20 "Trigram index migration" "Add a Postgres trigram index on searchable columns" \
  "Migration is reversible" "Index covers product and article tables" "Query plan uses the index"
make_task SPEC-20 "Snippet highlighter" "Render highlighted match snippets" \
  "Matches are wrapped in mark tags" "Snippet length is bounded" "HTML is escaped before highlighting"

# SPEC-21 Recommendations Engine
make_task SPEC-21 "Cold-start popularity model" "Serve popular items to new users" \
  "Top-N popular items are precomputed" "Cold-start users get popular items" "Refresh runs daily"

# SPEC-22 Analytics Event Pipeline
make_task SPEC-22 "Client batching" "Batch events on the client to reduce overhead" \
  "Events are flushed every 10 seconds or 50 events" "Failed batches retry on next flush" "Batch size is configurable"
make_task SPEC-22 "Server-side context enrichment" "Attach user and session to server events" \
  "User ID is attached when authenticated" "Session ID is attached for all events" "Anonymous events keep an anonymous ID"

# SPEC-23 Funnel Reports
make_task SPEC-23 "Multi-step funnel query" "Compute multi-step funnels per cohort" \
  "Funnel returns counts at each step" "Drop-off is reported per step" "Cohort filter is honored"

# SPEC-24 Feature Flags
make_task SPEC-24 "Local evaluator" "Evaluate flags locally with sub-5ms latency" \
  "Evaluator runs in under 5ms" "Cache refreshes within one minute" "Targeting rules are honored"

# SPEC-25 A/B Testing Framework
make_task SPEC-25 "Sticky bucketing" "Bucket users deterministically per experiment" \
  "Bucketing is stable across sessions" "Bucketing supports salting per experiment" "Variant exposure is logged"

# SPEC-26 Dashboard Builder
make_task SPEC-26 "Drag-and-drop grid" "Implement the dashboard grid layout" \
  "Widgets can be dragged and resized" "Layout persists per user" "Mobile view stacks widgets"

# SPEC-27 Cross-Service Tracing
make_task SPEC-27 "Tracecontext propagation middleware" "Forward traceparent headers across HTTP calls" \
  "Inbound traceparent is honored" "Outbound traceparent is propagated" "Sampling decisions are respected"
make_task SPEC-27 "Async boundary propagation" "Carry tracecontext across queues and jobs" \
  "Producers attach traceparent to messages" "Consumers re-establish context" "Spans are linked across boundaries"

# SPEC-28 Backup and Restore
make_task SPEC-28 "Daily encrypted backup job" "Run a nightly encrypted backup of primary databases" \
  "Backups are encrypted with a managed key" "Backups are stored in a separate region" "Job alerts on failure"
make_task SPEC-28 "Quarterly restore drill" "Document and run a quarterly restore drill" \
  "Drill restores into a clean environment" "Restore is verified by smoke tests" "Drill is logged in the runbook"

# SPEC-29 Disaster Recovery
make_task SPEC-29 "Replica lag monitor" "Monitor and alert on cross-region replica lag" \
  "Lag is exported as a metric" "Alerts fire above the documented threshold" "Runbook covers next steps"

# SPEC-30 Service-Level Objectives
make_task SPEC-30 "Define SLOs for critical services" "Author availability and latency SLOs per critical service" \
  "Each critical service has defined SLOs" "Targets are documented in the runbook" "Owners are named per SLO"
make_task SPEC-30 "Burn-rate alerts" "Page on-call when error budgets are at risk" \
  "Multi-window burn-rate alerts are configured" "Alerts route to the correct on-call" "Alerts have linked runbooks"

# SPEC-31 Localization Framework
make_task SPEC-31 "Externalize user-facing strings" "Move hard-coded strings into the i18n catalog" \
  "All visible strings come from the catalog" "Missing keys fall back to source locale" "CI fails on hard-coded strings"
make_task SPEC-31 "Locale-aware formatting" "Format dates and numbers per locale" \
  "Dates use the current locale's format" "Numbers use locale-specific separators" "Currency uses ISO 4217 codes"

# SPEC-32 Translation Workflow
make_task SPEC-32 "Vendor sync job" "Push new strings to and pull translations from the vendor" \
  "New strings reach the vendor within one hour" "Completed translations merge automatically" "Sync errors page the owner"

# SPEC-33 Onboarding Tour
make_task SPEC-33 "Tour engine" "Render and progress through tour steps" \
  "Steps anchor to target elements" "Progress persists per user" "Users can dismiss the tour"

# SPEC-34 In-App Messaging
make_task SPEC-34 "Targeting evaluator" "Evaluate plan, role, and country targeting rules" \
  "Rules combine with AND/OR semantics" "Evaluation runs locally for speed" "Rule changes propagate within a minute"

# SPEC-35 Help Center Integration
make_task SPEC-35 "Help search box" "Search help articles inline" \
  "Searches return within 300ms" "Results are ranked by relevance" "Empty results show a helpful prompt"

# SPEC-36 Customer Support Tickets
make_task SPEC-36 "Ticket creation form" "Let customers create a ticket with attachments" \
  "Form supports up to 25MB attachments" "Tickets are linked to the customer" "Confirmation email is sent on create"
make_task SPEC-36 "Ticket history view" "Show ticket history per customer" \
  "History is paginated" "Status changes are visible" "Owners are visible per ticket"

# SPEC-37 Mobile App Crash Reporting
make_task SPEC-37 "Symbolication pipeline" "Symbolicate iOS and Android crashes" \
  "Symbol files upload from CI" "Stack traces resolve to source lines" "Missing symbols are flagged"

# SPEC-38 Performance Budget
make_task SPEC-38 "Bundle size budget" "Fail CI on bundle size regressions" \
  "Budgets are configured per route" "CI fails when budgets are exceeded" "Reports show the contributing modules"
make_task SPEC-38 "Core Web Vitals capture" "Capture CWV per route from real users" \
  "Vitals are reported to analytics" "Per-route summaries are queryable" "Regressions trigger alerts"

# SPEC-39 Background Job Queue
make_task SPEC-39 "Priority queue" "Process high-priority jobs ahead of normal jobs" \
  "Priority levels are defined" "Workers honor priority order" "Starvation is bounded"
make_task SPEC-39 "Dead-letter queue" "Move repeatedly failing jobs to a DLQ" \
  "Jobs move after the retry budget is spent" "DLQ items are inspectable" "Replay tooling exists"

# SPEC-40 Scheduled Reports
make_task SPEC-40 "Schedule editor" "Let users configure cadence and recipients" \
  "Cadence supports daily/weekly/monthly" "Recipients accept multiple emails" "Schedules can be paused"

# SPEC-41 Data Export API
make_task SPEC-41 "Async export endpoint" "Kick off async exports and return a job ID" \
  "Endpoint returns a job ID immediately" "Job status is queryable" "Cancellation is supported"

# SPEC-42 Mobile Deep Linking
make_task SPEC-42 "Universal links setup" "Configure iOS Universal Links" \
  "Apple App Site Association file is served" "Links open the correct screen" "Fallback to web works for unknowns"

# SPEC-43 GDPR Data Export
make_task SPEC-43 "Personal-data exporter" "Aggregate all personal data per user" \
  "Exporter covers every personal-data table" "Output is a single archive" "Export job runs asynchronously"

# SPEC-44 GDPR Data Deletion
make_task SPEC-44 "Delete-account flow" "Delete the user's data within 30 days" \
  "Settings page exposes delete account" "Deletion job runs asynchronously" "Cascade covers all referenced tables"

# SPEC-45 Cookie Consent
make_task SPEC-45 "Consent banner" "Show a region-aware consent banner" \
  "Banner appears for new visitors" "Choice is remembered per browser" "Per-category opt-in is supported"

echo ""
echo "-- Moving a sample of tasks to non-default stages for variety"
$SPECD update-task --id "TASK-1"  --status "in_progress" || true
$SPECD update-task --id "TASK-3"  --status "todo" || true
$SPECD update-task --id "TASK-6"  --status "done" || true
$SPECD update-task --id "TASK-9"  --status "blocked" || true
$SPECD update-task --id "TASK-15" --status "in_progress" || true
$SPECD update-task --id "TASK-18" --status "todo" || true
$SPECD update-task --id "TASK-21" --status "done" || true
$SPECD update-task --id "TASK-22" --status "in_progress" || true
$SPECD update-task --id "TASK-25" --status "blocked" || true
$SPECD update-task --id "TASK-28" --status "pending_verification" || true
$SPECD update-task --id "TASK-30" --status "todo" || true
$SPECD update-task --id "TASK-33" --status "cancelled" || true
$SPECD update-task --id "TASK-37" --status "done" || true
$SPECD update-task --id "TASK-40" --status "in_progress" || true
$SPECD update-task --id "TASK-44" --status "wont_fix" || true
$SPECD update-task --id "TASK-50" --status "done" || true
$SPECD update-task --id "TASK-55" --status "in_progress" || true
$SPECD update-task --id "TASK-60" --status "blocked" || true

echo ""
echo "=== Seeding KB documents ==="
# KB sync from markdown is not yet implemented, so insert directly into the
# cache DB. Search and the Web UI both read from these tables, so this
# produces realistic, searchable KB content for QA.

NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
sqlite3 $DB <<SQL
INSERT INTO kb_docs (id, title, summary, source_type, path, content_hash, added_at, added_by) VALUES
  ('KB-1', 'OAuth2 Implementation Guide', 'Step-by-step OAuth2 authorization-code flow with PKCE', 'md', 'kb/KB-1.md', 'h-kb1', '$NOW', 'qa-tester'),
  ('KB-2', 'JWT Best Practices', 'Signing algorithms, expiry, rotation, and revocation patterns', 'md', 'kb/KB-2.md', 'h-kb2', '$NOW', 'qa-tester'),
  ('KB-3', 'Rate Limiting Strategies', 'Token bucket vs sliding window algorithms with trade-offs', 'md', 'kb/KB-3.md', 'h-kb3', '$NOW', 'qa-tester'),
  ('KB-4', 'PDF Generation with HeadlessChrome', 'Render HTML templates to PDF using Chrome DevTools Protocol', 'md', 'kb/KB-4.md', 'h-kb4', '$NOW', 'qa-tester'),
  ('KB-5', 'GraphQL Schema Stitching', 'Combining schemas from multiple services into a unified gateway', 'md', 'kb/KB-5.md', 'h-kb5', '$NOW', 'qa-tester'),
  ('KB-6', 'Audit Logging for Compliance', 'SOC2 and GDPR retention requirements with structured event design', 'md', 'kb/KB-6.md', 'h-kb6', '$NOW', 'qa-tester');

INSERT INTO kb_chunks (doc_id, position, summary, text, char_start, char_end) VALUES
  ('KB-1', 0, 'Authorization-code flow overview', 'OAuth2 authorization-code flow exchanges a short-lived code for an access token. Always use PKCE on public clients to prevent authorization-code interception attacks.', 0, 200),
  ('KB-1', 1, 'Refresh token rotation guidance', 'Refresh tokens should rotate on every use. Detecting reuse of a rotated refresh token is the canonical signal of a stolen credential — invalidate the entire chain immediately.', 0, 220),
  ('KB-2', 0, 'Choosing a JWT signing algorithm', 'Prefer asymmetric algorithms like RS256 or EdDSA so verifiers do not need the signing key. Avoid HS256 in distributed systems where many services must verify tokens.', 0, 220),
  ('KB-2', 1, 'JWT expiry and revocation', 'Keep access tokens short-lived (5–15 minutes). For revocation, pair access tokens with a refresh token whose rotation chain you can invalidate centrally.', 0, 200),
  ('KB-3', 0, 'Sliding window vs token bucket', 'Sliding window rate limiters give smoother throughput but cost more memory. Token bucket is cheap but allows burstiness up to the bucket size.', 0, 200),
  ('KB-3', 1, 'Per-IP vs per-user limiting', 'Anonymous traffic should be limited per source IP; authenticated traffic should be limited per user identity to avoid penalising users behind shared NATs.', 0, 200),
  ('KB-4', 0, 'Rendering PDFs with Chromium', 'Chromium in headless mode renders HTML to PDF deterministically. Drive it via the Chrome DevTools Protocol and pin the binary version for reproducible output.', 0, 220),
  ('KB-5', 0, 'Schema stitching basics', 'A GraphQL gateway merges remote schemas using type extensions and delegations. Use dataloader on the gateway to batch downstream calls and avoid N+1 queries.', 0, 220),
  ('KB-6', 0, 'Audit event shape', 'Every audit event should include actor, action, resource, outcome, IP, user-agent, and a trace ID. Emit as one structured JSON line per event.', 0, 200),
  ('KB-6', 1, 'Retention and immutability', 'SOC2 typically requires 12 months of immutable audit logs. Store on append-only storage and back up off-site to satisfy availability requirements.', 0, 200);
SQL

echo ""
echo "=== Verification ==="
echo ""
echo "--- Specs in DB ---"
sqlite3 $DB "SELECT id, type, title FROM specs ORDER BY id;"

echo ""
echo "--- Links ---"
sqlite3 $DB "SELECT from_spec, to_spec FROM spec_links ORDER BY from_spec, to_spec;"

echo ""
echo "--- Tasks in DB ---"
sqlite3 $DB "SELECT id, spec_id, status, title FROM tasks ORDER BY id;"

echo ""
echo "--- KB docs in DB ---"
sqlite3 $DB "SELECT id, title FROM kb_docs ORDER BY id;"

echo ""
echo "--- KB chunks per doc ---"
sqlite3 $DB "SELECT doc_id, COUNT(*) AS chunks FROM kb_chunks GROUP BY doc_id ORDER BY doc_id;"

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
