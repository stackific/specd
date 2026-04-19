# specd Test Cases

Comprehensive functional and non-functional test cases for specd. Since backend unit tests exist for workspace, DB, frontmatter, hash, and lock packages, this document focuses on **web UI (E2E)**, **API endpoints**, **integration flows**, **watcher behavior**, **CLI output**, and **non-functional requirements**.

---

## Table of Contents

1. [Web UI - Board (Kanban)](#1-web-ui---board-kanban)
2. [Web UI - Specs List](#2-web-ui---specs-list)
3. [Web UI - Spec Detail](#3-web-ui---spec-detail)
4. [Web UI - Task Detail](#4-web-ui---task-detail)
5. [Web UI - Knowledge Base List](#5-web-ui---knowledge-base-list)
6. [Web UI - KB Detail / Reader](#6-web-ui---kb-detail--reader)
7. [Web UI - Search](#7-web-ui---search)
8. [Web UI - Status](#8-web-ui---status)
9. [Web UI - Trash](#9-web-ui---trash)
10. [Web UI - Rejected Files](#10-web-ui---rejected-files)
11. [Web UI - Navigation & Layout](#11-web-ui---navigation--layout)
12. [Web UI - Theme & Dark Mode](#12-web-ui---theme--dark-mode)
13. [Web UI - Error Pages](#13-web-ui---error-pages)
14. [API Endpoints - KB](#14-api-endpoints---kb)
15. [API Endpoints - Board Drag-and-Drop](#15-api-endpoints---board-drag-and-drop)
16. [CLI - Init](#16-cli---init)
17. [CLI - Config](#17-cli---config)
18. [CLI - Spec Commands](#18-cli---spec-commands)
19. [CLI - Task Commands](#19-cli---task-commands)
20. [CLI - Criteria Commands](#20-cli---criteria-commands)
21. [CLI - Link Commands](#21-cli---link-commands)
22. [CLI - Depend Commands](#22-cli---depend-commands)
23. [CLI - Cite Commands](#23-cli---cite-commands)
24. [CLI - KB Commands](#24-cli---kb-commands)
25. [CLI - Search](#25-cli---search)
26. [CLI - Next](#26-cli---next)
27. [CLI - Maintenance](#27-cli---maintenance)
28. [CLI - Trash Commands](#28-cli---trash-commands)
29. [CLI - Serve & Watch](#29-cli---serve--watch)
30. [Watcher - Sync Behavior](#30-watcher---sync-behavior)
31. [Watcher - Deletion Handling](#31-watcher---deletion-handling)
32. [Watcher - Rejection Logic](#32-watcher---rejection-logic)
33. [Watcher - Debouncing](#33-watcher---debouncing)
34. [Integration - Spec Lifecycle](#34-integration---spec-lifecycle)
35. [Integration - Task Lifecycle](#35-integration---task-lifecycle)
36. [Integration - KB Lifecycle](#36-integration---kb-lifecycle)
37. [Integration - Citation Flow](#37-integration---citation-flow)
38. [Integration - Dependency Graph](#38-integration---dependency-graph)
39. [Integration - Search Across Entities](#39-integration---search-across-entities)
40. [Integration - Rebuild & Merge-Fixup](#40-integration---rebuild--merge-fixup)
41. [Non-Functional - Performance](#41-non-functional---performance)
42. [Non-Functional - Security](#42-non-functional---security)
43. [Non-Functional - Accessibility](#43-non-functional---accessibility)
44. [Non-Functional - Reliability & Data Integrity](#44-non-functional---reliability--data-integrity)
45. [Non-Functional - Responsiveness & Layout](#45-non-functional---responsiveness--layout)
46. [Non-Functional - Offline & Embedded Assets](#46-non-functional---offline--embedded-assets)

---

## 1. Web UI - Board (Kanban)

### 1.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.1.1 | Empty board renders | Navigate to `/` with no tasks | All 8 columns visible with headers and "No tasks" empty state in each | PASS |
| 1.1.2 | Board with tasks | Create tasks in various statuses | Tasks appear in correct columns with correct counts in badges | PASS |
| 1.1.3 | Column headers show icons | Load board | Each column displays its Material icon (inventory_2, checklist, pending, block, verified, check_circle, cancel, do_not_disturb_on) | PASS |
| 1.1.4 | Task card shows title | Create a task | Card displays task title as clickable link | PASS |
| 1.1.5 | Task card shows spec title | Create a task under a spec | Card displays parent spec title | PASS |
| 1.1.6 | Task card shows criteria progress | Create task with 3 criteria, check 2 | Card shows circular progress indicator reading "2/3" | PASS |
| 1.1.7 | Task card shows blocked icon | Create task with unresolved dependency | Warning icon visible on card | PASS |
| 1.1.8 | Task card shows citation icon | Create task with KB citation | Citation/link icon visible on card | PASS |
| 1.1.9 | Full-page load (no HX-Request) | Open `/` in new browser tab | Full HTML page with base layout, nav, footer rendered | PASS |
| 1.1.10 | HTMX partial load | Click "Board" in nav from another page | Only content block swapped, no full page reload | PASS |
| 1.1.11 | Column ordering left-to-right | Load board | Columns in order: Backlog, To Do, In Progress, Blocked, Verification, Done, Cancelled, Won't Fix | PASS |
| 1.1.12 | Column task count badge | Add 3 tasks to "todo" | "To Do" column header badge shows "3" | PASS |
| 1.1.13 | Task card shows created_by | Config user.name, create task | Card or tooltip shows creator name | SKIP |

### 1.2 Spec Filter

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.2.1 | Filter dropdown lists all specs | Create 3 specs, load board | Dropdown shows "All specs" + all 3 spec titles | PASS |
| 1.2.2 | Filter by spec | Select a spec from dropdown | Board shows only tasks belonging to selected spec | PASS |
| 1.2.3 | Clear filter | Select "All specs" after filtering | All tasks visible again | PASS |
| 1.2.4 | Filter preserves on htmx reload | Filter by spec, then perform board action | Filter dropdown retains selected spec after content swap | PASS |
| 1.2.5 | Filter with no matching tasks | Select spec with no tasks | All columns show "No tasks" empty state | PASS |

### 1.3 New Task Dialog (from Board)

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.3.1 | Dialog opens | Click "New Task" button | Dialog appears with form fields: spec_id, title, summary, status, body | PASS |
| 1.3.2 | Button hidden when no specs | Load board with zero specs | "New Task" button not rendered | SKIP |
| 1.3.3 | Spec dropdown populated | Open dialog with 3 specs | All specs listed in spec_id select | PASS |
| 1.3.4 | Status defaults to backlog | Open dialog | Status select pre-selects "backlog" | PASS |
| 1.3.5 | Valid submission | Fill all required fields, submit | Dialog closes, task appears in correct column, redirect to task detail | PASS |
| 1.3.6 | Empty title rejected | Submit with blank title | Form re-renders with error "Title must be at least 2 characters", HTTP 422 | PASS |
| 1.3.7 | Whitespace-only title rejected | Submit title "   " | Validation error displayed, form preserved | PASS |
| 1.3.8 | Title too short rejected | Submit title "A" | Validation error: title must be 2+ chars | PASS |
| 1.3.9 | Body too short rejected | Submit body "Short" | Validation error: body must be 20+ chars | PASS |
| 1.3.10 | Empty body rejected | Submit with blank body | Validation error displayed | PASS |
| 1.3.11 | Form preserves input on error | Submit invalid form | All previously entered values preserved in form fields | PASS |
| 1.3.12 | Cancel closes dialog | Click cancel button | Dialog closes, no task created | PASS |
| 1.3.13 | Select "todo" status | Choose "todo" from status dropdown, submit | Task created in todo column | PASS |
| 1.3.14 | Select "in_progress" status | Choose "in_progress", submit | Task created in In Progress column | PASS |
| 1.3.15 | Summary auto-generated | Leave summary blank, submit valid form | Task created with auto-generated summary | PASS |
| 1.3.16 | Consecutive spaces collapsed in title | Submit title "Auth   Flow   System" | Title stored as "Auth Flow System" | PASS |
| 1.3.17 | Escape key closes dialog | Press Escape while dialog is open | Dialog closes, no task created | PASS |

### 1.4 Drag-and-Drop

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.4.1 | Drag card to different column | Drag from backlog to todo | Task moves to todo column, status updated in DB | PASS |
| 1.4.2 | Drag card within same column | Drag card above another in same column | Card reordered, position updated in DB | PASS |
| 1.4.3 | Drop placeholder appears | Drag card over another column | Visual drop placeholder shown at target position | PASS |
| 1.4.4 | Drag to empty column | Drag card to column with no cards | Card placed in empty column, status updated | PASS |
| 1.4.5 | Board re-renders after move | Complete a drag-move | Entire board re-fetched and kanban re-initialized | PASS |
| 1.4.6 | Board re-renders after reorder | Complete a drag-reorder | Board re-fetched, new order preserved | PASS |
| 1.4.7 | Multiple rapid drags | Quickly drag 3 cards in succession | All moves processed correctly, no data loss | SKIP |
| 1.4.8 | Drag across all 8 columns | Move a card through each status | Card ends in correct final column | SKIP |
| 1.4.9 | Card link still works after re-render | Drag a card, then click its title | Navigates to task detail page | PASS |
| 1.4.10 | Error on move shows snackbar | Simulate server error on `/api/board/move` | Error snackbar appears, card returns to original column | SKIP |

### 1.5 Card Navigation

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.5.1 | Click card title | Click task title on card | Navigates to `/tasks/{id}` | PASS |
| 1.5.2 | Click spec title on card | Click spec label on card | Navigates to `/specs/{specId}` | PASS |

---

## 2. Web UI - Specs List

### 2.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 2.1.1 | Empty specs page | Navigate to `/specs` with no specs | "No specs yet" empty state message | SKIP |
| 2.1.2 | Specs grouped by type | Create business, functional, non-functional specs | Three groups rendered with correct labels | PASS |
| 2.1.3 | Group shows spec count | Create 3 functional specs | Functional group shows all 3 | PASS |
| 2.1.4 | Spec card shows title | Load specs page | Each card displays clickable spec title | PASS |
| 2.1.5 | Spec card shows type badge | Load specs page | Each card shows type chip (business/functional/non-functional) | PASS |
| 2.1.6 | Spec card shows task progress | Create spec with 5 tasks (3 done) | Progress bar shows 60% | PASS |
| 2.1.7 | Spec card links to detail | Click spec title | Navigates to `/specs/{id}` | PASS |
| 2.1.8 | Groups only rendered if specs exist | Create only functional specs | Only "Functional" group header rendered | PASS |

### 2.2 New Spec Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 2.2.1 | Dialog opens | Click "New Spec" button | Dialog with title, type, summary, body fields | PASS |
| 2.2.2 | Type dropdown options | Open dialog | Three options: business, functional, non-functional | PASS |
| 2.2.3 | Valid submission | Fill all fields, submit | Redirects to `/specs/{newId}` | PASS |
| 2.2.4 | Title min 2 chars enforced | Submit title "X" | Error: title must be at least 2 characters | PASS |
| 2.2.5 | Body min 20 chars enforced | Submit body "Short body" | Error: body must be at least 20 characters | PASS |
| 2.2.6 | Type is required | Submit without selecting type | Browser-level required validation blocks submission | PASS |
| 2.2.7 | Form preserved on validation error | Submit invalid, inspect form | All user-entered values retained | PASS |
| 2.2.8 | HTTP 422 on validation error | Submit invalid form | Response status 422, form partial re-rendered | PASS |
| 2.2.9 | Summary is optional | Submit without summary | Spec created successfully | PASS |
| 2.2.10 | Long title (250 chars) | Submit title of exactly 250 characters | Spec created, title fully preserved | PASS |
| 2.2.11 | Title with special characters | Submit title: `OAuth & GitHub "Login" <Flow>` | Title preserved with correct escaping in HTML | PASS |

---

## 3. Web UI - Spec Detail

### 3.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.1.1 | Spec header renders | Navigate to `/specs/SPEC-1` | Title, type badge, ID displayed | PASS |
| 3.1.2 | Back link to specs | Click back chip | Navigates to `/specs` | PASS |
| 3.1.3 | Markdown body rendered | Create spec with markdown body | Body rendered as HTML (headers, lists, code blocks) | PASS |
| 3.1.4 | Summary displayed | Spec has summary text | Summary shown below title | PASS |
| 3.1.5 | Progress bar shown | Spec has tasks | Progress bar with done/active counts | PASS |
| 3.1.6 | No progress bar if no tasks | Spec has zero tasks | Progress section hidden | SKIP |
| 3.1.7 | Linked specs section | Link two specs | "Related Specs" section shows linked spec with link | PASS |
| 3.1.8 | No linked specs section | Spec has no links | Related Specs section hidden | SKIP |
| 3.1.9 | Citations section | Cite KB chunk from spec | "References" section shows citation card | PASS |
| 3.1.10 | Citation card content | Cite KB-1:3 | Card shows source type icon, doc title, chunk position, text preview, "View in source" link | PASS |
| 3.1.11 | Citation "View in source" | Click link on citation | Navigates to `/kb/{docId}?chunk={position}` | PASS |
| 3.1.12 | No citations section | Spec with no citations | References section hidden | SKIP |
| 3.1.13 | Task grid renders | Spec has 5 tasks | Tasks shown in grid layout with status and title | PASS |
| 3.1.14 | Task card links to detail | Click task in grid | Navigates to `/tasks/{taskId}` | PASS |
| 3.1.15 | Empty task grid | Spec with no tasks | "No tasks yet" message shown | SKIP |
| 3.1.16 | 404 for nonexistent spec | Navigate to `/specs/SPEC-999` | Error page with 404 code | PASS |
| 3.1.17 | Markdown body renders tables | Spec body has GFM table | Table rendered with borders and alignment | PASS |
| 3.1.18 | Markdown body renders code blocks | Spec body has fenced code block | `<pre><code>` rendered with language class | PASS |
| 3.1.19 | Markdown body renders nested lists | Spec body has nested bullet/numbered lists | Proper indentation and nesting | PASS |
| 3.1.20 | Markdown body renders inline links | Body has `[link](url)` | Rendered as clickable `<a>` tag | PASS |
| 3.1.21 | Progress all tasks done | 5 tasks all in "done" | Progress bar at 100% | SKIP |
| 3.1.22 | Progress all tasks cancelled | 3 tasks all "cancelled" | Active count = 0, progress N/A or 0% | SKIP |
| 3.1.23 | Progress mixed wontfix and done | 2 done, 1 wontfix, 1 in_progress | Active = 3 (total - wontfix), done/active percent | SKIP |
| 3.1.24 | Escape key closes edit dialog | Press Escape in edit spec dialog | Dialog closes without saving | PASS |

### 3.2 Edit Spec Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.2.1 | Dialog pre-fills values | Click edit button | Title, type, summary, body populated with current values | PASS |
| 3.2.2 | Update title | Change title, submit | Title updated on page, slug unchanged | PASS |
| 3.2.3 | Update type | Change from functional to business | Type badge updated | PASS |
| 3.2.4 | Update body | Edit body markdown, submit | New body rendered | PASS |
| 3.2.5 | Validation on edit | Clear title, submit | 422 error with validation message | PASS |
| 3.2.6 | Cancel preserves original | Open edit, change title, cancel | Original title still shown | PASS |

### 3.3 Delete Spec

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.3.1 | Confirmation dialog | Click delete button | Confirmation dialog mentions cascading task deletion | PASS |
| 3.3.2 | Confirm delete | Click delete in confirmation | Redirected away, spec removed from list | PASS |
| 3.3.3 | Cancel delete | Click cancel in confirmation | Spec still exists | PASS |
| 3.3.4 | Cascade deletes tasks | Delete spec with tasks | All child tasks also trashed | PASS |
| 3.3.5 | Spec appears in trash | Delete spec | Spec visible in `/trash` | PASS |

### 3.4 New Task Dialog (from Spec Detail)

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.4.1 | Spec ID pre-filled | Open new task dialog from spec detail | spec_id hidden field set to current spec | PASS |
| 3.4.2 | Valid submission | Fill fields, submit | Task created under current spec, redirected to task detail | PASS |
| 3.4.3 | Validation error shows spec-detail variant | Submit invalid form | Error form rendered in spec-detail context (not board variant) | PASS |

---

## 4. Web UI - Task Detail

### 4.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.1.1 | Task header renders | Navigate to `/tasks/TASK-1` | Title, ID, parent spec ID displayed | PASS |
| 4.1.2 | Back to parent spec | Click parent spec link | Navigates to `/specs/{specId}` | PASS |
| 4.1.3 | Back to board (no parent) | Parent spec deleted | Back link goes to `/` | SKIP |
| 4.1.4 | Status dropdown shows current | Task in "todo" status | Status dropdown pre-selects "todo" | PASS |
| 4.1.5 | Markdown body rendered | Task has markdown body | Rendered as HTML | PASS |
| 4.1.6 | Summary displayed | Task has summary | Summary text shown | PASS |
| 4.1.7 | 404 for nonexistent task | Navigate to `/tasks/TASK-999` | Error page with 404 | PASS |
| 4.1.8 | Markdown body renders in task | Task body has headers, lists, code | Rendered as HTML correctly | PASS |
| 4.1.9 | created_by / updated_by shown | Task has creator configured | Attribution info displayed | PASS |

### 4.2 Status Change

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.2.1 | Auto-submit on change | Select different status from dropdown | Form auto-submits, page reloads with new status | PASS |
| 4.2.2 | Move backlog to todo | Change status to todo | Status updated, task in todo column on board | PASS |
| 4.2.3 | Move to in_progress | Change to in_progress | Status persisted | PASS |
| 4.2.4 | Move to blocked | Change to blocked | Status persisted | PASS |
| 4.2.5 | Move to pending_verification | Change status | Status persisted | PASS |
| 4.2.6 | Move to done | Change to done | Status persisted | PASS |
| 4.2.7 | Move to cancelled | Change to cancelled | Status persisted | PASS |
| 4.2.8 | Move to wontfix | Change to wontfix | Status persisted | PASS |
| 4.2.9 | All 8 statuses selectable | Open dropdown | All 8 status options present | PASS |

### 4.3 Acceptance Criteria

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.3.1 | Criteria list renders | Task has 3 criteria | All 3 shown with checkboxes | PASS |
| 4.3.2 | Check criterion | Click unchecked checkbox | Criterion marked as checked, page re-renders | PASS |
| 4.3.3 | Uncheck criterion | Click checked checkbox | Criterion unchecked, page re-renders | PASS |
| 4.3.4 | Add criterion | Enter text, submit | New criterion appended to list | PASS |
| 4.3.5 | Add criterion validation | Submit empty text | Form not submitted (HTML required attribute) | PASS |
| 4.3.6 | Add criterion min 2 chars | Submit "A" | Error via redirect | PASS |
| 4.3.7 | Remove criterion | Click remove button on criterion | Criterion removed, remaining renumbered | PASS |
| 4.3.8 | Empty criteria state | Task with no criteria | "No criteria yet." message | PASS |
| 4.3.9 | Criterion text preserved | Add criterion with markdown text | Text displayed as-is (not rendered as markdown) | PASS |
| 4.3.10 | Multiple criteria checked | Check 3 out of 5 | All 3 show checked state, progress updates | PASS |
| 4.3.11 | Criteria synced to markdown | Check criterion via UI | Task markdown file updated with `[x]` | PASS |

### 4.4 Dependencies

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.4.1 | Dependencies section renders | Task has 2 dependencies | Both shown with titles and status | PASS |
| 4.4.2 | Ready dependency icon | Blocker task is "done" | Green check_circle icon shown | PASS |
| 4.4.3 | Not-ready dependency icon | Blocker task is "in_progress" | Red block icon shown | PASS |
| 4.4.4 | Dependency links to task | Click dependency title | Navigates to blocker task detail | PASS |
| 4.4.5 | No dependencies section | Task has no deps | Dependencies section hidden | PASS |

### 4.5 Linked Tasks

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.5.1 | Linked tasks section renders | Task linked to 2 others | Both shown with title and status | PASS |
| 4.5.2 | Link navigates to task | Click linked task title | Navigates to linked task detail | PASS |
| 4.5.3 | No linked tasks section | Task has no links | Section hidden | PASS |

### 4.6 Citations

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.6.1 | Citations section renders | Task cites KB chunk | Citation card with type icon, title, chunk info | PASS |
| 4.6.2 | Citation preview text | Cite a long chunk | Text truncated to ~200 chars | PASS |
| 4.6.3 | View in source link | Click "View in source" | Navigates to KB reader at correct chunk | PASS |
| 4.6.4 | PDF citation shows page | Cite PDF chunk | "page N" displayed on citation card | SKIP |
| 4.6.5 | No citations section | Task has no citations | References section hidden | PASS |

### 4.7 Edit Task Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.7.1 | Pre-fills current values | Click edit | Title, summary, body populated | PASS |
| 4.7.2 | Update title | Change title, submit | Title updated on page | PASS |
| 4.7.3 | Update body with criteria | Edit body that has criteria section | Criteria re-parsed from new body | PASS |
| 4.7.4 | Validation on edit | Clear title, submit | 422 error | PASS |
| 4.7.5 | Status preserved on edit | Edit title only | Status unchanged after update | PASS |

### 4.8 Delete Task

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.8.1 | Confirmation dialog | Click delete | Confirmation dialog appears | PASS |
| 4.8.2 | Confirm delete | Click delete in dialog | Redirected, task removed | PASS |
| 4.8.3 | Task appears in trash | Delete task | Task visible in trash page | PASS |
| 4.8.4 | Citations cleaned up | Delete task with citations | Citations removed from DB | PASS |

---

## 5. Web UI - Knowledge Base List

### 5.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.1.1 | Empty KB page | No KB docs | "No documents" empty state | SKIP |
| 5.1.2 | KB docs table | Add 3 docs | Table shows ID, title, type, date for each | PASS |
| 5.1.3 | Type badge on doc | Add HTML doc | Type chip shows "HTML" | PASS |
| 5.1.4 | Doc links to detail | Click doc title | Navigates to `/kb/{id}` | PASS |

### 5.2 Type Filter

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.2.1 | Filter by Markdown | Click "Markdown" chip | Only .md docs shown | PASS |
| 5.2.2 | Filter by HTML | Click "HTML" chip | Only .html docs shown | PASS |
| 5.2.3 | Filter by PDF | Click "PDF" chip | Only .pdf docs shown | PASS |
| 5.2.4 | Filter by Text | Click "Text" chip | Only .txt docs shown | PASS |
| 5.2.5 | Clear filter (All) | Click "All" chip | All docs shown | PASS |
| 5.2.6 | Active chip highlighted | Click a filter chip | Selected chip visually distinct | PASS |
| 5.2.7 | Filter preserves search query | Search, then filter | Query param preserved in filter links | PASS |

### 5.3 KB Search

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.3.1 | Search returns results | Search for existing term | Search results section with matching chunks | PASS |
| 5.3.2 | Search result shows doc info | Search and get hits | Each result shows doc ID, chunk position, text preview | PASS |
| 5.3.3 | No results message | Search for nonexistent term | "No results" message displayed | PASS |
| 5.3.4 | Empty search shows doc list | Clear search | Full document list shown | PASS |
| 5.3.5 | Search combined with filter | Search + type filter | Results filtered to matching type | PASS |

### 5.4 Add Document Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.4.1 | Dialog opens | Click "Add Document" | Dialog with file, URL, title, note fields | PASS |
| 5.4.2 | Upload markdown file | Select .md file, submit | Doc added, redirect to `/kb` | PASS |
| 5.4.3 | Upload HTML file | Select .html file, submit | Doc added with clean sidecar created | PASS |
| 5.4.4 | Upload PDF file | Select .pdf file, submit | Doc added with page count | SKIP |
| 5.4.5 | Upload text file | Select .txt file, submit | Doc added | PASS |
| 5.4.6 | Add by URL | Enter URL, submit | Doc fetched, added | SKIP |
| 5.4.7 | Neither file nor URL | Submit empty form | Error: "Please provide a file or URL" | PASS |
| 5.4.8 | Title optional | Submit without title | Title derived from filename | PASS |
| 5.4.9 | Note optional | Submit without note | Doc created with empty note | PASS |
| 5.4.10 | Custom title | Provide title, submit | Custom title used instead of filename | PASS |
| 5.4.11 | File type validation | Select .exe file | Browser-level accept filter prevents selection (.md,.html,.htm,.pdf,.txt) | PASS |
| 5.4.12 | File name displayed | Select file | Selected filename shown in UI | PASS |
| 5.4.13 | Multipart encoding | Submit with file | Request uses `multipart/form-data` | PASS |
| 5.4.14 | Escape key closes dialog | Press Escape | Dialog closes without adding | PASS |

### 5.5 Delete KB Document

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.5.1 | Confirm delete | Click delete, confirm | Doc removed from list | PASS |
| 5.5.2 | Browser confirmation | Click delete | `confirm()` dialog shown | PASS |
| 5.5.3 | Cascade deletes chunks | Delete doc | All chunks and connections removed | PASS |
| 5.5.4 | Cascade deletes citations | Delete cited doc | Citations from specs/tasks cleaned | PASS |
| 5.5.5 | Clean sidecar removed | Delete HTML doc | Both .html and .clean.html removed | PASS |
| 5.5.6 | Doc appears in trash | Delete doc | KB entry in trash | PASS |

---

## 6. Web UI - KB Detail / Reader

### 6.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.1.1 | Header shows doc info | Navigate to `/kb/KB-1` | Title, type badge, chunk count, date | PASS |
| 6.1.2 | Back to KB link | Click back chip | Navigates to `/kb` | PASS |
| 6.1.3 | PDF shows page count | Open PDF doc detail | Page count displayed in header | SKIP |
| 6.1.4 | Note displayed | Doc has note | Note text shown | PASS |
| 6.1.5 | Loading state | Open doc detail | "Loading document..." shown before render | PASS |
| 6.1.6 | 404 for nonexistent doc | Navigate to `/kb/KB-999` | Error page | PASS |

### 6.2 Markdown Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.2.1 | Markdown rendered as HTML | Open .md doc | Headings, lists, code, links rendered | PASS |
| 6.2.2 | Code blocks syntax highlighted | Doc with code blocks | `<pre><code>` elements rendered | PASS |
| 6.2.3 | Active chunk highlighted | Open with `?chunk=3` | Chunk 3 text wrapped in `<mark>` element | PASS |
| 6.2.4 | Highlight scrolled into view | Open deep chunk | Page scrolls to highlighted region | PASS |
| 6.2.5 | Prefix match for highlight | Chunk text split across DOM nodes | Highlight found using first 80 chars | PASS |

### 6.3 Plain Text Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.3.1 | Text rendered in pre | Open .txt doc | Content in `<pre>` element | PASS |
| 6.3.2 | Character offset highlighting | Open with chunk | Chunk highlighted using char_start/char_end | PASS |
| 6.3.3 | HTML entities escaped | Text contains `<script>` literal | Rendered as text, not executed | PASS |

### 6.4 HTML Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.4.1 | Sandboxed iframe | Open .html doc | Content in iframe with `sandbox="allow-same-origin"` | PASS |
| 6.4.2 | No script execution | HTML has `<script>` tags | Scripts not executed (sanitized at ingest) | PASS |
| 6.4.3 | Iframe auto-resizes | Open long HTML doc | Iframe height matches content | PASS |
| 6.4.4 | Chunk highlighted in iframe | Open with `?chunk=N` | Highlight applied inside iframe DOM | PASS |
| 6.4.5 | CSS isolated | HTML has custom CSS | Does not affect parent page styles | PASS |

### 6.5 PDF Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.5.1 | PDF pages rendered | Open .pdf doc | All pages rendered as canvas elements | SKIP |
| 6.5.2 | Text layer overlay | Open .pdf doc | Invisible text layer for selection/highlighting | SKIP |
| 6.5.3 | Chunk highlighted | Open with `?chunk=N` | Text highlighted in text layer | SKIP |
| 6.5.4 | Page navigation on chunk change | Navigate to chunk on different page | Correct page scrolled into view | SKIP |
| 6.5.5 | PDF.js worker loaded | Open any PDF | Worker loads from `/assets/pdf.worker.min.mjs` | SKIP |
| 6.5.6 | Large PDF (100+ pages) | Open 100-page PDF | Renders and navigates smoothly | SKIP |

### 6.6 Chunk Navigation

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.6.1 | Position display | Open doc with 47 chunks | Shows "1 / 47" (or active chunk position) | PASS |
| 6.6.2 | Next chunk button | Click next | Moves to chunk N+1, highlight updates | PASS |
| 6.6.3 | Previous chunk button | Click previous at chunk 5 | Moves to chunk 4 | PASS |
| 6.6.4 | Previous disabled at first | View chunk 0 | Previous button disabled | PASS |
| 6.6.5 | Next disabled at last | View last chunk | Next button disabled | PASS |
| 6.6.6 | Keyboard navigation (right) | Press right arrow key | Moves to next chunk | PASS |
| 6.6.7 | Keyboard navigation (left) | Press left arrow key | Moves to previous chunk | PASS |
| 6.6.8 | Chunk sidebar toggle | Click toggle button | Sidebar with all chunk previews appears | PASS |
| 6.6.9 | Sidebar chunk click | Click chunk in sidebar | Jumps to clicked chunk, highlight updates | PASS |
| 6.6.10 | Sidebar chunk preview text | Open sidebar | Each chunk shows first ~60 chars | PASS |
| 6.6.11 | Active chunk in sidebar | Navigate to chunk 5 | Chunk 5 highlighted in sidebar list | PASS |
| 6.6.12 | Highlight toggle | Click highlight toggle button | Highlight removed/restored | PASS |

### 6.7 Connections

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.7.1 | Related chunks displayed | Doc has TF-IDF connections | "Related Chunks" section with linked chunks | PASS |
| 6.7.2 | Connection shows target doc | View connections | Target doc title and chunk position shown | PASS |
| 6.7.3 | Connection link navigates | Click related chunk | Navigates to `/kb/{docId}?chunk={pos}` | PASS |
| 6.7.4 | No connections | Chunk has no connections | Section hidden or empty state | PASS |

### 6.8 Citations

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.8.1 | Citing specs/tasks shown | Doc chunk is cited | "Referenced By" section with citing entities | PASS |
| 6.8.2 | Citation links to entity | Click citing spec/task | Navigates to spec/task detail | PASS |
| 6.8.3 | No citations | No entities cite this doc | Section hidden | PASS |

---

## 7. Web UI - Search

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 7.1 | Empty search page | Navigate to `/search` | Search input with prompt text, no results | PASS |
| 7.2 | Search returns grouped results | Search for term matching all kinds | Results grouped: Specs, Tasks, KB sections | PASS |
| 7.3 | Spec results | Search matches specs | Shows spec ID, title, summary, score, match type | PASS |
| 7.4 | Task results | Search matches tasks | Shows task ID, title, summary, score, match type | PASS |
| 7.5 | KB results | Search matches KB chunks | Shows doc ID, title, chunk position, text preview | PASS |
| 7.6 | Result count per section | Multiple hits per kind | Section headers show count | PASS |
| 7.7 | Match type badge (BM25) | Primary FTS5 match | Badge shows "bm25" | PASS |
| 7.8 | Match type badge (trigram) | Fuzzy/substring match | Badge shows "trigram" | PASS |
| 7.9 | Score displayed | Search results | Relevance score shown | PASS |
| 7.10 | Result links navigate | Click spec result | Navigates to `/specs/{id}` | PASS |
| 7.11 | Task result links | Click task result | Navigates to `/tasks/{id}` | PASS |
| 7.12 | KB result links | Click KB result | Navigates to `/kb/{id}` | PASS |
| 7.13 | No results message | Search for gibberish | "No results found for 'xyz'" message | PASS |
| 7.14 | Quoted phrase search | Search `"exact phrase"` | FTS phrase matching applied | PASS |
| 7.15 | Prefix search | Search `auth*` | FTS prefix matching | PASS |
| 7.16 | GET form submission | Submit search | URL shows `/search?q=term` | PASS |
| 7.17 | Query preserved in input | Search, then view results | Search input retains query text | PASS |
| 7.18 | Empty query submission | Submit empty search | No results, guidance text shown | PASS |

---

## 8. Web UI - Status

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 8.1 | Spec counts | Create 2 functional, 1 business | Shows Total: 3, Functional: 2, Business: 1 | PASS |
| 8.2 | Task counts by status | Tasks in various statuses | Breakdown table shows each status count | PASS |
| 8.3 | KB counts | Add 3 docs (1 md, 1 html, 1 pdf) | Total: 3, by-type counts, chunk count | PASS |
| 8.4 | Tidy health OK | Recently ran tidy | "OK" in green | PASS |
| 8.5 | Tidy health stale | Tidy older than 7 days | "Stale" in red | SKIP |
| 8.6 | Lint issues table | Lint reports issues | Table with severity, category, message | PASS |
| 8.7 | No lint issues | Clean workspace | "No issues found" with checkmark | SKIP |
| 8.8 | Trash summary | Items in trash | Count and link to `/trash` | PASS |
| 8.9 | Empty trash summary | No trash | "Empty." with checkmark | PASS |
| 8.10 | Rejected files link | Rejected files exist | Count and link to `/rejected` | PASS |
| 8.11 | No rejected files | None rejected | "None." with checkmark | SKIP |
| 8.12 | Empty workspace status | Fresh init | All counts zero, health OK | SKIP |

---

## 9. Web UI - Trash

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 9.1 | Empty trash page | No deleted items | "Trash is empty" with check icon | PASS |
| 9.2 | Trash table | Delete spec, task, KB doc | Table shows all 3 with kind, original ID, path, date, deleted_by | SKIP |
| 9.3 | Kind badges | Items of different kinds | Chips show spec/task/kb with correct icons | SKIP |
| 9.4 | Restore spec | Click restore on spec | Spec recreated, disappears from trash | PASS |
| 9.5 | Restore task | Click restore on task | Task recreated under parent spec | PASS |
| 9.6 | Restore KB | Click restore on KB | Doc and chunks recreated | PASS |
| 9.7 | Restore with ID conflict | Delete spec, create new with same ID slot, restore | New ID allocated, warning shown | SKIP |
| 9.8 | Restore task with missing spec | Delete spec and task, don't restore spec, try restore task | Error: parent spec no longer exists | SKIP |
| 9.9 | Purge all button visible | Items in trash | "Purge All" button rendered | PASS |
| 9.10 | Purge all hidden | No items | "Purge All" button not rendered | PASS |
| 9.11 | Purge confirmation dialog | Click "Purge All" | Confirmation dialog appears | PASS |
| 9.12 | Confirm purge all | Click confirm in dialog | All trash permanently deleted | PASS |
| 9.13 | Cancel purge | Click cancel in dialog | Trash unchanged | PASS |
| 9.14 | Deleted by shows watcher | Delete file externally | "watcher" in deleted_by column | PASS |
| 9.15 | Deleted by shows cli | Delete via CLI | "cli" in deleted_by column | PASS |

---

## 10. Web UI - Rejected Files

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 10.1 | Empty rejected page | No rejected files | "No rejected files" with check icon | SKIP |
| 10.2 | Rejected files table | Hand-create files in specd/ | Table shows path, reason, detected_at | PASS |
| 10.3 | Reason displayed | Non-canonical file created | Reason explains why file was rejected | PASS |
| 10.4 | Detection timestamp | File rejected | ISO timestamp shown | PASS |
| 10.5 | Nav highlights status | Navigate to `/rejected` | "Status" nav item active | PASS |

---

## 11. Web UI - Navigation & Layout

### 11.1 Desktop Sidebar

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 11.1.1 | Sidebar visible on large screens | Open on desktop width (>=993px) | Left sidebar with nav links visible | PASS |
| 11.1.2 | Active nav link highlighted | Navigate to `/specs` | "Specs" link has active class | PASS |
| 11.1.3 | All nav links present | Load any page | Board, Specs, KB, Search, Status, Trash links | PASS |
| 11.1.4 | Nav icons | View sidebar | Each link has correct Material icon | PASS |
| 11.1.5 | Sidebar collapse | Click toggle button | Sidebar collapses, state persisted to localStorage | PASS |
| 11.1.6 | Sidebar expand | Click toggle on collapsed sidebar | Sidebar expands | PASS |
| 11.1.7 | Collapse persisted | Collapse sidebar, reload page | Sidebar still collapsed | PASS |
| 11.1.8 | Aria-expanded attribute | Toggle sidebar | `aria-expanded` toggles between true/false | PASS |

### 11.2 Mobile Navigation

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 11.2.1 | Top bar on small screens | Open on mobile width (<601px) | Top bar with menu button, logo, dark mode toggle | SKIP |
| 11.2.2 | Menu button opens dialog | Click hamburger menu | Mobile menu dialog opens | SKIP |
| 11.2.3 | Mobile menu links | Open mobile menu | All nav links present | SKIP |
| 11.2.4 | Mobile link navigates | Click link in mobile menu | Navigates to page, menu closes | SKIP |
| 11.2.5 | Logo links to board | Click specd logo | Navigates to `/` | PASS |

### 11.3 Footer

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 11.3.1 | Footer renders | Scroll to bottom | Footer with about, social, learn, legal sections | PASS |
| 11.3.2 | GitHub link | Click GitHub link | Opens repository | PASS |
| 11.3.3 | Legal links | Click Privacy/Terms | Navigate to correct pages | PASS |

---

## 12. Web UI - Theme & Dark Mode

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 12.1 | Default theme | First visit | Light or dark based on system preference | PASS |
| 12.2 | Toggle to dark mode | Click dark mode button | UI switches to dark theme | PASS |
| 12.3 | Toggle to light mode | Click again | UI switches to light theme | PASS |
| 12.4 | Preference persisted | Toggle to dark, reload | Dark mode preserved via localStorage | PASS |
| 12.5 | Brand color applied | Load any page | Material Design tokens use `#1c4bea` | PASS |
| 12.6 | Dark mode icon correct | In light mode | Shows `dark_mode` icon | PASS |
| 12.7 | Light mode icon correct | In dark mode | Shows `light_mode` icon | PASS |
| 12.8 | Logo switches in dark mode | Toggle mode | Logo SVG changes between light/dark variants | PASS |

---

## 13. Web UI - Error Pages

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 13.1 | 404 page | Navigate to `/nonexistent` | Error page: code 404, "Not Found" message | FAIL |
| 13.2 | 404 for bad spec ID | Navigate to `/specs/NOTASPEC` | 404 error page | PASS |
| 13.3 | 404 for bad task ID | Navigate to `/tasks/NOTATASK` | 404 error page | PASS |
| 13.4 | 404 for bad KB ID | Navigate to `/kb/NOTAKB` | 404 error page | PASS |
| 13.5 | Home button on error | View error page | Button navigates to `/` | PASS |
| 13.6 | Error snackbar | Redirect with `?error=msg` | Snackbar appears with error message | PASS |
| 13.7 | Snackbar auto-dismiss | Wait after snackbar appears | Snackbar disappears after ~4 seconds | PASS |
| 13.8 | Error on htmx request | htmx request returns error | Error displayed appropriately | PASS |
| 13.9 | Browser back button | Navigate Board -> Specs -> Spec Detail, press back | Returns to Specs list (htmx history) | PASS |
| 13.10 | Browser forward button | Press back then forward | Returns to Spec Detail | PASS |
| 13.11 | Direct URL access | Paste `/specs/SPEC-1` into address bar | Full page loads (no htmx, base layout rendered) | PASS |

---

## 14. API Endpoints - KB

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 14.1 | GET `/api/kb/{id}` - valid | Request existing doc | JSON with id, title, source_type, page_count, note | PASS |
| 14.2 | GET `/api/kb/{id}` - not found | Request KB-999 | 404 JSON error | PASS |
| 14.3 | GET `/api/kb/{id}/chunks` | Request chunks for doc | JSON array of all chunks with position, text, char_start, char_end, page | PASS |
| 14.4 | GET `/api/kb/{id}/chunk/{pos}` - valid | Request chunk at position 3 | Single chunk JSON | PASS |
| 14.5 | GET `/api/kb/{id}/chunk/{pos}` - not found | Request position 999 | 404 error | PASS |
| 14.6 | GET `/api/kb/{id}/chunk/{pos}` - invalid pos | Request position "abc" | 400 error | PASS |
| 14.7 | GET `/api/kb/{id}/raw` - markdown | Request .md raw | `Content-Type: text/plain; charset=utf-8`, raw markdown bytes | PASS |
| 14.8 | GET `/api/kb/{id}/raw` - text | Request .txt raw | `Content-Type: text/plain; charset=utf-8` | PASS |
| 14.9 | GET `/api/kb/{id}/raw` - html | Request .html raw | `Content-Type: text/html; charset=utf-8`, clean sidecar served | PASS |
| 14.10 | GET `/api/kb/{id}/raw` - pdf | Request .pdf raw | `Content-Type: application/pdf`, raw PDF bytes | SKIP |
| 14.11 | Path traversal protection | Manipulate doc_id to escape kb/ | 403 Forbidden | PASS |
| 14.12 | Files constrained to specd/kb/ | Request with valid ID but symlinked path | Path verified within workspace | PASS |
| 14.13 | Raw serves clean HTML | Request HTML raw | Serves `.clean.html` sidecar, never original | PASS |

---

## 15. API Endpoints - Board Drag-and-Drop

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 15.1 | POST `/api/board/move` - valid | `{"task_id":"TASK-1","status":"todo"}` | 200, re-rendered board HTML | PASS |
| 15.2 | POST `/api/board/move` - invalid status | `{"task_id":"TASK-1","status":"invalid"}` | Error response | PASS |
| 15.3 | POST `/api/board/move` - missing task | `{"task_id":"TASK-999","status":"todo"}` | Error response | PASS |
| 15.4 | POST `/api/board/reorder` - valid | `{"task_id":"TASK-1","position":0}` | 200, re-rendered board | PASS |
| 15.5 | POST `/api/board/reorder` - invalid pos | `{"task_id":"TASK-1","position":-1}` | Error or clamped to 0 | PASS |
| 15.6 | JSON Content-Type required | POST with form data | Correct handling or error | PASS |

---

## 16. CLI - Init

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 16.1 | Init empty directory | `specd init /tmp/test` | Creates specd/, .specd/, cache.db, index.md, log.md | PASS |
| 16.2 | Init current directory | `cd /tmp/test && specd init` | Workspace created at CWD | PASS |
| 16.3 | Init already initialized | `specd init` twice | Error: "workspace already initialized" | PASS |
| 16.4 | Init with --force | `specd init --force` on existing | Reinitializes successfully | PASS |
| 16.5 | .gitignore updated | `specd init` in git repo | `.specd/` line appended | PASS |
| 16.6 | .gitignore not duplicated | `specd init --force` | `.specd/` not added twice | PASS |
| 16.7 | User name from git | Git config has user.name | meta.user_name seeded | PASS |
| 16.8 | No git config | No git user.name | meta.user_name empty | SKIP |
| 16.9 | JSON output | `specd init --json` | JSON with root and message | PASS |

---

## 17. CLI - Config

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 17.1 | Set user.name | `specd config user.name "John"` | Stored in meta table | PASS |
| 17.2 | Get user.name | `specd config user.name` | Prints current value | PASS |
| 17.3 | JSON output | `specd config user.name --json` | JSON `{"user.name": "John"}` | PASS |
| 17.4 | Unknown key | `specd config foo.bar` | Error or unsupported message | PASS |
| 17.5 | user.name populates created_by | Set user.name "Alice", create spec | Spec JSON shows `created_by: "Alice"` | PASS |
| 17.6 | user.name populates updated_by | Set user.name "Bob", update a spec | Spec JSON shows `updated_by: "Bob"` | PASS |
| 17.7 | Tidy reminder in CLI response | Don't run tidy for 7+ days, run `specd new-spec ...` | Response includes non-null `tidy_reminder` field | SKIP |
| 17.8 | Tidy reminder absent when recent | Run tidy, then create spec | Response has `tidy_reminder: null` | PASS |

---

## 18. CLI - Spec Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 18.1 | new-spec happy path | `specd new-spec --title "Auth" --type functional --summary "Auth flow" --body "..."` | SPEC-1 created, JSON with id, path, candidates | PASS |
| 18.2 | new-spec creates directory | Run new-spec | `specd/specs/SPEC-1-auth/spec.md` exists | PASS |
| 18.3 | new-spec updates index.md | Run new-spec | Spec added to index.md | PASS |
| 18.4 | new-spec updates log.md | Run new-spec | Entry appended to log.md | PASS |
| 18.5 | new-spec with --link | `--link SPEC-1 --link SPEC-2` | Links created bidirectionally | PASS |
| 18.6 | new-spec with --cite | `--cite KB-1:3 --cite KB-2:7` | Citations created | PASS |
| 18.7 | new-spec --dry-run | Add `--dry-run` flag | No files or DB changes, preview output | PASS |
| 18.8 | new-spec candidates returned | Create spec similar to existing | Candidates block in response | PASS |
| 18.9 | read spec | `specd read SPEC-1` | JSON with all spec fields | PASS |
| 18.10 | read spec --with-tasks | Add flag | Includes child task list | PASS |
| 18.11 | read spec --with-links | Add flag | Includes linked specs | PASS |
| 18.12 | read spec --with-progress | Add flag | Includes done/active/percent | PASS |
| 18.13 | read spec --with-citations | Add flag | Includes cited chunk text | PASS |
| 18.14 | read nonexistent spec | `specd read SPEC-999` | Error: "spec SPEC-999 not found" | PASS |
| 18.15 | list specs | `specd list specs` | All specs listed | PASS |
| 18.16 | list specs --type | `--type functional` | Only functional specs | PASS |
| 18.17 | list specs --linked-to | `--linked-to SPEC-1` | Specs linked to SPEC-1 | PASS |
| 18.18 | list specs --empty | `--empty` | Only specs with 0 tasks | PASS |
| 18.19 | list specs --limit | `--limit 5` | Max 5 specs returned | PASS |
| 18.20 | update spec | `specd update SPEC-1 --title "New Title"` | Title updated, other fields unchanged | SKIP |
| 18.21 | rename spec | `specd rename SPEC-1 --title "New Name"` | Directory renamed, slug updated | SKIP |
| 18.22 | delete spec | `specd delete SPEC-1` | Soft-deleted to trash | SKIP |
| 18.23 | reorder spec --before | `specd reorder spec SPEC-2 --before SPEC-1` | SPEC-2 positioned before SPEC-1 | SKIP |
| 18.24 | reorder spec --after | `specd reorder spec SPEC-1 --after SPEC-3` | Position updated | SKIP |
| 18.25 | reorder spec --to | `specd reorder spec SPEC-1 --to 0` | Moved to first position | SKIP |
| 18.26 | new-spec --dry-run no side effects | `specd new-spec --dry-run --title "X" --type functional --summary "Y" --body "..."` | No directory created, no DB row, preview returned | PASS |
| 18.27 | new-spec with --link at creation | `specd new-spec ... --link SPEC-1` | Spec created AND link established in one command | PASS |
| 18.28 | new-spec with --cite at creation | `specd new-spec ... --cite KB-1:0` | Spec created AND citation established in one command | PASS |
| 18.29 | read spec combined flags | `specd read SPEC-1 --with-tasks --with-links --with-progress --with-citations` | All enrichments returned in single response | PASS |
| 18.30 | rename spec verifies path change | Rename SPEC-1 to "New Name" | Directory changes from `SPEC-1-old-slug/` to `SPEC-1-new-name/`, task paths inside updated | SKIP |
| 18.31 | new-spec index.md content | Create 3 specs, read index.md | All 3 specs listed in index with IDs and titles | PASS |
| 18.32 | new-spec log.md content | Create spec, read log.md | Entry with timestamp, ID, title appended | PASS |
| 18.33 | delete spec removes from index.md | Delete SPEC-2 | SPEC-2 removed from index.md | SKIP |
| 18.34 | new-spec slug generation | `--title "OAuth & GitHub 'Login' <Flow>"` | Slug: `oauth-github-login-flow` (special chars to hyphens) | PASS |
| 18.35 | slug truncation at 60 chars | `--title "Very Long Title That Exceeds Sixty Characters And Should Be Truncated Somewhere"` | Slug truncated to max 60 chars, no trailing hyphen | PASS |
| 18.36 | candidates command | `specd candidates SPEC-1 --limit 10` | Returns ranked candidate specs and KB chunks, excludes self and already-linked | PASS |
| 18.37 | candidates excludes linked | Link SPEC-1 to SPEC-2, run candidates for SPEC-1 | SPEC-2 not in results | PASS |
| 18.38 | candidates for task | `specd candidates TASK-1 --limit 10` | Returns candidate tasks and KB chunks | PASS |

---

## 19. CLI - Task Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 19.1 | new-task happy path | `specd new-task --spec-id SPEC-1 --title "Design" --summary "..." --body "..."` | TASK-1 created | PASS |
| 19.2 | new-task default status | Omit --status | Status defaults to "backlog" | PASS |
| 19.3 | new-task with --status | `--status todo` | Task created in todo | PASS |
| 19.4 | new-task missing spec | `--spec-id SPEC-999` | Error: parent spec not found | PASS |
| 19.5 | new-task with --depends-on | `--depends-on TASK-1` | Dependency created | PASS |
| 19.6 | new-task with criteria in body | Body contains `## Acceptance criteria\n- [ ] Item 1` | Criteria parsed and stored | PASS |
| 19.7 | read task | `specd read TASK-1` | JSON with all task fields | PASS |
| 19.8 | read --with-deps | Add flag | Includes blocker list with ready status | PASS |
| 19.9 | read --with-criteria | Add flag | Includes criteria with checked state | PASS |
| 19.10 | list tasks | `specd list tasks` | All tasks listed | PASS |
| 19.11 | list tasks --status | `--status in_progress` | Filtered by status | PASS |
| 19.12 | list tasks --spec-id | `--spec-id SPEC-1` | Only tasks under SPEC-1 | PASS |
| 19.13 | list tasks --created-by | `--created-by "John"` | Filtered by creator | PASS |
| 19.14 | move task | `specd move TASK-1 --status done` | Status changed, frontmatter updated | PASS |
| 19.15 | move task same status | `specd move TASK-1 --status backlog` (already backlog) | No-op, no error | PASS |
| 19.16 | rename task | `specd rename TASK-1 --title "New"` | File renamed, slug updated | SKIP |
| 19.17 | delete task | `specd delete TASK-1` | Soft-deleted to trash | SKIP |
| 19.18 | reorder task within status | `specd reorder task TASK-2 --before TASK-1` | Position updated within same status | SKIP |
| 19.19 | new-task --dry-run | `specd new-task --dry-run --spec-id SPEC-1 --title "X" --summary "Y" --body "..."` | No file created, no DB row, preview returned | PASS |
| 19.20 | new-task with --link at creation | `specd new-task ... --link TASK-1` | Task created AND link established in one command | PASS |
| 19.21 | new-task with --cite at creation | `specd new-task ... --cite KB-1:0` | Task created AND citation established in one command | PASS |
| 19.22 | read task --with-links | `specd read TASK-1 --with-links` | Includes linked tasks with title and status | PASS |
| 19.23 | read task --with-citations | `specd read TASK-1 --with-citations` | Includes cited KB chunks with text preview | PASS |
| 19.24 | read task combined flags | `specd read TASK-1 --with-links --with-deps --with-criteria --with-citations` | All enrichments returned in single response | PASS |
| 19.25 | list tasks --linked-to | `--linked-to TASK-1` | Only tasks linked to TASK-1 | PASS |
| 19.26 | list tasks --depends-on | `--depends-on TASK-1` | Only tasks that depend on TASK-1 | PASS |
| 19.27 | list tasks --limit | `--limit 3` | Max 3 tasks returned | PASS |
| 19.28 | rename task verifies path change | Rename TASK-1 to "New Task" | File changes from `TASK-1-old-slug.md` to `TASK-1-new-task.md` | SKIP |
| 19.29 | created_by populated | Config user.name "Alice", create task | Task `created_by` = "Alice" in JSON output | PASS |
| 19.30 | updated_by populated | Config user.name "Bob", update task | Task `updated_by` = "Bob" | PASS |

---

## 20. CLI - Criteria Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 20.1 | criteria list | `specd criteria list TASK-1` | Lists all criteria with position and checked state | PASS |
| 20.2 | criteria add | `specd criteria add TASK-1 "Write tests"` | Criterion appended, markdown updated | PASS |
| 20.3 | criteria check | `specd criteria check TASK-1 1` | `[ ]` changed to `[x]` in markdown and DB | PASS |
| 20.4 | criteria uncheck | `specd criteria uncheck TASK-1 1` | `[x]` changed to `[ ]` | PASS |
| 20.5 | criteria remove | `specd criteria remove TASK-1 2` | Criterion deleted, positions renumbered | PASS |
| 20.6 | check nonexistent position | `specd criteria check TASK-1 99` | Error: criterion not found | PASS |
| 20.7 | idempotent check | Check already-checked criterion | No error, state unchanged | PASS |

---

## 21. CLI - Link Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 21.1 | link specs | `specd link SPEC-1 SPEC-2` | Bidirectional link created | PASS |
| 21.2 | link tasks | `specd link TASK-1 TASK-2` | Bidirectional link created | PASS |
| 21.3 | link multiple targets | `specd link SPEC-1 SPEC-2 SPEC-3` | Two links created | PASS |
| 21.4 | cross-kind link rejected | `specd link SPEC-1 TASK-1` | Error: both must be same kind | PASS |
| 21.5 | link to nonexistent | `specd link SPEC-1 SPEC-999` | Error: target not found | PASS |
| 21.6 | link idempotent | Link same pair twice | No error on second call | PASS |
| 21.7 | Self-link rejected | `specd link SPEC-1 SPEC-1` | Error: cannot link to self | FAIL |
| 21.8 | unlink | `specd unlink SPEC-1 SPEC-2` | Both directions removed, frontmatter synced | PASS |
| 21.9 | frontmatter updated | After linking | `linked_specs` field in markdown updated | PASS |

---

## 22. CLI - Depend Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 22.1 | depend | `specd depend TASK-2 --on TASK-1` | Dependency created (TASK-1 blocks TASK-2) | PASS |
| 22.2 | depend multiple | `specd depend TASK-3 --on TASK-1 --on TASK-2` | Both blockers added | PASS |
| 22.3 | self-dependency rejected | `specd depend TASK-1 --on TASK-1` | Error: dependency cycle | PASS |
| 22.4 | direct cycle rejected | TASK-1 depends on TASK-2, then `depend TASK-2 --on TASK-1` | Error: dependency cycle detected with path | PASS |
| 22.5 | indirect cycle rejected | A->B->C exists, add C->A | Error: cycle A->B->C->A | PASS |
| 22.6 | undepend | `specd undepend TASK-2 --on TASK-1` | Dependency removed | PASS |
| 22.7 | frontmatter synced | After depend | `depends_on` field updated in markdown | PASS |
| 22.8 | depend nonexistent | `--on TASK-999` | Error: task not found | PASS |

---

## 23. CLI - Cite Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 23.1 | cite spec | `specd cite SPEC-1 KB-1:3` | Citation created | PASS |
| 23.2 | cite task | `specd cite TASK-1 KB-2:5` | Citation created | PASS |
| 23.3 | cite multiple chunks | `specd cite SPEC-1 KB-1:3 KB-1:5 KB-2:1` | All citations created | PASS |
| 23.4 | cite invalid format | `specd cite SPEC-1 KB1-3` | Error: invalid citation format | PASS |
| 23.5 | cite nonexistent KB | `specd cite SPEC-1 KB-999:0` | Error: KB doc not found | PASS |
| 23.6 | cite nonexistent chunk | `specd cite SPEC-1 KB-1:999` | Error: chunk not found | PASS |
| 23.7 | uncite | `specd uncite SPEC-1 KB-1:3` | Citation removed | PASS |
| 23.8 | frontmatter synced | After citing | `cites` field updated with grouped KB refs | PASS |
| 23.9 | idempotent cite | Cite same chunk twice | No error, single entry | PASS |

---

## 24. CLI - KB Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 24.1 | kb add markdown file | `specd kb add /path/to/doc.md` | KB-1 created, chunked, indexed | PASS |
| 24.2 | kb add text file | `specd kb add /path/to/notes.txt` | KB-1 with txt source_type | PASS |
| 24.3 | kb add HTML file | `specd kb add /path/to/page.html` | Clean sidecar created, chunked | PASS |
| 24.4 | kb add PDF file | `specd kb add /path/to/paper.pdf` | Page count recorded, page-aware chunks | SKIP |
| 24.5 | kb add with --title | `--title "Custom Title"` | Custom title overrides filename | PASS |
| 24.6 | kb add with --note | `--note "Reference for auth"` | Note stored | PASS |
| 24.7 | kb add nonexistent file | `specd kb add /tmp/nope.md` | Error: file not found | PASS |
| 24.8 | kb add unsupported type | `specd kb add /path/to/file.docx` | Error: unsupported file type | PASS |
| 24.9 | kb list | `specd kb list` | All docs listed with ID, type, title | PASS |
| 24.10 | kb list --source-type | `--source-type pdf` | Only PDFs listed | PASS |
| 24.11 | kb read | `specd kb read KB-1` | All chunks with positions and text | PASS |
| 24.12 | kb read --chunk | `--chunk 3` | Single chunk at position 3 | PASS |
| 24.13 | kb search | `specd kb search "authentication"` | Matching chunks with scores | PASS |
| 24.14 | kb search --limit | `--limit 5` | Max 5 results | PASS |
| 24.15 | kb connections | `specd kb connections KB-1` | TF-IDF connected chunks | PASS |
| 24.16 | kb connections --chunk | `--chunk 3` | Connections for specific chunk | PASS |
| 24.17 | kb rebuild-connections | Run command | All connections recomputed | PASS |
| 24.18 | kb rebuild-connections --threshold | `--threshold 0.5` | Higher threshold = fewer connections | PASS |
| 24.19 | kb rebuild-connections --top-k | `--top-k 5` | Max 5 connections per chunk | PASS |
| 24.20 | kb remove | `specd kb remove KB-1` | Soft-deleted to trash | SKIP |
| 24.21 | TF-IDF connections created on add | Add 2 related docs | Connections populated in chunk_connections | PASS |
| 24.22 | Chunking respects paragraph boundaries | Add doc with clear paragraphs | Chunks align to paragraph breaks | PASS |
| 24.23 | Chunk size constraints | Add doc with 2000+ char paragraph | Paragraph split at sentence boundaries | PASS |
| 24.24 | Max 10000 chunks | Add enormous document | Error or capped at 10000 | SKIP |
| 24.25 | kb add from URL | `specd kb add https://example.com/doc.html --title "Web Page"` | File fetched, saved locally, chunked, indexed | SKIP |
| 24.26 | kb add from URL - invalid URL | `specd kb add https://nonexistent.invalid/x.md` | Clean error, no partial state | SKIP |
| 24.27 | kb add from URL - type detection | Add URL returning HTML Content-Type | Source type detected as `html`, clean sidecar created | SKIP |

---

## 25. CLI - Search

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 25.1 | search all kinds | `specd search "auth" --kind all` | Results grouped by spec, task, kb | PASS |
| 25.2 | search specs only | `--kind spec` | Only spec results | PASS |
| 25.3 | search tasks only | `--kind task` | Only task results | PASS |
| 25.4 | search kb only | `--kind kb` | Only KB chunk results | PASS |
| 25.5 | BM25 primary search | Search for stemmed term | BM25 matches with match_type "bm25" | PASS |
| 25.6 | Trigram fallback | Search for substring | Trigram results tagged as "trigram" | PASS |
| 25.7 | Hybrid merge | BM25 returns <3 results | Trigram results appended below BM25 | PASS |
| 25.8 | Limit enforcement | `--limit 3` | Max 3 results per kind | PASS |
| 25.9 | FTS operators | `specd search "auth AND jwt"` | AND operator applied | PASS |
| 25.10 | Phrase search | `specd search '"exact match"'` | Phrase matching | PASS |
| 25.11 | Empty query | `specd search ""` | No results, no error | PASS |
| 25.12 | JSON output | `--json` | Structured JSON response | PASS |
| 25.13 | Score ordering | Search with multiple matches | Results ordered by relevance score descending | PASS |

---

## 26. CLI - Next

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 26.1 | Next returns todo tasks | Create tasks in various statuses | Only "todo" tasks returned | PASS |
| 26.2 | Ready before not-ready | Task A (no deps) and Task B (blocked) | Task A first | PASS |
| 26.3 | Partially done before fresh | Task A (1/3 criteria checked) vs Task B (0/3) | Task A first | PASS |
| 26.4 | Higher progress first | Task A (80%) vs Task B (50%) | Task A first | PASS |
| 26.5 | Position tiebreaker | Same progress, same readiness | Lower position first | PASS |
| 26.6 | --limit flag | `--limit 3` | Max 3 tasks | PASS |
| 26.7 | --spec-id filter | `--spec-id SPEC-1` | Only SPEC-1's tasks | PASS |
| 26.8 | Blocked-by list | Task with unresolved deps | `blocked_by` includes blocker IDs | PASS |
| 26.9 | Ready when deps done | Blocker in "done" status | Dependent task marked ready | PASS |
| 26.10 | Ready when deps cancelled | Blocker in "cancelled" | Dependent task marked ready | PASS |
| 26.11 | Ready when deps wontfix | Blocker in "wontfix" | Dependent task marked ready | PASS |
| 26.12 | Dependency cycle error | Circular dependency exists | Error with cycle path | SKIP |
| 26.13 | No todo tasks | All tasks in other statuses | Empty list, no error | PASS |
| 26.14 | JSON output | `--json` | Structured response with ready, partially_done, criteria_progress, blocked_by | PASS |

---

## 27. CLI - Maintenance

### 27.1 Lint

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.1.1 | Clean workspace | Lint with no issues | "0 errors, 0 warnings" | SKIP |
| 27.1.2 | Dangling spec link | Link to deleted spec | Error: dangling spec link | PASS |
| 27.1.3 | Dangling task link | Link to deleted task | Error: dangling task link | PASS |
| 27.1.4 | Dangling dependency | Depend on deleted task | Error: dangling dependency | PASS |
| 27.1.5 | Dangling citation | Cite deleted KB chunk | Error: dangling citation | PASS |
| 27.1.6 | Orphan spec | Spec with no links and no tasks | Warning: orphan spec | PASS |
| 27.1.7 | Orphan task | Task whose parent spec was removed | Error: orphan task | SKIP |
| 27.1.8 | System field drift | Manually edit linked_specs in file | Warning: field drift | SKIP |
| 27.1.9 | Stale tidy | last_tidy_at > 7 days | Warning: stale tidy | SKIP |
| 27.1.10 | Missing summary | Spec with empty summary | Warning: missing summary | SKIP |
| 27.1.11 | Trivial summary | Single-word summary | Warning: trivial summary | SKIP |
| 27.1.12 | Dependency cycle | Circular deps | Error with cycle path | SKIP |
| 27.1.13 | Rejected files | Files in rejected_files table | Warning per file | PASS |
| 27.1.14 | KB integrity - missing file | Delete KB source file from disk | Error: KB file missing | SKIP |
| 27.1.15 | KB integrity - hash mismatch | Modify KB file externally | Warning: hash mismatch | SKIP |
| 27.1.16 | Citation chunk missing | Cite chunk, then re-chunk (different positions) | Error: citation points to nonexistent chunk | SKIP |
| 27.1.17 | JSON output | `specd lint --json` | Structured JSON with issues array | PASS |

### 27.2 Tidy

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.2.1 | Tidy runs lint | `specd tidy` | Lint results output | PASS |
| 27.2.2 | Tidy updates timestamp | `specd tidy` | last_tidy_at updated to now | PASS |
| 27.2.3 | Tidy reminder cleared | After running tidy | tidy_reminder returns null | PASS |

### 27.3 Rebuild

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.3.1 | Rebuild empty workspace | `specd rebuild` | Fresh DB, 0 specs/tasks/kb | PASS |
| 27.3.2 | Rebuild with data | Create specs/tasks/kb, then rebuild | All data reconstructed from markdown | PASS |
| 27.3.3 | Counter preservation | Rebuild after SPEC-5 created | next_spec_id = 6 | PASS |
| 27.3.4 | User name preserved | Config user.name before rebuild | Name still set after | PASS |
| 27.3.5 | Links reconstructed | Linked specs, rebuild | Links restored from frontmatter | PASS |
| 27.3.6 | Dependencies reconstructed | Tasks with deps, rebuild | Dependencies restored | PASS |
| 27.3.7 | Citations reconstructed | Cited chunks, rebuild | Citations restored | FAIL |
| 27.3.8 | Criteria reconstructed | Tasks with criteria, rebuild | Criteria positions and checked states restored | PASS |
| 27.3.9 | KB chunks reconstructed | KB docs, rebuild | All chunks recreated | PASS |
| 27.3.10 | TF-IDF connections rebuilt | KB docs with connections, rebuild | Connections recomputed | FAIL |
| 27.3.11 | Rejected files reported | Non-canonical files in specd/ | Listed in rebuild output | PASS |
| 27.3.12 | Canonical files ingested | Properly named files | Ingested during rebuild | PASS |
| 27.3.13 | --force flag | `specd rebuild --force` | Succeeds even with existing cache | PASS |

### 27.4 Status

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.4.1 | status basic | `specd status` | Counts for specs, tasks, KB, trash | PASS |
| 27.4.2 | status --detailed | `specd status --detailed` | Includes lint summary | PASS |
| 27.4.3 | JSON output | `specd status --json` | Structured JSON | PASS |

### 27.5 Merge-Fixup

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.5.1 | No duplicates | Clean workspace | Empty result | PASS |
| 27.5.2 | Duplicate spec IDs | Two SPEC-1 directories | Detected, second renumbered | SKIP |
| 27.5.3 | Duplicate task IDs | Two TASK-1 files in different spec dirs | Detected, one renumbered | SKIP |
| 27.5.4 | Duplicate KB IDs | Two KB-1 files | Detected, one renumbered | SKIP |
| 27.5.5 | Rebuild after fixup | Fixup renumbers items | Full rebuild triggered | SKIP |
| 27.5.6 | Clean sidecar renamed | Duplicate HTML KB | .clean.html also renamed | SKIP |
| 27.5.7 | Counter updated | After renumbering | next_*_id reflects new max | SKIP |

---

## 28. CLI - Trash Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 28.1 | trash list | `specd trash list` | All trashed items | PASS |
| 28.2 | trash list --kind | `--kind spec` | Only spec trash | PASS |
| 28.3 | trash list --older-than | `--older-than 7d` | Only items older than 7 days | PASS |
| 28.4 | trash restore | `specd trash restore 1` | Item restored, file recreated | PASS |
| 28.5 | trash restore with new ID | Restore when original ID reused | New ID allocated, warning printed | SKIP |
| 28.6 | trash restore task orphan | Restore task whose spec was deleted | Error: parent spec no longer exists | SKIP |
| 28.7 | trash purge | `specd trash purge --older-than 30d` | Items older than 30d permanently deleted | PASS |
| 28.8 | trash purge-all | `specd trash purge-all` | All trash permanently deleted | PASS |
| 28.9 | trash purge returns count | Run purge | Outputs number of purged items | PASS |
| 28.10 | Duration parsing: days | `--older-than 30d` | Parsed correctly | PASS |
| 28.11 | Duration parsing: hours | `--older-than 24h` | Parsed correctly | PASS |
| 28.12 | Duration parsing: minutes | `--older-than 60m` | Parsed correctly | PASS |
| 28.13 | Duration parsing: invalid | `--older-than xyz` | Error: invalid duration | PASS |

---

## 29. CLI - Serve & Watch

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 29.1 | serve starts server | `specd serve` | HTTP server on port 7823 | PASS |
| 29.2 | serve custom port | `specd serve --port 9000` | Server on port 9000 | PASS |
| 29.3 | serve starts watcher | `specd serve` | File watcher running alongside server | PASS |
| 29.4 | serve responds to requests | Curl `http://localhost:7823/` | HTML response | PASS |
| 29.5 | watch standalone | `specd watch` | Watcher running without HTTP server | PASS |
| 29.6 | Ctrl+C stops serve | Send SIGINT | Graceful shutdown | PASS |
| 29.7 | Ctrl+C stops watch | Send SIGINT | Watcher stops | PASS |
| 29.8 | serve --open flag | `specd serve --open` | Server starts and browser auto-opens | SKIP |
| 29.9 | Concurrent CLI lock timeout | Start serve, run `specd new-spec` in another terminal during a write | Second process waits up to 5s, then errors if still locked | PASS |

---

## 30. Watcher - Sync Behavior

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 30.1 | Edit spec title in editor | Modify title in frontmatter, save | SQLite title updated within ~1s | PASS |
| 30.2 | Edit spec type | Change type field | SQLite type updated | PASS |
| 30.3 | Edit spec summary | Change summary | SQLite summary updated | PASS |
| 30.4 | Edit spec body | Add content to body | SQLite body updated | PASS |
| 30.5 | Edit task title | Modify task frontmatter | SQLite updated | PASS |
| 30.6 | Edit task status | Change status in frontmatter | SQLite status updated | PASS |
| 30.7 | Check acceptance criterion | Change `[ ]` to `[x]` in editor | task_criteria.checked updated | PASS |
| 30.8 | Uncheck criterion | Change `[x]` to `[ ]` | task_criteria.checked updated | PASS |
| 30.9 | Add criterion manually | Add `- [ ] New item` to criteria section | New row in task_criteria | PASS |
| 30.10 | CLI write not double-processed | Create spec via CLI | Watcher sees matching hash, skips | PASS |
| 30.11 | FTS index updated | Edit spec title | FTS5 search finds new title | PASS |
| 30.12 | Trigram index updated | Edit spec body | Trigram search finds new content | PASS |
| 30.13 | content_hash updated | External edit | content_hash recalculated | PASS |
| 30.14 | updated_at updated | External edit | Timestamp refreshed | PASS |
| 30.15 | System fields preserved | Edit linked_specs manually | SQLite version wins (log drift warning) | SKIP |
| 30.16 | index.md changes ignored | Edit index.md | Watcher skips auto-maintained file | PASS |
| 30.17 | log.md changes ignored | Edit log.md | Watcher skips auto-maintained file | PASS |

---

## 31. Watcher - Deletion Handling

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 31.1 | Delete spec directory | `rm -rf specd/specs/SPEC-1-auth/` | Spec and all tasks soft-deleted to trash | PASS |
| 31.2 | Delete task file | `rm specd/specs/SPEC-1-auth/TASK-1-design.md` | Task soft-deleted to trash | PASS |
| 31.3 | Delete KB file | `rm specd/kb/KB-1-doc.md` | KB doc soft-deleted to trash | PASS |
| 31.4 | Trash entry created | Delete file externally | trash row with deleted_by='watcher' | PASS |
| 31.5 | Citations cleaned | Delete spec/task with citations | Citations deleted | PASS |
| 31.6 | Clean sidecar ignored | Delete .clean.html file | Watcher skips sidecar | PASS |

---

## 32. Watcher - Rejection Logic

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 32.1 | Non-canonical spec dir | Create `specd/specs/my-spec/spec.md` | Inserted into rejected_files | PASS |
| 32.2 | Non-canonical task file | Create `specd/specs/SPEC-1-auth/my-task.md` | Rejected | PASS |
| 32.3 | Non-canonical KB file | Create `specd/kb/my-doc.md` | Rejected | SKIP |
| 32.4 | File outside specd/ | Create file in project root | Watcher ignores (filtered by path) | PASS |
| 32.5 | Rejected file reason | Check rejected entry | Reason explains why file was rejected | PASS |
| 32.6 | File left on disk | Non-canonical file created | File not deleted, just recorded | PASS |

---

## 33. Watcher - Debouncing

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 33.1 | Single save processed once | Save file once | One sync operation | PASS |
| 33.2 | Rapid saves debounced | Save same file 5 times in 100ms | Only final state processed | PASS |
| 33.3 | Different files processed independently | Save two different files simultaneously | Both processed | PASS |
| 33.4 | 200ms debounce window | Save, wait 200ms, save again | Both edits processed as separate events | PASS |
| 33.5 | Timer cancelled on new event | Start timer, save again before 200ms | First timer cancelled, new timer starts | PASS |
| 33.6 | Shutdown flushes pending | Stop watcher with pending timer | Pending events flushed before exit | SKIP |

---

## 34. Integration - Spec Lifecycle

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 34.1 | Create, read, update, delete | Full CRUD cycle via CLI | All operations succeed, data consistent | PASS |
| 34.2 | Create via CLI, view in UI | `specd new-spec`, open browser | Spec appears on specs page | PASS |
| 34.3 | Create via UI, read via CLI | Create in browser, `specd read` | CLI returns spec data | PASS |
| 34.4 | Edit in editor, see in UI | Edit spec.md, reload browser | Updated content displayed | PASS |
| 34.5 | Delete via UI, verify files | Delete spec in browser | Directory removed, trash entry exists | PASS |
| 34.6 | Rename cascades tasks | Rename spec | Task paths updated | SKIP |
| 34.7 | Delete cascades everything | Delete spec with tasks, links, citations | All related data cleaned up | PASS |
| 34.8 | Tidy reminder on responses | Don't run tidy for 7+ days | new-spec response includes tidy_reminder | SKIP |

---

## 35. Integration - Task Lifecycle

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 35.1 | Create, move through statuses, complete | Create backlog -> todo -> in_progress -> done | All transitions work | PASS |
| 35.2 | Full kanban workflow | Create task, drag through columns in UI | Statuses update, positions correct | PASS |
| 35.3 | Criteria workflow | Add criteria, check items, verify completion | Progress updates in real-time | PASS |
| 35.4 | Block and unblock | Add dependency, complete blocker | Blocked task becomes ready | PASS |
| 35.5 | Task with citations round-trip | Cite KB chunk, read task | Citations visible in CLI and UI | PASS |
| 35.6 | Multiple tasks under one spec | Create 10 tasks under SPEC-1 | All shown in spec detail, board | PASS |

---

## 36. Integration - KB Lifecycle

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 36.1 | Add doc, cite from spec, view in reader | Full flow | Citation links to reader, chunk highlighted | PASS |
| 36.2 | Add 3 related docs | Add docs with overlapping terminology | TF-IDF connections created between them | PASS |
| 36.3 | Remove doc with citations | Delete cited KB doc | Citations cleaned from citing specs/tasks | PASS |
| 36.4 | Rebuild connections after bulk add | Add 5 docs, rebuild-connections | All cross-doc connections recomputed | PASS |
| 36.5 | KB search after add | Add doc, search for its content | Found in search results | PASS |

---

## 37. Integration - Citation Flow

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 37.1 | Cite from spec detail | View spec, see citation card | Source type icon, title, preview text, page number | PASS |
| 37.2 | View in source from citation | Click "View in source" | KB reader opens at correct chunk | PASS |
| 37.3 | Multiple citations per entity | Spec cites 3 different KB chunks | All 3 shown as cards | PASS |
| 37.4 | Citation survives spec update | Update spec title | Citations unchanged | PASS |
| 37.5 | Citation removed on uncite | Uncite via CLI | Citation card disappears from UI | PASS |
| 37.6 | Citation cascade on KB remove | Remove KB doc | All citations referencing it removed | PASS |

---

## 38. Integration - Dependency Graph

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 38.1 | Chain: A -> B -> C | Create 3 tasks, chain dependencies | Next shows C only when A and B done | PASS |
| 38.2 | Diamond: A,B -> C | C depends on both A and B | C ready only when both A and B done | PASS |
| 38.3 | Partial completion | A done, B in_progress | C still blocked | PASS |
| 38.4 | Cancel unblocks | A cancelled | Dependents become ready | PASS |
| 38.5 | Wontfix unblocks | A set to wontfix | Dependents become ready | PASS |
| 38.6 | Cycle detection on complex graph | Attempt to create cycle in 5-node graph | Error with full cycle path | PASS |
| 38.7 | Board shows blocked icon | Task has unresolved dep | Warning icon on kanban card | PASS |
| 38.8 | Next ordering with deps | Multiple tasks with varying readiness | Ready tasks first, then progress, then position | PASS |

---

## 39. Integration - Search Across Entities

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 39.1 | Term found in spec, task, and KB | Create all 3 with shared term | Search returns results in all 3 groups | PASS |
| 39.2 | Deleted entities not in search | Delete a spec, search for its title | Not found in results | PASS |
| 39.3 | Updated content searchable | Update spec body, search new term | Found in results | PASS |
| 39.4 | Trigram finds partial match | Search for substring of a title | Trigram result returned | PASS |
| 39.5 | Porter stemming works | Search "running" matches "runs" | BM25 with stemming finds match | PASS |

---

## 40. Integration - Rebuild & Merge-Fixup

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 40.1 | Rebuild restores full state | Create data, delete cache.db, rebuild | All specs, tasks, KB, links, deps, citations restored | PASS |
| 40.2 | Rebuild + search works | Rebuild, then search | FTS and trigram indexes functional | PASS |
| 40.3 | Rebuild + next works | Rebuild, then `specd next` | Correct ordering with dependencies | PASS |
| 40.4 | Merge-fixup + rebuild | Create duplicates, fixup | Clean workspace after fixup | SKIP |
| 40.5 | Merge-fixup preserves non-conflicting | 3 specs, only 1 duplicate | Other 2 specs unchanged | SKIP |

---

## 41. Non-Functional - Performance

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 41.1 | Board load with 100 specs, 1000 tasks | Page renders in <500ms | PASS |
| 41.2 | Search response time | Results returned in <100ms for 1000-item workspace | PASS |
| 41.3 | Next command response | Ready tasks returned in <50ms | PASS |
| 41.4 | Lint completion time | Completes in <1s for 100 specs + 1000 tasks | PASS |
| 41.5 | Rebuild time | Completes in reasonable time for 100 specs, 1000 tasks, 50 KB docs | PASS |
| 41.6 | Watcher sync latency | File change reflected in SQLite within 1s | PASS |
| 41.7 | Drag-and-drop responsiveness | Board re-renders in <300ms after drop | PASS |
| 41.8 | KB reader markdown render | Large markdown doc renders in <500ms | PASS |
| 41.9 | KB reader PDF render (100 pages) | All pages render, navigation smooth | SKIP |
| 41.10 | TF-IDF computation | Connections computed in <5s for 50 docs | PASS |
| 41.11 | Static asset load time | CSS + JS bundle loads in <200ms (all local) | PASS |
| 41.12 | Concurrent CLI invocations | Second invocation waits up to 5s on lock | PASS |

---

## 42. Non-Functional - Security

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 42.1 | XSS in spec title | HTML entities escaped in rendered output | PASS |
| 42.2 | XSS in task body | Markdown rendered safely (no raw HTML injection) | PASS |
| 42.3 | XSS in KB doc title | Title HTML-escaped in templates | PASS |
| 42.4 | XSS in search query | Query echoed safely in results page | PASS |
| 42.5 | XSS in criterion text | Text rendered as plain text, not HTML | PASS |
| 42.6 | XSS in error messages | Error query param HTML-escaped in snackbar | PASS |
| 42.7 | HTML sanitization at ingest | `<script>` tags removed by bluemonday | PASS |
| 42.8 | HTML iframe sandboxed | `sandbox="allow-same-origin"` only, no `allow-scripts` | PASS |
| 42.9 | PDF.js JavaScript disabled | Embedded PDF JS not executed | SKIP |
| 42.10 | Path traversal on `/api/kb/{id}/raw` | Manipulated paths blocked (403) | PASS |
| 42.11 | Path constrained to specd/kb/ | Symlink escaping blocked | PASS |
| 42.12 | SQL injection via search | Special characters in search query | FTS query properly escaped | PASS |
| 42.13 | SQL injection via ID parameters | `'; DROP TABLE specs;--` as ID | Parameterized queries prevent injection | PASS |
| 42.14 | CSRF on mutation endpoints | POST without proper origin | Server-side validation (localhost only) | PASS |
| 42.15 | File upload size limits | Upload very large file | Handled gracefully (no OOM) | PASS |
| 42.16 | Lock file timeout | Concurrent writers | Second writer gets clean error after 5s | PASS |
| 42.17 | No external network requests | Monitor network during full UI usage | Zero outbound requests | PASS |
| 42.18 | Content-Type headers correct | Check all API responses | Correct MIME types set | PASS |
| 42.19 | No sensitive data in error messages | Trigger various errors | No file paths or SQL exposed to client | PASS |

---

## 43. Non-Functional - Accessibility

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 43.1 | Semantic HTML structure | Pages use proper heading hierarchy (h1->h2->h3) | PASS |
| 43.2 | ARIA labels on interactive elements | Buttons, toggles, dialogs have aria-label | PASS |
| 43.3 | ARIA-expanded on sidebar toggle | Toggle updates aria-expanded attribute | PASS |
| 43.4 | ARIA-pressed on highlight toggle | Button reflects current state | PASS |
| 43.5 | Dialog aria-label | All dialogs have descriptive aria-label | PASS |
| 43.6 | Form labels associated with inputs | All form fields have proper `<label>` or aria-label | PASS |
| 43.7 | Error alerts with role="alert" | Form validation errors have `role="alert"` | PASS |
| 43.8 | Snackbar with role="alert" | Error snackbar has `role="alert"` | PASS |
| 43.9 | Skip to main content | Tab from top of page | Focus reaches main content efficiently | SKIP |
| 43.10 | Keyboard-navigable nav | Tab through navigation | All nav links focusable | PASS |
| 43.11 | Keyboard-navigable forms | Tab through form fields | All fields reachable | PASS |
| 43.12 | Keyboard-navigable KB reader | Arrow keys for chunks | Navigation works without mouse | PASS |
| 43.13 | Color contrast in light mode | Check text contrast ratios | Meets WCAG AA (4.5:1 for text) | PASS |
| 43.14 | Color contrast in dark mode | Check dark theme | Meets WCAG AA | PASS |
| 43.15 | Focus indicators visible | Tab through UI | Focus ring visible on all interactive elements | PASS |
| 43.16 | Screen reader for kanban cards | Use screen reader | Card content readable (title, spec, progress) | SKIP |
| 43.17 | Drag-and-drop alternative | Keyboard-only status change | Status dropdown on task detail as alternative | PASS |
| 43.18 | Image alt text | Check all images/icons | Decorative icons aria-hidden, meaningful ones have alt | PASS |
| 43.19 | Responsive text sizing | Zoom to 200% | Content readable, no overflow | PASS |
| 43.20 | Mobile touch targets | Check button sizes on mobile | Minimum 44x44px touch targets | SKIP |

---

## 44. Non-Functional - Reliability & Data Integrity

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 44.1 | DB and markdown in sync | After any mutation, DB matches markdown frontmatter | PASS |
| 44.2 | Transactional writes | Spec creation fails mid-write | No partial state (no orphan directory without DB row) | PASS |
| 44.3 | Lock prevents corruption | Two CLI commands simultaneously | Second waits or errors cleanly | PASS |
| 44.4 | Graceful shutdown | Kill server during write | No corrupt DB (WAL journal recovery) | PASS |
| 44.5 | Rebuild recovers from corrupt DB | Delete cache.db, rebuild | Full recovery from markdown | PASS |
| 44.6 | Content hash prevents double-processing | CLI write followed by watcher event | Watcher skips (hashes match) | PASS |
| 44.7 | Cascade deletes complete | Delete spec with deep graph | All tasks, links, deps, citations, criteria removed | PASS |
| 44.8 | FTS indexes consistent | After mutations | Search results match actual data | PASS |
| 44.9 | Trigram indexes consistent | After mutations | Substring search finds expected results | PASS |
| 44.10 | Counter monotonically increases | Create 5 specs, delete 3, create 2 more | IDs never reused (SPEC-6, SPEC-7) | PASS |
| 44.11 | Trash preserves full state | Restore trashed spec | Title, type, body, summary all preserved | PASS |
| 44.12 | KB clean sidecar integrity | Add HTML, verify sidecar | Sidecar sanitized by bluemonday, scripts stripped | PASS |
| 44.13 | PDF chunk page tracking | Add multi-page PDF | Each chunk has correct page number | SKIP |
| 44.14 | Criteria positions contiguous | Remove middle criterion | Remaining criteria renumbered 1, 2, 3... | PASS |
| 44.15 | Bidirectional links consistent | Link A->B | Both A->B and B->A exist in spec_links | PASS |
| 44.16 | Dependency directed correctly | Depend TASK-2 on TASK-1 | blocker_task=TASK-1, blocked_task=TASK-2 | PASS |
| 44.17 | Workspace path with spaces | `specd init "/tmp/my workspace"` | Init succeeds, all operations work with quoted paths | SKIP |
| 44.18 | Symlinks inside workspace | Symlink within specd/ pointing to another file inside specd/ | Followed and processed normally | SKIP |
| 44.19 | Symlinks outside workspace | Symlink inside specd/ pointing to /etc/passwd | Ignored with warning, not followed | SKIP |
| 44.20 | Large KB file rejection | Add file that would produce >10,000 chunks | Clean error, no partial KB doc left | SKIP |
| 44.21 | Slug uniqueness | Create two specs with similar titles | Both get unique directory names | PASS |
| 44.22 | Title with unicode characters | Create spec with title "Authentifizierung und Sicherheit" | Title preserved, slug generated correctly | PASS |

---

## 45. Non-Functional - Responsiveness & Layout

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 45.1 | Desktop layout (>=993px) | Sidebar visible, full-width content | PASS |
| 45.2 | Tablet layout (601-992px) | Top bar, content fills width | PASS |
| 45.3 | Mobile layout (<601px) | Top bar with hamburger, stacked content | SKIP |
| 45.4 | Kanban columns scroll | 8 columns on small screen | Horizontal scroll or responsive reflow | PASS |
| 45.5 | Long spec title wraps | 200-char title | Text wraps, no overflow | PASS |
| 45.6 | Long KB document readable | Very wide HTML content | Contained within reader bounds | PASS |
| 45.7 | Dialog responsive | Open dialog on mobile | Dialog fits viewport, scrollable if needed | PASS |
| 45.8 | Form fields full-width on mobile | Open form on mobile | Inputs span full width | PASS |
| 45.9 | Table scrollable on mobile | Status or trash table with many columns | Horizontal scroll on overflow | PASS |
| 45.10 | Footer doesn't overlap content | Short content page | Footer at bottom, not overlapping | PASS |
| 45.11 | KB reader sidebar responsive | Toggle sidebar on mobile | Sidebar overlays content or collapses | PASS |

---

## 46. Non-Functional - Offline & Embedded Assets

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 46.1 | No CDN requests | Load all pages with network tab open | Zero external requests | PASS |
| 46.2 | BeerCSS loads from /vendor/ | Check CSS link | Local `beer.min.css` served | PASS |
| 46.3 | htmx loads from /assets/ | Check script src | Local `htmx.min.js` | PASS |
| 46.4 | marked.js loads locally | Check script src | Local `marked.min.js` | PASS |
| 46.5 | PDF.js loads locally | Open PDF in reader | Worker loaded from `/assets/pdf.worker.min.mjs` | SKIP |
| 46.6 | Fonts embedded | Check font requests | No Google Fonts or external font CDN calls | PASS |
| 46.7 | Material icons local | Check icon rendering | Icons render without network | PASS |
| 46.8 | Theme JS local | Check BeerCSS + material-dynamic-colors | Both load from embedded assets | PASS |
| 46.9 | App works with network disabled | Disconnect network, use app | All features functional | PASS |
| 46.10 | Binary is self-contained | Run binary on fresh machine | No runtime dependencies needed | PASS |
| 46.11 | CSS bundle served correctly | Check response headers | Correct Content-Type and caching headers | PASS |
| 46.12 | Favicon served | Check browser tab | Favicon displayed | PASS |
