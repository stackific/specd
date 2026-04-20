---
title: CSV Data Import
summary: Import bulk data from CSV files with validation and error reporting
type: functional
---

## Overview

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
- Progress bar updates during import
