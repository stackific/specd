---
title: API Schema Registry
summary: Central registry for GraphQL gateway schemas and versioning
type: functional
---

## Overview

Maintain a central registry where all service schemas are published and versioned. The frontend team consumes the merged schema from this registry.

## Requirements

- Schema upload via CLI or CI pipeline
- Breaking change detection on PR
- Version history with rollback support
- Webhook notifications on schema changes
