---
title: GraphQL Gateway
summary: Unified API gateway for all microservices
type: functional
---

## Overview

Build a GraphQL gateway that aggregates data from multiple backend microservices into a single query endpoint.

## Requirements

- Schema stitching across order, inventory, and user services
- Dataloader batching to avoid N+1 queries
- Rate limiting per client API key
- Authentication via JWT forwarded from the edge proxy
