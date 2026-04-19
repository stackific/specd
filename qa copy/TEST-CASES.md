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
| 1.1.1 | Empty board renders | Navigate to `/` with no tasks | All 8 columns visible with headers and "No tasks" empty state in each | |
| 1.1.2 | Board with tasks | Create tasks in various statuses | Tasks appear in correct columns with correct counts in badges | |
| 1.1.3 | Column headers show icons | Load board | Each column displays its Material icon (inventory_2, checklist, pending, block, verified, check_circle, cancel, do_not_disturb_on) | |
| 1.1.4 | Task card shows title | Create a task | Card displays task title as clickable link | |
| 1.1.5 | Task card shows spec title | Create a task under a spec | Card displays parent spec title | |
| 1.1.6 | Task card shows criteria progress | Create task with 3 criteria, check 2 | Card shows circular progress indicator reading "2/3" | |
| 1.1.7 | Task card shows blocked icon | Create task with unresolved dependency | Warning icon visible on card | |
| 1.1.8 | Task card shows citation icon | Create task with KB citation | Citation/link icon visible on card | |
| 1.1.9 | Full-page load (no HX-Request) | Open `/` in new browser tab | Full HTML page with base layout, nav, footer rendered | |
| 1.1.10 | HTMX partial load | Click "Board" in nav from another page | Only content block swapped, no full page reload | |
| 1.1.11 | Column ordering left-to-right | Load board | Columns in order: Backlog, To Do, In Progress, Blocked, Verification, Done, Cancelled, Won't Fix | |
| 1.1.12 | Column task count badge | Add 3 tasks to "todo" | "To Do" column header badge shows "3" | |
| 1.1.13 | Task card shows created_by | Config user.name, create task | Card or tooltip shows creator name | |

### 1.2 Spec Filter

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.2.1 | Filter dropdown lists all specs | Create 3 specs, load board | Dropdown shows "All specs" + all 3 spec titles | |
| 1.2.2 | Filter by spec | Select a spec from dropdown | Board shows only tasks belonging to selected spec | |
| 1.2.3 | Clear filter | Select "All specs" after filtering | All tasks visible again | |
| 1.2.4 | Filter preserves on htmx reload | Filter by spec, then perform board action | Filter dropdown retains selected spec after content swap | |
| 1.2.5 | Filter with no matching tasks | Select spec with no tasks | All columns show "No tasks" empty state | |

### 1.3 New Task Dialog (from Board)

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.3.1 | Dialog opens | Click "New Task" button | Dialog appears with form fields: spec_id, title, summary, status, body | |
| 1.3.2 | Button hidden when no specs | Load board with zero specs | "New Task" button not rendered | |
| 1.3.3 | Spec dropdown populated | Open dialog with 3 specs | All specs listed in spec_id select | |
| 1.3.4 | Status defaults to backlog | Open dialog | Status select pre-selects "backlog" | |
| 1.3.5 | Valid submission | Fill all required fields, submit | Dialog closes, task appears in correct column, redirect to task detail | |
| 1.3.6 | Empty title rejected | Submit with blank title | Form re-renders with error "Title must be at least 2 characters", HTTP 422 | |
| 1.3.7 | Whitespace-only title rejected | Submit title "   " | Validation error displayed, form preserved | |
| 1.3.8 | Title too short rejected | Submit title "A" | Validation error: title must be 2+ chars | |
| 1.3.9 | Body too short rejected | Submit body "Short" | Validation error: body must be 20+ chars | |
| 1.3.10 | Empty body rejected | Submit with blank body | Validation error displayed | |
| 1.3.11 | Form preserves input on error | Submit invalid form | All previously entered values preserved in form fields | |
| 1.3.12 | Cancel closes dialog | Click cancel button | Dialog closes, no task created | |
| 1.3.13 | Select "todo" status | Choose "todo" from status dropdown, submit | Task created in todo column | |
| 1.3.14 | Select "in_progress" status | Choose "in_progress", submit | Task created in In Progress column | |
| 1.3.15 | Summary auto-generated | Leave summary blank, submit valid form | Task created with auto-generated summary | |
| 1.3.16 | Consecutive spaces collapsed in title | Submit title "Auth   Flow   System" | Title stored as "Auth Flow System" | |
| 1.3.17 | Escape key closes dialog | Press Escape while dialog is open | Dialog closes, no task created | |

### 1.4 Drag-and-Drop

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.4.1 | Drag card to different column | Drag from backlog to todo | Task moves to todo column, status updated in DB | |
| 1.4.2 | Drag card within same column | Drag card above another in same column | Card reordered, position updated in DB | |
| 1.4.3 | Drop placeholder appears | Drag card over another column | Visual drop placeholder shown at target position | |
| 1.4.4 | Drag to empty column | Drag card to column with no cards | Card placed in empty column, status updated | |
| 1.4.5 | Board re-renders after move | Complete a drag-move | Entire board re-fetched and kanban re-initialized | |
| 1.4.6 | Board re-renders after reorder | Complete a drag-reorder | Board re-fetched, new order preserved | |
| 1.4.7 | Multiple rapid drags | Quickly drag 3 cards in succession | All moves processed correctly, no data loss | |
| 1.4.8 | Drag across all 8 columns | Move a card through each status | Card ends in correct final column | |
| 1.4.9 | Card link still works after re-render | Drag a card, then click its title | Navigates to task detail page | |
| 1.4.10 | Error on move shows snackbar | Simulate server error on `/api/board/move` | Error snackbar appears, card returns to original column | |

### 1.5 Card Navigation

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 1.5.1 | Click card title | Click task title on card | Navigates to `/tasks/{id}` | |
| 1.5.2 | Click spec title on card | Click spec label on card | Navigates to `/specs/{specId}` | |

---

## 2. Web UI - Specs List

### 2.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 2.1.1 | Empty specs page | Navigate to `/specs` with no specs | "No specs yet" empty state message | |
| 2.1.2 | Specs grouped by type | Create business, functional, non-functional specs | Three groups rendered with correct labels | |
| 2.1.3 | Group shows spec count | Create 3 functional specs | Functional group shows all 3 | |
| 2.1.4 | Spec card shows title | Load specs page | Each card displays clickable spec title | |
| 2.1.5 | Spec card shows type badge | Load specs page | Each card shows type chip (business/functional/non-functional) | |
| 2.1.6 | Spec card shows task progress | Create spec with 5 tasks (3 done) | Progress bar shows 60% | |
| 2.1.7 | Spec card links to detail | Click spec title | Navigates to `/specs/{id}` | |
| 2.1.8 | Groups only rendered if specs exist | Create only functional specs | Only "Functional" group header rendered | |

### 2.2 New Spec Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 2.2.1 | Dialog opens | Click "New Spec" button | Dialog with title, type, summary, body fields | |
| 2.2.2 | Type dropdown options | Open dialog | Three options: business, functional, non-functional | |
| 2.2.3 | Valid submission | Fill all fields, submit | Redirects to `/specs/{newId}` | |
| 2.2.4 | Title min 2 chars enforced | Submit title "X" | Error: title must be at least 2 characters | |
| 2.2.5 | Body min 20 chars enforced | Submit body "Short body" | Error: body must be at least 20 characters | |
| 2.2.6 | Type is required | Submit without selecting type | Browser-level required validation blocks submission | |
| 2.2.7 | Form preserved on validation error | Submit invalid, inspect form | All user-entered values retained | |
| 2.2.8 | HTTP 422 on validation error | Submit invalid form | Response status 422, form partial re-rendered | |
| 2.2.9 | Summary is optional | Submit without summary | Spec created successfully | |
| 2.2.10 | Long title (250 chars) | Submit title of exactly 250 characters | Spec created, title fully preserved | |
| 2.2.11 | Title with special characters | Submit title: `OAuth & GitHub "Login" <Flow>` | Title preserved with correct escaping in HTML | |

---

## 3. Web UI - Spec Detail

### 3.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.1.1 | Spec header renders | Navigate to `/specs/SPEC-1` | Title, type badge, ID displayed | |
| 3.1.2 | Back link to specs | Click back chip | Navigates to `/specs` | |
| 3.1.3 | Markdown body rendered | Create spec with markdown body | Body rendered as HTML (headers, lists, code blocks) | |
| 3.1.4 | Summary displayed | Spec has summary text | Summary shown below title | |
| 3.1.5 | Progress bar shown | Spec has tasks | Progress bar with done/active counts | |
| 3.1.6 | No progress bar if no tasks | Spec has zero tasks | Progress section hidden | |
| 3.1.7 | Linked specs section | Link two specs | "Related Specs" section shows linked spec with link | |
| 3.1.8 | No linked specs section | Spec has no links | Related Specs section hidden | |
| 3.1.9 | Citations section | Cite KB chunk from spec | "References" section shows citation card | |
| 3.1.10 | Citation card content | Cite KB-1:3 | Card shows source type icon, doc title, chunk position, text preview, "View in source" link | |
| 3.1.11 | Citation "View in source" | Click link on citation | Navigates to `/kb/{docId}?chunk={position}` | |
| 3.1.12 | No citations section | Spec with no citations | References section hidden | |
| 3.1.13 | Task grid renders | Spec has 5 tasks | Tasks shown in grid layout with status and title | |
| 3.1.14 | Task card links to detail | Click task in grid | Navigates to `/tasks/{taskId}` | |
| 3.1.15 | Empty task grid | Spec with no tasks | "No tasks yet" message shown | |
| 3.1.16 | 404 for nonexistent spec | Navigate to `/specs/SPEC-999` | Error page with 404 code | |
| 3.1.17 | Markdown body renders tables | Spec body has GFM table | Table rendered with borders and alignment | |
| 3.1.18 | Markdown body renders code blocks | Spec body has fenced code block | `<pre><code>` rendered with language class | |
| 3.1.19 | Markdown body renders nested lists | Spec body has nested bullet/numbered lists | Proper indentation and nesting | |
| 3.1.20 | Markdown body renders inline links | Body has `[link](url)` | Rendered as clickable `<a>` tag | |
| 3.1.21 | Progress all tasks done | 5 tasks all in "done" | Progress bar at 100% | |
| 3.1.22 | Progress all tasks cancelled | 3 tasks all "cancelled" | Active count = 0, progress N/A or 0% | |
| 3.1.23 | Progress mixed wontfix and done | 2 done, 1 wontfix, 1 in_progress | Active = 3 (total - wontfix), done/active percent | |
| 3.1.24 | Escape key closes edit dialog | Press Escape in edit spec dialog | Dialog closes without saving | |

### 3.2 Edit Spec Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.2.1 | Dialog pre-fills values | Click edit button | Title, type, summary, body populated with current values | |
| 3.2.2 | Update title | Change title, submit | Title updated on page, slug unchanged | |
| 3.2.3 | Update type | Change from functional to business | Type badge updated | |
| 3.2.4 | Update body | Edit body markdown, submit | New body rendered | |
| 3.2.5 | Validation on edit | Clear title, submit | 422 error with validation message | |
| 3.2.6 | Cancel preserves original | Open edit, change title, cancel | Original title still shown | |

### 3.3 Delete Spec

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.3.1 | Confirmation dialog | Click delete button | Confirmation dialog mentions cascading task deletion | |
| 3.3.2 | Confirm delete | Click delete in confirmation | Redirected away, spec removed from list | |
| 3.3.3 | Cancel delete | Click cancel in confirmation | Spec still exists | |
| 3.3.4 | Cascade deletes tasks | Delete spec with tasks | All child tasks also trashed | |
| 3.3.5 | Spec appears in trash | Delete spec | Spec visible in `/trash` | |

### 3.4 New Task Dialog (from Spec Detail)

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 3.4.1 | Spec ID pre-filled | Open new task dialog from spec detail | spec_id hidden field set to current spec | |
| 3.4.2 | Valid submission | Fill fields, submit | Task created under current spec, redirected to task detail | |
| 3.4.3 | Validation error shows spec-detail variant | Submit invalid form | Error form rendered in spec-detail context (not board variant) | |

---

## 4. Web UI - Task Detail

### 4.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.1.1 | Task header renders | Navigate to `/tasks/TASK-1` | Title, ID, parent spec ID displayed | |
| 4.1.2 | Back to parent spec | Click parent spec link | Navigates to `/specs/{specId}` | |
| 4.1.3 | Back to board (no parent) | Parent spec deleted | Back link goes to `/` | |
| 4.1.4 | Status dropdown shows current | Task in "todo" status | Status dropdown pre-selects "todo" | |
| 4.1.5 | Markdown body rendered | Task has markdown body | Rendered as HTML | |
| 4.1.6 | Summary displayed | Task has summary | Summary text shown | |
| 4.1.7 | 404 for nonexistent task | Navigate to `/tasks/TASK-999` | Error page with 404 | |
| 4.1.8 | Markdown body renders in task | Task body has headers, lists, code | Rendered as HTML correctly | |
| 4.1.9 | created_by / updated_by shown | Task has creator configured | Attribution info displayed | |

### 4.2 Status Change

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.2.1 | Auto-submit on change | Select different status from dropdown | Form auto-submits, page reloads with new status | |
| 4.2.2 | Move backlog to todo | Change status to todo | Status updated, task in todo column on board | |
| 4.2.3 | Move to in_progress | Change to in_progress | Status persisted | |
| 4.2.4 | Move to blocked | Change to blocked | Status persisted | |
| 4.2.5 | Move to pending_verification | Change status | Status persisted | |
| 4.2.6 | Move to done | Change to done | Status persisted | |
| 4.2.7 | Move to cancelled | Change to cancelled | Status persisted | |
| 4.2.8 | Move to wontfix | Change to wontfix | Status persisted | |
| 4.2.9 | All 8 statuses selectable | Open dropdown | All 8 status options present | |

### 4.3 Acceptance Criteria

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.3.1 | Criteria list renders | Task has 3 criteria | All 3 shown with checkboxes | |
| 4.3.2 | Check criterion | Click unchecked checkbox | Criterion marked as checked, page re-renders | |
| 4.3.3 | Uncheck criterion | Click checked checkbox | Criterion unchecked, page re-renders | |
| 4.3.4 | Add criterion | Enter text, submit | New criterion appended to list | |
| 4.3.5 | Add criterion validation | Submit empty text | Form not submitted (HTML required attribute) | |
| 4.3.6 | Add criterion min 2 chars | Submit "A" | Error via redirect | |
| 4.3.7 | Remove criterion | Click remove button on criterion | Criterion removed, remaining renumbered | |
| 4.3.8 | Empty criteria state | Task with no criteria | "No criteria yet." message | |
| 4.3.9 | Criterion text preserved | Add criterion with markdown text | Text displayed as-is (not rendered as markdown) | |
| 4.3.10 | Multiple criteria checked | Check 3 out of 5 | All 3 show checked state, progress updates | |
| 4.3.11 | Criteria synced to markdown | Check criterion via UI | Task markdown file updated with `[x]` | |

### 4.4 Dependencies

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.4.1 | Dependencies section renders | Task has 2 dependencies | Both shown with titles and status | |
| 4.4.2 | Ready dependency icon | Blocker task is "done" | Green check_circle icon shown | |
| 4.4.3 | Not-ready dependency icon | Blocker task is "in_progress" | Red block icon shown | |
| 4.4.4 | Dependency links to task | Click dependency title | Navigates to blocker task detail | |
| 4.4.5 | No dependencies section | Task has no deps | Dependencies section hidden | |

### 4.5 Linked Tasks

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.5.1 | Linked tasks section renders | Task linked to 2 others | Both shown with title and status | |
| 4.5.2 | Link navigates to task | Click linked task title | Navigates to linked task detail | |
| 4.5.3 | No linked tasks section | Task has no links | Section hidden | |

### 4.6 Citations

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.6.1 | Citations section renders | Task cites KB chunk | Citation card with type icon, title, chunk info | |
| 4.6.2 | Citation preview text | Cite a long chunk | Text truncated to ~200 chars | |
| 4.6.3 | View in source link | Click "View in source" | Navigates to KB reader at correct chunk | |
| 4.6.4 | PDF citation shows page | Cite PDF chunk | "page N" displayed on citation card | |
| 4.6.5 | No citations section | Task has no citations | References section hidden | |

### 4.7 Edit Task Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.7.1 | Pre-fills current values | Click edit | Title, summary, body populated | |
| 4.7.2 | Update title | Change title, submit | Title updated on page | |
| 4.7.3 | Update body with criteria | Edit body that has criteria section | Criteria re-parsed from new body | |
| 4.7.4 | Validation on edit | Clear title, submit | 422 error | |
| 4.7.5 | Status preserved on edit | Edit title only | Status unchanged after update | |

### 4.8 Delete Task

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 4.8.1 | Confirmation dialog | Click delete | Confirmation dialog appears | |
| 4.8.2 | Confirm delete | Click delete in dialog | Redirected, task removed | |
| 4.8.3 | Task appears in trash | Delete task | Task visible in trash page | |
| 4.8.4 | Citations cleaned up | Delete task with citations | Citations removed from DB | |

---

## 5. Web UI - Knowledge Base List

### 5.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.1.1 | Empty KB page | No KB docs | "No documents" empty state | |
| 5.1.2 | KB docs table | Add 3 docs | Table shows ID, title, type, date for each | |
| 5.1.3 | Type badge on doc | Add HTML doc | Type chip shows "HTML" | |
| 5.1.4 | Doc links to detail | Click doc title | Navigates to `/kb/{id}` | |

### 5.2 Type Filter

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.2.1 | Filter by Markdown | Click "Markdown" chip | Only .md docs shown | |
| 5.2.2 | Filter by HTML | Click "HTML" chip | Only .html docs shown | |
| 5.2.3 | Filter by PDF | Click "PDF" chip | Only .pdf docs shown | |
| 5.2.4 | Filter by Text | Click "Text" chip | Only .txt docs shown | |
| 5.2.5 | Clear filter (All) | Click "All" chip | All docs shown | |
| 5.2.6 | Active chip highlighted | Click a filter chip | Selected chip visually distinct | |
| 5.2.7 | Filter preserves search query | Search, then filter | Query param preserved in filter links | |

### 5.3 KB Search

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.3.1 | Search returns results | Search for existing term | Search results section with matching chunks | |
| 5.3.2 | Search result shows doc info | Search and get hits | Each result shows doc ID, chunk position, text preview | |
| 5.3.3 | No results message | Search for nonexistent term | "No results" message displayed | |
| 5.3.4 | Empty search shows doc list | Clear search | Full document list shown | |
| 5.3.5 | Search combined with filter | Search + type filter | Results filtered to matching type | |

### 5.4 Add Document Dialog

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.4.1 | Dialog opens | Click "Add Document" | Dialog with file, URL, title, note fields | |
| 5.4.2 | Upload markdown file | Select .md file, submit | Doc added, redirect to `/kb` | |
| 5.4.3 | Upload HTML file | Select .html file, submit | Doc added with clean sidecar created | |
| 5.4.4 | Upload PDF file | Select .pdf file, submit | Doc added with page count | |
| 5.4.5 | Upload text file | Select .txt file, submit | Doc added | |
| 5.4.6 | Add by URL | Enter URL, submit | Doc fetched, added | |
| 5.4.7 | Neither file nor URL | Submit empty form | Error: "Please provide a file or URL" | |
| 5.4.8 | Title optional | Submit without title | Title derived from filename | |
| 5.4.9 | Note optional | Submit without note | Doc created with empty note | |
| 5.4.10 | Custom title | Provide title, submit | Custom title used instead of filename | |
| 5.4.11 | File type validation | Select .exe file | Browser-level accept filter prevents selection (.md,.html,.htm,.pdf,.txt) | |
| 5.4.12 | File name displayed | Select file | Selected filename shown in UI | |
| 5.4.13 | Multipart encoding | Submit with file | Request uses `multipart/form-data` | |
| 5.4.14 | Escape key closes dialog | Press Escape | Dialog closes without adding | |

### 5.5 Delete KB Document

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 5.5.1 | Confirm delete | Click delete, confirm | Doc removed from list | |
| 5.5.2 | Browser confirmation | Click delete | `confirm()` dialog shown | |
| 5.5.3 | Cascade deletes chunks | Delete doc | All chunks and connections removed | |
| 5.5.4 | Cascade deletes citations | Delete cited doc | Citations from specs/tasks cleaned | |
| 5.5.5 | Clean sidecar removed | Delete HTML doc | Both .html and .clean.html removed | |
| 5.5.6 | Doc appears in trash | Delete doc | KB entry in trash | |

---

## 6. Web UI - KB Detail / Reader

### 6.1 Page Load

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.1.1 | Header shows doc info | Navigate to `/kb/KB-1` | Title, type badge, chunk count, date | |
| 6.1.2 | Back to KB link | Click back chip | Navigates to `/kb` | |
| 6.1.3 | PDF shows page count | Open PDF doc detail | Page count displayed in header | |
| 6.1.4 | Note displayed | Doc has note | Note text shown | |
| 6.1.5 | Loading state | Open doc detail | "Loading document..." shown before render | |
| 6.1.6 | 404 for nonexistent doc | Navigate to `/kb/KB-999` | Error page | |

### 6.2 Markdown Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.2.1 | Markdown rendered as HTML | Open .md doc | Headings, lists, code, links rendered | |
| 6.2.2 | Code blocks syntax highlighted | Doc with code blocks | `<pre><code>` elements rendered | |
| 6.2.3 | Active chunk highlighted | Open with `?chunk=3` | Chunk 3 text wrapped in `<mark>` element | |
| 6.2.4 | Highlight scrolled into view | Open deep chunk | Page scrolls to highlighted region | |
| 6.2.5 | Prefix match for highlight | Chunk text split across DOM nodes | Highlight found using first 80 chars | |

### 6.3 Plain Text Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.3.1 | Text rendered in pre | Open .txt doc | Content in `<pre>` element | |
| 6.3.2 | Character offset highlighting | Open with chunk | Chunk highlighted using char_start/char_end | |
| 6.3.3 | HTML entities escaped | Text contains `<script>` literal | Rendered as text, not executed | |

### 6.4 HTML Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.4.1 | Sandboxed iframe | Open .html doc | Content in iframe with `sandbox="allow-same-origin"` | |
| 6.4.2 | No script execution | HTML has `<script>` tags | Scripts not executed (sanitized at ingest) | |
| 6.4.3 | Iframe auto-resizes | Open long HTML doc | Iframe height matches content | |
| 6.4.4 | Chunk highlighted in iframe | Open with `?chunk=N` | Highlight applied inside iframe DOM | |
| 6.4.5 | CSS isolated | HTML has custom CSS | Does not affect parent page styles | |

### 6.5 PDF Rendering

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.5.1 | PDF pages rendered | Open .pdf doc | All pages rendered as canvas elements | |
| 6.5.2 | Text layer overlay | Open .pdf doc | Invisible text layer for selection/highlighting | |
| 6.5.3 | Chunk highlighted | Open with `?chunk=N` | Text highlighted in text layer | |
| 6.5.4 | Page navigation on chunk change | Navigate to chunk on different page | Correct page scrolled into view | |
| 6.5.5 | PDF.js worker loaded | Open any PDF | Worker loads from `/assets/pdf.worker.min.mjs` | |
| 6.5.6 | Large PDF (100+ pages) | Open 100-page PDF | Renders and navigates smoothly | |

### 6.6 Chunk Navigation

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.6.1 | Position display | Open doc with 47 chunks | Shows "1 / 47" (or active chunk position) | |
| 6.6.2 | Next chunk button | Click next | Moves to chunk N+1, highlight updates | |
| 6.6.3 | Previous chunk button | Click previous at chunk 5 | Moves to chunk 4 | |
| 6.6.4 | Previous disabled at first | View chunk 0 | Previous button disabled | |
| 6.6.5 | Next disabled at last | View last chunk | Next button disabled | |
| 6.6.6 | Keyboard navigation (right) | Press right arrow key | Moves to next chunk | |
| 6.6.7 | Keyboard navigation (left) | Press left arrow key | Moves to previous chunk | |
| 6.6.8 | Chunk sidebar toggle | Click toggle button | Sidebar with all chunk previews appears | |
| 6.6.9 | Sidebar chunk click | Click chunk in sidebar | Jumps to clicked chunk, highlight updates | |
| 6.6.10 | Sidebar chunk preview text | Open sidebar | Each chunk shows first ~60 chars | |
| 6.6.11 | Active chunk in sidebar | Navigate to chunk 5 | Chunk 5 highlighted in sidebar list | |
| 6.6.12 | Highlight toggle | Click highlight toggle button | Highlight removed/restored | |

### 6.7 Connections

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.7.1 | Related chunks displayed | Doc has TF-IDF connections | "Related Chunks" section with linked chunks | |
| 6.7.2 | Connection shows target doc | View connections | Target doc title and chunk position shown | |
| 6.7.3 | Connection link navigates | Click related chunk | Navigates to `/kb/{docId}?chunk={pos}` | |
| 6.7.4 | No connections | Chunk has no connections | Section hidden or empty state | |

### 6.8 Citations

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 6.8.1 | Citing specs/tasks shown | Doc chunk is cited | "Referenced By" section with citing entities | |
| 6.8.2 | Citation links to entity | Click citing spec/task | Navigates to spec/task detail | |
| 6.8.3 | No citations | No entities cite this doc | Section hidden | |

---

## 7. Web UI - Search

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 7.1 | Empty search page | Navigate to `/search` | Search input with prompt text, no results | |
| 7.2 | Search returns grouped results | Search for term matching all kinds | Results grouped: Specs, Tasks, KB sections | |
| 7.3 | Spec results | Search matches specs | Shows spec ID, title, summary, score, match type | |
| 7.4 | Task results | Search matches tasks | Shows task ID, title, summary, score, match type | |
| 7.5 | KB results | Search matches KB chunks | Shows doc ID, title, chunk position, text preview | |
| 7.6 | Result count per section | Multiple hits per kind | Section headers show count | |
| 7.7 | Match type badge (BM25) | Primary FTS5 match | Badge shows "bm25" | |
| 7.8 | Match type badge (trigram) | Fuzzy/substring match | Badge shows "trigram" | |
| 7.9 | Score displayed | Search results | Relevance score shown | |
| 7.10 | Result links navigate | Click spec result | Navigates to `/specs/{id}` | |
| 7.11 | Task result links | Click task result | Navigates to `/tasks/{id}` | |
| 7.12 | KB result links | Click KB result | Navigates to `/kb/{id}` | |
| 7.13 | No results message | Search for gibberish | "No results found for 'xyz'" message | |
| 7.14 | Quoted phrase search | Search `"exact phrase"` | FTS phrase matching applied | |
| 7.15 | Prefix search | Search `auth*` | FTS prefix matching | |
| 7.16 | GET form submission | Submit search | URL shows `/search?q=term` | |
| 7.17 | Query preserved in input | Search, then view results | Search input retains query text | |
| 7.18 | Empty query submission | Submit empty search | No results, guidance text shown | |

---

## 8. Web UI - Status

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 8.1 | Spec counts | Create 2 functional, 1 business | Shows Total: 3, Functional: 2, Business: 1 | |
| 8.2 | Task counts by status | Tasks in various statuses | Breakdown table shows each status count | |
| 8.3 | KB counts | Add 3 docs (1 md, 1 html, 1 pdf) | Total: 3, by-type counts, chunk count | |
| 8.4 | Tidy health OK | Recently ran tidy | "OK" in green | |
| 8.5 | Tidy health stale | Tidy older than 7 days | "Stale" in red | |
| 8.6 | Lint issues table | Lint reports issues | Table with severity, category, message | |
| 8.7 | No lint issues | Clean workspace | "No issues found" with checkmark | |
| 8.8 | Trash summary | Items in trash | Count and link to `/trash` | |
| 8.9 | Empty trash summary | No trash | "Empty." with checkmark | |
| 8.10 | Rejected files link | Rejected files exist | Count and link to `/rejected` | |
| 8.11 | No rejected files | None rejected | "None." with checkmark | |
| 8.12 | Empty workspace status | Fresh init | All counts zero, health OK | |

---

## 9. Web UI - Trash

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 9.1 | Empty trash page | No deleted items | "Trash is empty" with check icon | |
| 9.2 | Trash table | Delete spec, task, KB doc | Table shows all 3 with kind, original ID, path, date, deleted_by | |
| 9.3 | Kind badges | Items of different kinds | Chips show spec/task/kb with correct icons | |
| 9.4 | Restore spec | Click restore on spec | Spec recreated, disappears from trash | |
| 9.5 | Restore task | Click restore on task | Task recreated under parent spec | |
| 9.6 | Restore KB | Click restore on KB | Doc and chunks recreated | |
| 9.7 | Restore with ID conflict | Delete spec, create new with same ID slot, restore | New ID allocated, warning shown | |
| 9.8 | Restore task with missing spec | Delete spec and task, don't restore spec, try restore task | Error: parent spec no longer exists | |
| 9.9 | Purge all button visible | Items in trash | "Purge All" button rendered | |
| 9.10 | Purge all hidden | No items | "Purge All" button not rendered | |
| 9.11 | Purge confirmation dialog | Click "Purge All" | Confirmation dialog appears | |
| 9.12 | Confirm purge all | Click confirm in dialog | All trash permanently deleted | |
| 9.13 | Cancel purge | Click cancel in dialog | Trash unchanged | |
| 9.14 | Deleted by shows watcher | Delete file externally | "watcher" in deleted_by column | |
| 9.15 | Deleted by shows cli | Delete via CLI | "cli" in deleted_by column | |

---

## 10. Web UI - Rejected Files

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 10.1 | Empty rejected page | No rejected files | "No rejected files" with check icon | |
| 10.2 | Rejected files table | Hand-create files in specd/ | Table shows path, reason, detected_at | |
| 10.3 | Reason displayed | Non-canonical file created | Reason explains why file was rejected | |
| 10.4 | Detection timestamp | File rejected | ISO timestamp shown | |
| 10.5 | Nav highlights status | Navigate to `/rejected` | "Status" nav item active | |

---

## 11. Web UI - Navigation & Layout

### 11.1 Desktop Sidebar

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 11.1.1 | Sidebar visible on large screens | Open on desktop width (>=993px) | Left sidebar with nav links visible | |
| 11.1.2 | Active nav link highlighted | Navigate to `/specs` | "Specs" link has active class | |
| 11.1.3 | All nav links present | Load any page | Board, Specs, KB, Search, Status, Trash links | |
| 11.1.4 | Nav icons | View sidebar | Each link has correct Material icon | |
| 11.1.5 | Sidebar collapse | Click toggle button | Sidebar collapses, state persisted to localStorage | |
| 11.1.6 | Sidebar expand | Click toggle on collapsed sidebar | Sidebar expands | |
| 11.1.7 | Collapse persisted | Collapse sidebar, reload page | Sidebar still collapsed | |
| 11.1.8 | Aria-expanded attribute | Toggle sidebar | `aria-expanded` toggles between true/false | |

### 11.2 Mobile Navigation

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 11.2.1 | Top bar on small screens | Open on mobile width (<601px) | Top bar with menu button, logo, dark mode toggle | |
| 11.2.2 | Menu button opens dialog | Click hamburger menu | Mobile menu dialog opens | |
| 11.2.3 | Mobile menu links | Open mobile menu | All nav links present | |
| 11.2.4 | Mobile link navigates | Click link in mobile menu | Navigates to page, menu closes | |
| 11.2.5 | Logo links to board | Click specd logo | Navigates to `/` | |

### 11.3 Footer

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 11.3.1 | Footer renders | Scroll to bottom | Footer with about, social, learn, legal sections | |
| 11.3.2 | GitHub link | Click GitHub link | Opens repository | |
| 11.3.3 | Legal links | Click Privacy/Terms | Navigate to correct pages | |

---

## 12. Web UI - Theme & Dark Mode

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 12.1 | Default theme | First visit | Light or dark based on system preference | |
| 12.2 | Toggle to dark mode | Click dark mode button | UI switches to dark theme | |
| 12.3 | Toggle to light mode | Click again | UI switches to light theme | |
| 12.4 | Preference persisted | Toggle to dark, reload | Dark mode preserved via localStorage | |
| 12.5 | Brand color applied | Load any page | Material Design tokens use `#1c4bea` | |
| 12.6 | Dark mode icon correct | In light mode | Shows `dark_mode` icon | |
| 12.7 | Light mode icon correct | In dark mode | Shows `light_mode` icon | |
| 12.8 | Logo switches in dark mode | Toggle mode | Logo SVG changes between light/dark variants | |

---

## 13. Web UI - Error Pages

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 13.1 | 404 page | Navigate to `/nonexistent` | Error page: code 404, "Not Found" message | |
| 13.2 | 404 for bad spec ID | Navigate to `/specs/NOTASPEC` | 404 error page | |
| 13.3 | 404 for bad task ID | Navigate to `/tasks/NOTATASK` | 404 error page | |
| 13.4 | 404 for bad KB ID | Navigate to `/kb/NOTAKB` | 404 error page | |
| 13.5 | Home button on error | View error page | Button navigates to `/` | |
| 13.6 | Error snackbar | Redirect with `?error=msg` | Snackbar appears with error message | |
| 13.7 | Snackbar auto-dismiss | Wait after snackbar appears | Snackbar disappears after ~4 seconds | |
| 13.8 | Error on htmx request | htmx request returns error | Error displayed appropriately | |
| 13.9 | Browser back button | Navigate Board -> Specs -> Spec Detail, press back | Returns to Specs list (htmx history) | |
| 13.10 | Browser forward button | Press back then forward | Returns to Spec Detail | |
| 13.11 | Direct URL access | Paste `/specs/SPEC-1` into address bar | Full page loads (no htmx, base layout rendered) | |

---

## 14. API Endpoints - KB

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 14.1 | GET `/api/kb/{id}` - valid | Request existing doc | JSON with id, title, source_type, page_count, note | |
| 14.2 | GET `/api/kb/{id}` - not found | Request KB-999 | 404 JSON error | |
| 14.3 | GET `/api/kb/{id}/chunks` | Request chunks for doc | JSON array of all chunks with position, text, char_start, char_end, page | |
| 14.4 | GET `/api/kb/{id}/chunk/{pos}` - valid | Request chunk at position 3 | Single chunk JSON | |
| 14.5 | GET `/api/kb/{id}/chunk/{pos}` - not found | Request position 999 | 404 error | |
| 14.6 | GET `/api/kb/{id}/chunk/{pos}` - invalid pos | Request position "abc" | 400 error | |
| 14.7 | GET `/api/kb/{id}/raw` - markdown | Request .md raw | `Content-Type: text/plain; charset=utf-8`, raw markdown bytes | |
| 14.8 | GET `/api/kb/{id}/raw` - text | Request .txt raw | `Content-Type: text/plain; charset=utf-8` | |
| 14.9 | GET `/api/kb/{id}/raw` - html | Request .html raw | `Content-Type: text/html; charset=utf-8`, clean sidecar served | |
| 14.10 | GET `/api/kb/{id}/raw` - pdf | Request .pdf raw | `Content-Type: application/pdf`, raw PDF bytes | |
| 14.11 | Path traversal protection | Manipulate doc_id to escape kb/ | 403 Forbidden | |
| 14.12 | Files constrained to specd/kb/ | Request with valid ID but symlinked path | Path verified within workspace | |
| 14.13 | Raw serves clean HTML | Request HTML raw | Serves `.clean.html` sidecar, never original | |

---

## 15. API Endpoints - Board Drag-and-Drop

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 15.1 | POST `/api/board/move` - valid | `{"task_id":"TASK-1","status":"todo"}` | 200, re-rendered board HTML | |
| 15.2 | POST `/api/board/move` - invalid status | `{"task_id":"TASK-1","status":"invalid"}` | Error response | |
| 15.3 | POST `/api/board/move` - missing task | `{"task_id":"TASK-999","status":"todo"}` | Error response | |
| 15.4 | POST `/api/board/reorder` - valid | `{"task_id":"TASK-1","position":0}` | 200, re-rendered board | |
| 15.5 | POST `/api/board/reorder` - invalid pos | `{"task_id":"TASK-1","position":-1}` | Error or clamped to 0 | |
| 15.6 | JSON Content-Type required | POST with form data | Correct handling or error | |

---

## 16. CLI - Init

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 16.1 | Init empty directory | `specd init /tmp/test` | Creates specd/, .specd/, cache.db, index.md, log.md | |
| 16.2 | Init current directory | `cd /tmp/test && specd init` | Workspace created at CWD | |
| 16.3 | Init already initialized | `specd init` twice | Error: "workspace already initialized" | |
| 16.4 | Init with --force | `specd init --force` on existing | Reinitializes successfully | |
| 16.5 | .gitignore updated | `specd init` in git repo | `.specd/` line appended | |
| 16.6 | .gitignore not duplicated | `specd init --force` | `.specd/` not added twice | |
| 16.7 | User name from git | Git config has user.name | meta.user_name seeded | |
| 16.8 | No git config | No git user.name | meta.user_name empty | |
| 16.9 | JSON output | `specd init --json` | JSON with root and message | |

---

## 17. CLI - Config

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 17.1 | Set user.name | `specd config user.name "John"` | Stored in meta table | |
| 17.2 | Get user.name | `specd config user.name` | Prints current value | |
| 17.3 | JSON output | `specd config user.name --json` | JSON `{"user.name": "John"}` | |
| 17.4 | Unknown key | `specd config foo.bar` | Error or unsupported message | |
| 17.5 | user.name populates created_by | Set user.name "Alice", create spec | Spec JSON shows `created_by: "Alice"` | |
| 17.6 | user.name populates updated_by | Set user.name "Bob", update a spec | Spec JSON shows `updated_by: "Bob"` | |
| 17.7 | Tidy reminder in CLI response | Don't run tidy for 7+ days, run `specd new-spec ...` | Response includes non-null `tidy_reminder` field | |
| 17.8 | Tidy reminder absent when recent | Run tidy, then create spec | Response has `tidy_reminder: null` | |

---

## 18. CLI - Spec Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 18.1 | new-spec happy path | `specd new-spec --title "Auth" --type functional --summary "Auth flow" --body "..."` | SPEC-1 created, JSON with id, path, candidates | |
| 18.2 | new-spec creates directory | Run new-spec | `specd/specs/SPEC-1-auth/spec.md` exists | |
| 18.3 | new-spec updates index.md | Run new-spec | Spec added to index.md | |
| 18.4 | new-spec updates log.md | Run new-spec | Entry appended to log.md | |
| 18.5 | new-spec with --link | `--link SPEC-1 --link SPEC-2` | Links created bidirectionally | |
| 18.6 | new-spec with --cite | `--cite KB-1:3 --cite KB-2:7` | Citations created | |
| 18.7 | new-spec --dry-run | Add `--dry-run` flag | No files or DB changes, preview output | |
| 18.8 | new-spec candidates returned | Create spec similar to existing | Candidates block in response | |
| 18.9 | read spec | `specd read SPEC-1` | JSON with all spec fields | |
| 18.10 | read spec --with-tasks | Add flag | Includes child task list | |
| 18.11 | read spec --with-links | Add flag | Includes linked specs | |
| 18.12 | read spec --with-progress | Add flag | Includes done/active/percent | |
| 18.13 | read spec --with-citations | Add flag | Includes cited chunk text | |
| 18.14 | read nonexistent spec | `specd read SPEC-999` | Error: "spec SPEC-999 not found" | |
| 18.15 | list specs | `specd list specs` | All specs listed | |
| 18.16 | list specs --type | `--type functional` | Only functional specs | |
| 18.17 | list specs --linked-to | `--linked-to SPEC-1` | Specs linked to SPEC-1 | |
| 18.18 | list specs --empty | `--empty` | Only specs with 0 tasks | |
| 18.19 | list specs --limit | `--limit 5` | Max 5 specs returned | |
| 18.20 | update spec | `specd update SPEC-1 --title "New Title"` | Title updated, other fields unchanged | |
| 18.21 | rename spec | `specd rename SPEC-1 --title "New Name"` | Directory renamed, slug updated | |
| 18.22 | delete spec | `specd delete SPEC-1` | Soft-deleted to trash | |
| 18.23 | reorder spec --before | `specd reorder spec SPEC-2 --before SPEC-1` | SPEC-2 positioned before SPEC-1 | |
| 18.24 | reorder spec --after | `specd reorder spec SPEC-1 --after SPEC-3` | Position updated | |
| 18.25 | reorder spec --to | `specd reorder spec SPEC-1 --to 0` | Moved to first position | |
| 18.26 | new-spec --dry-run no side effects | `specd new-spec --dry-run --title "X" --type functional --summary "Y" --body "..."` | No directory created, no DB row, preview returned | |
| 18.27 | new-spec with --link at creation | `specd new-spec ... --link SPEC-1` | Spec created AND link established in one command | |
| 18.28 | new-spec with --cite at creation | `specd new-spec ... --cite KB-1:0` | Spec created AND citation established in one command | |
| 18.29 | read spec combined flags | `specd read SPEC-1 --with-tasks --with-links --with-progress --with-citations` | All enrichments returned in single response | |
| 18.30 | rename spec verifies path change | Rename SPEC-1 to "New Name" | Directory changes from `SPEC-1-old-slug/` to `SPEC-1-new-name/`, task paths inside updated | |
| 18.31 | new-spec index.md content | Create 3 specs, read index.md | All 3 specs listed in index with IDs and titles | |
| 18.32 | new-spec log.md content | Create spec, read log.md | Entry with timestamp, ID, title appended | |
| 18.33 | delete spec removes from index.md | Delete SPEC-2 | SPEC-2 removed from index.md | |
| 18.34 | new-spec slug generation | `--title "OAuth & GitHub 'Login' <Flow>"` | Slug: `oauth-github-login-flow` (special chars to hyphens) | |
| 18.35 | slug truncation at 60 chars | `--title "Very Long Title That Exceeds Sixty Characters And Should Be Truncated Somewhere"` | Slug truncated to max 60 chars, no trailing hyphen | |
| 18.36 | candidates command | `specd candidates SPEC-1 --limit 10` | Returns ranked candidate specs and KB chunks, excludes self and already-linked | |
| 18.37 | candidates excludes linked | Link SPEC-1 to SPEC-2, run candidates for SPEC-1 | SPEC-2 not in results | |
| 18.38 | candidates for task | `specd candidates TASK-1 --limit 10` | Returns candidate tasks and KB chunks | |

---

## 19. CLI - Task Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 19.1 | new-task happy path | `specd new-task --spec-id SPEC-1 --title "Design" --summary "..." --body "..."` | TASK-1 created | |
| 19.2 | new-task default status | Omit --status | Status defaults to "backlog" | |
| 19.3 | new-task with --status | `--status todo` | Task created in todo | |
| 19.4 | new-task missing spec | `--spec-id SPEC-999` | Error: parent spec not found | |
| 19.5 | new-task with --depends-on | `--depends-on TASK-1` | Dependency created | |
| 19.6 | new-task with criteria in body | Body contains `## Acceptance criteria\n- [ ] Item 1` | Criteria parsed and stored | |
| 19.7 | read task | `specd read TASK-1` | JSON with all task fields | |
| 19.8 | read --with-deps | Add flag | Includes blocker list with ready status | |
| 19.9 | read --with-criteria | Add flag | Includes criteria with checked state | |
| 19.10 | list tasks | `specd list tasks` | All tasks listed | |
| 19.11 | list tasks --status | `--status in_progress` | Filtered by status | |
| 19.12 | list tasks --spec-id | `--spec-id SPEC-1` | Only tasks under SPEC-1 | |
| 19.13 | list tasks --created-by | `--created-by "John"` | Filtered by creator | |
| 19.14 | move task | `specd move TASK-1 --status done` | Status changed, frontmatter updated | |
| 19.15 | move task same status | `specd move TASK-1 --status backlog` (already backlog) | No-op, no error | |
| 19.16 | rename task | `specd rename TASK-1 --title "New"` | File renamed, slug updated | |
| 19.17 | delete task | `specd delete TASK-1` | Soft-deleted to trash | |
| 19.18 | reorder task within status | `specd reorder task TASK-2 --before TASK-1` | Position updated within same status | |
| 19.19 | new-task --dry-run | `specd new-task --dry-run --spec-id SPEC-1 --title "X" --summary "Y" --body "..."` | No file created, no DB row, preview returned | |
| 19.20 | new-task with --link at creation | `specd new-task ... --link TASK-1` | Task created AND link established in one command | |
| 19.21 | new-task with --cite at creation | `specd new-task ... --cite KB-1:0` | Task created AND citation established in one command | |
| 19.22 | read task --with-links | `specd read TASK-1 --with-links` | Includes linked tasks with title and status | |
| 19.23 | read task --with-citations | `specd read TASK-1 --with-citations` | Includes cited KB chunks with text preview | |
| 19.24 | read task combined flags | `specd read TASK-1 --with-links --with-deps --with-criteria --with-citations` | All enrichments returned in single response | |
| 19.25 | list tasks --linked-to | `--linked-to TASK-1` | Only tasks linked to TASK-1 | |
| 19.26 | list tasks --depends-on | `--depends-on TASK-1` | Only tasks that depend on TASK-1 | |
| 19.27 | list tasks --limit | `--limit 3` | Max 3 tasks returned | |
| 19.28 | rename task verifies path change | Rename TASK-1 to "New Task" | File changes from `TASK-1-old-slug.md` to `TASK-1-new-task.md` | |
| 19.29 | created_by populated | Config user.name "Alice", create task | Task `created_by` = "Alice" in JSON output | |
| 19.30 | updated_by populated | Config user.name "Bob", update task | Task `updated_by` = "Bob" | |

---

## 20. CLI - Criteria Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 20.1 | criteria list | `specd criteria list TASK-1` | Lists all criteria with position and checked state | |
| 20.2 | criteria add | `specd criteria add TASK-1 "Write tests"` | Criterion appended, markdown updated | |
| 20.3 | criteria check | `specd criteria check TASK-1 1` | `[ ]` changed to `[x]` in markdown and DB | |
| 20.4 | criteria uncheck | `specd criteria uncheck TASK-1 1` | `[x]` changed to `[ ]` | |
| 20.5 | criteria remove | `specd criteria remove TASK-1 2` | Criterion deleted, positions renumbered | |
| 20.6 | check nonexistent position | `specd criteria check TASK-1 99` | Error: criterion not found | |
| 20.7 | idempotent check | Check already-checked criterion | No error, state unchanged | |

---

## 21. CLI - Link Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 21.1 | link specs | `specd link SPEC-1 SPEC-2` | Bidirectional link created | |
| 21.2 | link tasks | `specd link TASK-1 TASK-2` | Bidirectional link created | |
| 21.3 | link multiple targets | `specd link SPEC-1 SPEC-2 SPEC-3` | Two links created | |
| 21.4 | cross-kind link rejected | `specd link SPEC-1 TASK-1` | Error: both must be same kind | |
| 21.5 | link to nonexistent | `specd link SPEC-1 SPEC-999` | Error: target not found | |
| 21.6 | link idempotent | Link same pair twice | No error on second call | |
| 21.7 | self-link rejected | `specd link SPEC-1 SPEC-1` | Error: cannot link to self | |
| 21.8 | unlink | `specd unlink SPEC-1 SPEC-2` | Both directions removed, frontmatter synced | |
| 21.9 | frontmatter updated | After linking | `linked_specs` field in markdown updated | |

---

## 22. CLI - Depend Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 22.1 | depend | `specd depend TASK-2 --on TASK-1` | Dependency created (TASK-1 blocks TASK-2) | |
| 22.2 | depend multiple | `specd depend TASK-3 --on TASK-1 --on TASK-2` | Both blockers added | |
| 22.3 | self-dependency rejected | `specd depend TASK-1 --on TASK-1` | Error: dependency cycle | |
| 22.4 | direct cycle rejected | TASK-1 depends on TASK-2, then `depend TASK-2 --on TASK-1` | Error: dependency cycle detected with path | |
| 22.5 | indirect cycle rejected | A->B->C exists, add C->A | Error: cycle A->B->C->A | |
| 22.6 | undepend | `specd undepend TASK-2 --on TASK-1` | Dependency removed | |
| 22.7 | frontmatter synced | After depend | `depends_on` field updated in markdown | |
| 22.8 | depend nonexistent | `--on TASK-999` | Error: task not found | |

---

## 23. CLI - Cite Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 23.1 | cite spec | `specd cite SPEC-1 KB-1:3` | Citation created | |
| 23.2 | cite task | `specd cite TASK-1 KB-2:5` | Citation created | |
| 23.3 | cite multiple chunks | `specd cite SPEC-1 KB-1:3 KB-1:5 KB-2:1` | All citations created | |
| 23.4 | cite invalid format | `specd cite SPEC-1 KB1-3` | Error: invalid citation format | |
| 23.5 | cite nonexistent KB | `specd cite SPEC-1 KB-999:0` | Error: KB doc not found | |
| 23.6 | cite nonexistent chunk | `specd cite SPEC-1 KB-1:999` | Error: chunk not found | |
| 23.7 | uncite | `specd uncite SPEC-1 KB-1:3` | Citation removed | |
| 23.8 | frontmatter synced | After citing | `cites` field updated with grouped KB refs | |
| 23.9 | idempotent cite | Cite same chunk twice | No error, single entry | |

---

## 24. CLI - KB Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 24.1 | kb add markdown file | `specd kb add /path/to/doc.md` | KB-1 created, chunked, indexed | |
| 24.2 | kb add text file | `specd kb add /path/to/notes.txt` | KB-1 with txt source_type | |
| 24.3 | kb add HTML file | `specd kb add /path/to/page.html` | Clean sidecar created, chunked | |
| 24.4 | kb add PDF file | `specd kb add /path/to/paper.pdf` | Page count recorded, page-aware chunks | |
| 24.5 | kb add with --title | `--title "Custom Title"` | Custom title overrides filename | |
| 24.6 | kb add with --note | `--note "Reference for auth"` | Note stored | |
| 24.7 | kb add nonexistent file | `specd kb add /tmp/nope.md` | Error: file not found | |
| 24.8 | kb add unsupported type | `specd kb add /path/to/file.docx` | Error: unsupported file type | |
| 24.9 | kb list | `specd kb list` | All docs listed with ID, type, title | |
| 24.10 | kb list --source-type | `--source-type pdf` | Only PDFs listed | |
| 24.11 | kb read | `specd kb read KB-1` | All chunks with positions and text | |
| 24.12 | kb read --chunk | `--chunk 3` | Single chunk at position 3 | |
| 24.13 | kb search | `specd kb search "authentication"` | Matching chunks with scores | |
| 24.14 | kb search --limit | `--limit 5` | Max 5 results | |
| 24.15 | kb connections | `specd kb connections KB-1` | TF-IDF connected chunks | |
| 24.16 | kb connections --chunk | `--chunk 3` | Connections for specific chunk | |
| 24.17 | kb rebuild-connections | Run command | All connections recomputed | |
| 24.18 | kb rebuild-connections --threshold | `--threshold 0.5` | Higher threshold = fewer connections | |
| 24.19 | kb rebuild-connections --top-k | `--top-k 5` | Max 5 connections per chunk | |
| 24.20 | kb remove | `specd kb remove KB-1` | Soft-deleted to trash | |
| 24.21 | TF-IDF connections created on add | Add 2 related docs | Connections populated in chunk_connections | |
| 24.22 | Chunking respects paragraph boundaries | Add doc with clear paragraphs | Chunks align to paragraph breaks | |
| 24.23 | Chunk size constraints | Add doc with 2000+ char paragraph | Paragraph split at sentence boundaries | |
| 24.24 | Max 10000 chunks | Add enormous document | Error or capped at 10000 | |
| 24.25 | kb add from URL | `specd kb add https://example.com/doc.html --title "Web Page"` | File fetched, saved locally, chunked, indexed | |
| 24.26 | kb add from URL - invalid URL | `specd kb add https://nonexistent.invalid/x.md` | Clean error, no partial state | |
| 24.27 | kb add from URL - type detection | Add URL returning HTML Content-Type | Source type detected as `html`, clean sidecar created | |

---

## 25. CLI - Search

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 25.1 | search all kinds | `specd search "auth" --kind all` | Results grouped by spec, task, kb | |
| 25.2 | search specs only | `--kind spec` | Only spec results | |
| 25.3 | search tasks only | `--kind task` | Only task results | |
| 25.4 | search kb only | `--kind kb` | Only KB chunk results | |
| 25.5 | BM25 primary search | Search for stemmed term | BM25 matches with match_type "bm25" | |
| 25.6 | Trigram fallback | Search for substring | Trigram results tagged as "trigram" | |
| 25.7 | Hybrid merge | BM25 returns <3 results | Trigram results appended below BM25 | |
| 25.8 | Limit enforcement | `--limit 3` | Max 3 results per kind | |
| 25.9 | FTS operators | `specd search "auth AND jwt"` | AND operator applied | |
| 25.10 | Phrase search | `specd search '"exact match"'` | Phrase matching | |
| 25.11 | Empty query | `specd search ""` | No results, no error | |
| 25.12 | JSON output | `--json` | Structured JSON response | |
| 25.13 | Score ordering | Search with multiple matches | Results ordered by relevance score descending | |

---

## 26. CLI - Next

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 26.1 | Next returns todo tasks | Create tasks in various statuses | Only "todo" tasks returned | |
| 26.2 | Ready before not-ready | Task A (no deps) and Task B (blocked) | Task A first | |
| 26.3 | Partially done before fresh | Task A (1/3 criteria checked) vs Task B (0/3) | Task A first | |
| 26.4 | Higher progress first | Task A (80%) vs Task B (50%) | Task A first | |
| 26.5 | Position tiebreaker | Same progress, same readiness | Lower position first | |
| 26.6 | --limit flag | `--limit 3` | Max 3 tasks | |
| 26.7 | --spec-id filter | `--spec-id SPEC-1` | Only SPEC-1's tasks | |
| 26.8 | Blocked-by list | Task with unresolved deps | `blocked_by` includes blocker IDs | |
| 26.9 | Ready when deps done | Blocker in "done" status | Dependent task marked ready | |
| 26.10 | Ready when deps cancelled | Blocker in "cancelled" | Dependent task marked ready | |
| 26.11 | Ready when deps wontfix | Blocker in "wontfix" | Dependent task marked ready | |
| 26.12 | Dependency cycle error | Circular dependency exists | Error with cycle path | |
| 26.13 | No todo tasks | All tasks in other statuses | Empty list, no error | |
| 26.14 | JSON output | `--json` | Structured response with ready, partially_done, criteria_progress, blocked_by | |

---

## 27. CLI - Maintenance

### 27.1 Lint

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.1.1 | Clean workspace | Lint with no issues | "0 errors, 0 warnings" | |
| 27.1.2 | Dangling spec link | Link to deleted spec | Error: dangling spec link | |
| 27.1.3 | Dangling task link | Link to deleted task | Error: dangling task link | |
| 27.1.4 | Dangling dependency | Depend on deleted task | Error: dangling dependency | |
| 27.1.5 | Dangling citation | Cite deleted KB chunk | Error: dangling citation | |
| 27.1.6 | Orphan spec | Spec with no links and no tasks | Warning: orphan spec | |
| 27.1.7 | Orphan task | Task whose parent spec was removed | Error: orphan task | |
| 27.1.8 | System field drift | Manually edit linked_specs in file | Warning: field drift | |
| 27.1.9 | Stale tidy | last_tidy_at > 7 days | Warning: stale tidy | |
| 27.1.10 | Missing summary | Spec with empty summary | Warning: missing summary | |
| 27.1.11 | Trivial summary | Single-word summary | Warning: trivial summary | |
| 27.1.12 | Dependency cycle | Circular deps | Error with cycle path | |
| 27.1.13 | Rejected files | Files in rejected_files table | Warning per file | |
| 27.1.14 | KB integrity - missing file | Delete KB source file from disk | Error: KB file missing | |
| 27.1.15 | KB integrity - hash mismatch | Modify KB file externally | Warning: hash mismatch | |
| 27.1.16 | Citation chunk missing | Cite chunk, then re-chunk (different positions) | Error: citation points to nonexistent chunk | |
| 27.1.17 | JSON output | `specd lint --json` | Structured JSON with issues array | |

### 27.2 Tidy

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.2.1 | Tidy runs lint | `specd tidy` | Lint results output | |
| 27.2.2 | Tidy updates timestamp | `specd tidy` | last_tidy_at updated to now | |
| 27.2.3 | Tidy reminder cleared | After running tidy | tidy_reminder returns null | |

### 27.3 Rebuild

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.3.1 | Rebuild empty workspace | `specd rebuild` | Fresh DB, 0 specs/tasks/kb | |
| 27.3.2 | Rebuild with data | Create specs/tasks/kb, then rebuild | All data reconstructed from markdown | |
| 27.3.3 | Counter preservation | Rebuild after SPEC-5 created | next_spec_id = 6 | |
| 27.3.4 | User name preserved | Config user.name before rebuild | Name still set after | |
| 27.3.5 | Links reconstructed | Linked specs, rebuild | Links restored from frontmatter | |
| 27.3.6 | Dependencies reconstructed | Tasks with deps, rebuild | Dependencies restored | |
| 27.3.7 | Citations reconstructed | Cited chunks, rebuild | Citations restored | |
| 27.3.8 | Criteria reconstructed | Tasks with criteria, rebuild | Criteria positions and checked states restored | |
| 27.3.9 | KB chunks reconstructed | KB docs, rebuild | All chunks recreated | |
| 27.3.10 | TF-IDF connections rebuilt | KB docs with connections, rebuild | Connections recomputed | |
| 27.3.11 | Rejected files reported | Non-canonical files in specd/ | Listed in rebuild output | |
| 27.3.12 | Canonical files ingested | Properly named files | Ingested during rebuild | |
| 27.3.13 | --force flag | `specd rebuild --force` | Succeeds even with existing cache | |

### 27.4 Status

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.4.1 | status basic | `specd status` | Counts for specs, tasks, KB, trash | |
| 27.4.2 | status --detailed | `specd status --detailed` | Includes lint summary | |
| 27.4.3 | JSON output | `specd status --json` | Structured JSON | |

### 27.5 Merge-Fixup

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 27.5.1 | No duplicates | Clean workspace | Empty result | |
| 27.5.2 | Duplicate spec IDs | Two SPEC-1 directories | Detected, second renumbered | |
| 27.5.3 | Duplicate task IDs | Two TASK-1 files in different spec dirs | Detected, one renumbered | |
| 27.5.4 | Duplicate KB IDs | Two KB-1 files | Detected, one renumbered | |
| 27.5.5 | Rebuild after fixup | Fixup renumbers items | Full rebuild triggered | |
| 27.5.6 | Clean sidecar renamed | Duplicate HTML KB | .clean.html also renamed | |
| 27.5.7 | Counter updated | After renumbering | next_*_id reflects new max | |

---

## 28. CLI - Trash Commands

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 28.1 | trash list | `specd trash list` | All trashed items | |
| 28.2 | trash list --kind | `--kind spec` | Only spec trash | |
| 28.3 | trash list --older-than | `--older-than 7d` | Only items older than 7 days | |
| 28.4 | trash restore | `specd trash restore 1` | Item restored, file recreated | |
| 28.5 | trash restore with new ID | Restore when original ID reused | New ID allocated, warning printed | |
| 28.6 | trash restore task orphan | Restore task whose spec was deleted | Error: parent spec no longer exists | |
| 28.7 | trash purge | `specd trash purge --older-than 30d` | Items older than 30d permanently deleted | |
| 28.8 | trash purge-all | `specd trash purge-all` | All trash permanently deleted | |
| 28.9 | trash purge returns count | Run purge | Outputs number of purged items | |
| 28.10 | Duration parsing: days | `--older-than 30d` | Parsed correctly | |
| 28.11 | Duration parsing: hours | `--older-than 24h` | Parsed correctly | |
| 28.12 | Duration parsing: minutes | `--older-than 60m` | Parsed correctly | |
| 28.13 | Duration parsing: invalid | `--older-than xyz` | Error: invalid duration | |

---

## 29. CLI - Serve & Watch

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 29.1 | serve starts server | `specd serve` | HTTP server on port 7823 | |
| 29.2 | serve custom port | `specd serve --port 9000` | Server on port 9000 | |
| 29.3 | serve starts watcher | `specd serve` | File watcher running alongside server | |
| 29.4 | serve responds to requests | Curl `http://localhost:7823/` | HTML response | |
| 29.5 | watch standalone | `specd watch` | Watcher running without HTTP server | |
| 29.6 | Ctrl+C stops serve | Send SIGINT | Graceful shutdown | |
| 29.7 | Ctrl+C stops watch | Send SIGINT | Watcher stops | |
| 29.8 | serve --open flag | `specd serve --open` | Server starts and browser auto-opens | |
| 29.9 | Concurrent CLI lock timeout | Start serve, run `specd new-spec` in another terminal during a write | Second process waits up to 5s, then errors if still locked | |

---

## 30. Watcher - Sync Behavior

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 30.1 | Edit spec title in editor | Modify title in frontmatter, save | SQLite title updated within ~1s | |
| 30.2 | Edit spec type | Change type field | SQLite type updated | |
| 30.3 | Edit spec summary | Change summary | SQLite summary updated | |
| 30.4 | Edit spec body | Add content to body | SQLite body updated | |
| 30.5 | Edit task title | Modify task frontmatter | SQLite updated | |
| 30.6 | Edit task status | Change status in frontmatter | SQLite status updated | |
| 30.7 | Check acceptance criterion | Change `[ ]` to `[x]` in editor | task_criteria.checked updated | |
| 30.8 | Uncheck criterion | Change `[x]` to `[ ]` | task_criteria.checked updated | |
| 30.9 | Add criterion manually | Add `- [ ] New item` to criteria section | New row in task_criteria | |
| 30.10 | CLI write not double-processed | Create spec via CLI | Watcher sees matching hash, skips | |
| 30.11 | FTS index updated | Edit spec title | FTS5 search finds new title | |
| 30.12 | Trigram index updated | Edit spec body | Trigram search finds new content | |
| 30.13 | content_hash updated | External edit | content_hash recalculated | |
| 30.14 | updated_at updated | External edit | Timestamp refreshed | |
| 30.15 | System fields preserved | Edit linked_specs manually | SQLite version wins (log drift warning) | |
| 30.16 | index.md changes ignored | Edit index.md | Watcher skips auto-maintained file | |
| 30.17 | log.md changes ignored | Edit log.md | Watcher skips auto-maintained file | |

---

## 31. Watcher - Deletion Handling

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 31.1 | Delete spec directory | `rm -rf specd/specs/SPEC-1-auth/` | Spec and all tasks soft-deleted to trash | |
| 31.2 | Delete task file | `rm specd/specs/SPEC-1-auth/TASK-1-design.md` | Task soft-deleted to trash | |
| 31.3 | Delete KB file | `rm specd/kb/KB-1-doc.md` | KB doc soft-deleted to trash | |
| 31.4 | Trash entry created | Delete file externally | trash row with deleted_by='watcher' | |
| 31.5 | Citations cleaned | Delete spec/task with citations | Citations deleted | |
| 31.6 | Clean sidecar ignored | Delete .clean.html file | Watcher skips sidecar | |

---

## 32. Watcher - Rejection Logic

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 32.1 | Non-canonical spec dir | Create `specd/specs/my-spec/spec.md` | Inserted into rejected_files | |
| 32.2 | Non-canonical task file | Create `specd/specs/SPEC-1-auth/my-task.md` | Rejected | |
| 32.3 | Non-canonical KB file | Create `specd/kb/my-doc.md` | Rejected | |
| 32.4 | File outside specd/ | Create file in project root | Watcher ignores (filtered by path) | |
| 32.5 | Rejected file reason | Check rejected entry | Reason explains why file was rejected | |
| 32.6 | File left on disk | Non-canonical file created | File not deleted, just recorded | |

---

## 33. Watcher - Debouncing

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 33.1 | Single save processed once | Save file once | One sync operation | |
| 33.2 | Rapid saves debounced | Save same file 5 times in 100ms | Only final state processed | |
| 33.3 | Different files processed independently | Save two different files simultaneously | Both processed | |
| 33.4 | 200ms debounce window | Save, wait 200ms, save again | Both edits processed as separate events | |
| 33.5 | Timer cancelled on new event | Start timer, save again before 200ms | First timer cancelled, new timer starts | |
| 33.6 | Shutdown flushes pending | Stop watcher with pending timer | Pending events flushed before exit | |

---

## 34. Integration - Spec Lifecycle

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 34.1 | Create, read, update, delete | Full CRUD cycle via CLI | All operations succeed, data consistent | |
| 34.2 | Create via CLI, view in UI | `specd new-spec`, open browser | Spec appears on specs page | |
| 34.3 | Create via UI, read via CLI | Create in browser, `specd read` | CLI returns spec data | |
| 34.4 | Edit in editor, see in UI | Edit spec.md, reload browser | Updated content displayed | |
| 34.5 | Delete via UI, verify files | Delete spec in browser | Directory removed, trash entry exists | |
| 34.6 | Rename cascades tasks | Rename spec | Task paths updated | |
| 34.7 | Delete cascades everything | Delete spec with tasks, links, citations | All related data cleaned up | |
| 34.8 | Tidy reminder on responses | Don't run tidy for 7+ days | new-spec response includes tidy_reminder | |

---

## 35. Integration - Task Lifecycle

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 35.1 | Create, move through statuses, complete | Create backlog -> todo -> in_progress -> done | All transitions work | |
| 35.2 | Full kanban workflow | Create task, drag through columns in UI | Statuses update, positions correct | |
| 35.3 | Criteria workflow | Add criteria, check items, verify completion | Progress updates in real-time | |
| 35.4 | Block and unblock | Add dependency, complete blocker | Blocked task becomes ready | |
| 35.5 | Task with citations round-trip | Cite KB chunk, read task | Citations visible in CLI and UI | |
| 35.6 | Multiple tasks under one spec | Create 10 tasks under SPEC-1 | All shown in spec detail, board | |

---

## 36. Integration - KB Lifecycle

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 36.1 | Add doc, cite from spec, view in reader | Full flow | Citation links to reader, chunk highlighted | |
| 36.2 | Add 3 related docs | Add docs with overlapping terminology | TF-IDF connections created between them | |
| 36.3 | Remove doc with citations | Delete cited KB doc | Citations cleaned from citing specs/tasks | |
| 36.4 | Rebuild connections after bulk add | Add 5 docs, rebuild-connections | All cross-doc connections recomputed | |
| 36.5 | KB search after add | Add doc, search for its content | Found in search results | |

---

## 37. Integration - Citation Flow

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 37.1 | Cite from spec detail | View spec, see citation card | Source type icon, title, preview text, page number | |
| 37.2 | View in source from citation | Click "View in source" | KB reader opens at correct chunk | |
| 37.3 | Multiple citations per entity | Spec cites 3 different KB chunks | All 3 shown as cards | |
| 37.4 | Citation survives spec update | Update spec title | Citations unchanged | |
| 37.5 | Citation removed on uncite | Uncite via CLI | Citation card disappears from UI | |
| 37.6 | Citation cascade on KB remove | Remove KB doc | All citations referencing it removed | |

---

## 38. Integration - Dependency Graph

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 38.1 | Chain: A -> B -> C | Create 3 tasks, chain dependencies | Next shows C only when A and B done | |
| 38.2 | Diamond: A,B -> C | C depends on both A and B | C ready only when both A and B done | |
| 38.3 | Partial completion | A done, B in_progress | C still blocked | |
| 38.4 | Cancel unblocks | A cancelled | Dependents become ready | |
| 38.5 | Wontfix unblocks | A set to wontfix | Dependents become ready | |
| 38.6 | Cycle detection on complex graph | Attempt to create cycle in 5-node graph | Error with full cycle path | |
| 38.7 | Board shows blocked icon | Task has unresolved dep | Warning icon on kanban card | |
| 38.8 | Next ordering with deps | Multiple tasks with varying readiness | Ready tasks first, then progress, then position | |

---

## 39. Integration - Search Across Entities

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 39.1 | Term found in spec, task, and KB | Create all 3 with shared term | Search returns results in all 3 groups | |
| 39.2 | Deleted entities not in search | Delete a spec, search for its title | Not found in results | |
| 39.3 | Updated content searchable | Update spec body, search new term | Found in results | |
| 39.4 | Trigram finds partial match | Search for substring of a title | Trigram result returned | |
| 39.5 | Porter stemming works | Search "running" matches "runs" | BM25 with stemming finds match | |

---

## 40. Integration - Rebuild & Merge-Fixup

| # | Scenario | Steps | Expected | Result |
|---|----------|-------|----------|--------|
| 40.1 | Rebuild restores full state | Create data, delete cache.db, rebuild | All specs, tasks, KB, links, deps, citations restored | |
| 40.2 | Rebuild + search works | Rebuild, then search | FTS and trigram indexes functional | |
| 40.3 | Rebuild + next works | Rebuild, then `specd next` | Correct ordering with dependencies | |
| 40.4 | Merge-fixup + rebuild | Create duplicates, fixup | Clean workspace after fixup | |
| 40.5 | Merge-fixup preserves non-conflicting | 3 specs, only 1 duplicate | Other 2 specs unchanged | |

---

## 41. Non-Functional - Performance

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 41.1 | Board load with 100 specs, 1000 tasks | Page renders in <500ms | |
| 41.2 | Search response time | Results returned in <100ms for 1000-item workspace | |
| 41.3 | Next command response | Ready tasks returned in <50ms | |
| 41.4 | Lint completion time | Completes in <1s for 100 specs + 1000 tasks | |
| 41.5 | Rebuild time | Completes in reasonable time for 100 specs, 1000 tasks, 50 KB docs | |
| 41.6 | Watcher sync latency | File change reflected in SQLite within 1s | |
| 41.7 | Drag-and-drop responsiveness | Board re-renders in <300ms after drop | |
| 41.8 | KB reader markdown render | Large markdown doc renders in <500ms | |
| 41.9 | KB reader PDF render (100 pages) | All pages render, navigation smooth | |
| 41.10 | TF-IDF computation | Connections computed in <5s for 50 docs | |
| 41.11 | Static asset load time | CSS + JS bundle loads in <200ms (all local) | |
| 41.12 | Concurrent CLI invocations | Second invocation waits up to 5s on lock | |

---

## 42. Non-Functional - Security

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 42.1 | XSS in spec title | HTML entities escaped in rendered output | |
| 42.2 | XSS in task body | Markdown rendered safely (no raw HTML injection) | |
| 42.3 | XSS in KB doc title | Title HTML-escaped in templates | |
| 42.4 | XSS in search query | Query echoed safely in results page | |
| 42.5 | XSS in criterion text | Text rendered as plain text, not HTML | |
| 42.6 | XSS in error messages | Error query param HTML-escaped in snackbar | |
| 42.7 | HTML sanitization at ingest | `<script>` tags removed by bluemonday | |
| 42.8 | HTML iframe sandboxed | `sandbox="allow-same-origin"` only, no `allow-scripts` | |
| 42.9 | PDF.js JavaScript disabled | Embedded PDF JS not executed | |
| 42.10 | Path traversal on `/api/kb/{id}/raw` | Manipulated paths blocked (403) | |
| 42.11 | Path constrained to specd/kb/ | Symlink escaping blocked | |
| 42.12 | SQL injection via search | Special characters in search query | FTS query properly escaped | |
| 42.13 | SQL injection via ID parameters | `'; DROP TABLE specs;--` as ID | Parameterized queries prevent injection | |
| 42.14 | CSRF on mutation endpoints | POST without proper origin | Server-side validation (localhost only) | |
| 42.15 | File upload size limits | Upload very large file | Handled gracefully (no OOM) | |
| 42.16 | Lock file timeout | Concurrent writers | Second writer gets clean error after 5s | |
| 42.17 | No external network requests | Monitor network during full UI usage | Zero outbound requests | |
| 42.18 | Content-Type headers correct | Check all API responses | Correct MIME types set | |
| 42.19 | No sensitive data in error messages | Trigger various errors | No file paths or SQL exposed to client | |

---

## 43. Non-Functional - Accessibility

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 43.1 | Semantic HTML structure | Pages use proper heading hierarchy (h1->h2->h3) | |
| 43.2 | ARIA labels on interactive elements | Buttons, toggles, dialogs have aria-label | |
| 43.3 | ARIA-expanded on sidebar toggle | Toggle updates aria-expanded attribute | |
| 43.4 | ARIA-pressed on highlight toggle | Button reflects current state | |
| 43.5 | Dialog aria-label | All dialogs have descriptive aria-label | |
| 43.6 | Form labels associated with inputs | All form fields have proper `<label>` or aria-label | |
| 43.7 | Error alerts with role="alert" | Form validation errors have `role="alert"` | |
| 43.8 | Snackbar with role="alert" | Error snackbar has `role="alert"` | |
| 43.9 | Skip to main content | Tab from top of page | Focus reaches main content efficiently | |
| 43.10 | Keyboard-navigable nav | Tab through navigation | All nav links focusable | |
| 43.11 | Keyboard-navigable forms | Tab through form fields | All fields reachable | |
| 43.12 | Keyboard-navigable KB reader | Arrow keys for chunks | Navigation works without mouse | |
| 43.13 | Color contrast in light mode | Check text contrast ratios | Meets WCAG AA (4.5:1 for text) | |
| 43.14 | Color contrast in dark mode | Check dark theme | Meets WCAG AA | |
| 43.15 | Focus indicators visible | Tab through UI | Focus ring visible on all interactive elements | |
| 43.16 | Screen reader for kanban cards | Use screen reader | Card content readable (title, spec, progress) | |
| 43.17 | Drag-and-drop alternative | Keyboard-only status change | Status dropdown on task detail as alternative | |
| 43.18 | Image alt text | Check all images/icons | Decorative icons aria-hidden, meaningful ones have alt | |
| 43.19 | Responsive text sizing | Zoom to 200% | Content readable, no overflow | |
| 43.20 | Mobile touch targets | Check button sizes on mobile | Minimum 44x44px touch targets | |

---

## 44. Non-Functional - Reliability & Data Integrity

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 44.1 | DB and markdown in sync | After any mutation, DB matches markdown frontmatter | |
| 44.2 | Transactional writes | Spec creation fails mid-write | No partial state (no orphan directory without DB row) | |
| 44.3 | Lock prevents corruption | Two CLI commands simultaneously | Second waits or errors cleanly | |
| 44.4 | Graceful shutdown | Kill server during write | No corrupt DB (WAL journal recovery) | |
| 44.5 | Rebuild recovers from corrupt DB | Delete cache.db, rebuild | Full recovery from markdown | |
| 44.6 | Content hash prevents double-processing | CLI write followed by watcher event | Watcher skips (hashes match) | |
| 44.7 | Cascade deletes complete | Delete spec with deep graph | All tasks, links, deps, citations, criteria removed | |
| 44.8 | FTS indexes consistent | After mutations | Search results match actual data | |
| 44.9 | Trigram indexes consistent | After mutations | Substring search finds expected results | |
| 44.10 | Counter monotonically increases | Create 5 specs, delete 3, create 2 more | IDs never reused (SPEC-6, SPEC-7) | |
| 44.11 | Trash preserves full state | Restore trashed spec | Title, type, body, summary all preserved | |
| 44.12 | KB clean sidecar integrity | Add HTML, verify sidecar | Sidecar sanitized by bluemonday, scripts stripped | |
| 44.13 | PDF chunk page tracking | Add multi-page PDF | Each chunk has correct page number | |
| 44.14 | Criteria positions contiguous | Remove middle criterion | Remaining criteria renumbered 1, 2, 3... | |
| 44.15 | Bidirectional links consistent | Link A->B | Both A->B and B->A exist in spec_links | |
| 44.16 | Dependency directed correctly | Depend TASK-2 on TASK-1 | blocker_task=TASK-1, blocked_task=TASK-2 | |
| 44.17 | Workspace path with spaces | `specd init "/tmp/my workspace"` | Init succeeds, all operations work with quoted paths | |
| 44.18 | Symlinks inside workspace | Symlink within specd/ pointing to another file inside specd/ | Followed and processed normally | |
| 44.19 | Symlinks outside workspace | Symlink inside specd/ pointing to /etc/passwd | Ignored with warning, not followed | |
| 44.20 | Large KB file rejection | Add file that would produce >10,000 chunks | Clean error, no partial KB doc left | |
| 44.21 | Slug uniqueness | Create two specs with similar titles | Both get unique directory names | |
| 44.22 | Title with unicode characters | Create spec with title "Authentifizierung und Sicherheit" | Title preserved, slug generated correctly | |

---

## 45. Non-Functional - Responsiveness & Layout

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 45.1 | Desktop layout (>=993px) | Sidebar visible, full-width content | |
| 45.2 | Tablet layout (601-992px) | Top bar, content fills width | |
| 45.3 | Mobile layout (<601px) | Top bar with hamburger, stacked content | |
| 45.4 | Kanban columns scroll | 8 columns on small screen | Horizontal scroll or responsive reflow | |
| 45.5 | Long spec title wraps | 200-char title | Text wraps, no overflow | |
| 45.6 | Long KB document readable | Very wide HTML content | Contained within reader bounds | |
| 45.7 | Dialog responsive | Open dialog on mobile | Dialog fits viewport, scrollable if needed | |
| 45.8 | Form fields full-width on mobile | Open form on mobile | Inputs span full width | |
| 45.9 | Table scrollable on mobile | Status or trash table with many columns | Horizontal scroll on overflow | |
| 45.10 | Footer doesn't overlap content | Short content page | Footer at bottom, not overlapping | |
| 45.11 | KB reader sidebar responsive | Toggle sidebar on mobile | Sidebar overlays content or collapses | |

---

## 46. Non-Functional - Offline & Embedded Assets

| # | Scenario | Expected | Result |
|---|----------|----------|--------|
| 46.1 | No CDN requests | Load all pages with network tab open | Zero external requests | |
| 46.2 | BeerCSS loads from /vendor/ | Check CSS link | Local `beer.min.css` served | |
| 46.3 | htmx loads from /assets/ | Check script src | Local `htmx.min.js` | |
| 46.4 | marked.js loads locally | Check script src | Local `marked.min.js` | |
| 46.5 | PDF.js loads locally | Open PDF in reader | Worker loaded from `/assets/pdf.worker.min.mjs` | |
| 46.6 | Fonts embedded | Check font requests | No Google Fonts or external font CDN calls | |
| 46.7 | Material icons local | Check icon rendering | Icons render without network | |
| 46.8 | Theme JS local | Check BeerCSS + material-dynamic-colors | Both load from embedded assets | |
| 46.9 | App works with network disabled | Disconnect network, use app | All features functional | |
| 46.10 | Binary is self-contained | Run binary on fresh machine | No runtime dependencies needed | |
| 46.11 | CSS bundle served correctly | Check response headers | Correct Content-Type and caching headers | |
| 46.12 | Favicon served | Check browser tab | Favicon displayed | |
