---
title: Service Communication Layer
summary: Internal RPC framework for backend services
type: functional
---

## Overview

Design the inter-service communication layer using gRPC with protobuf schemas.

## Requirements

- All services register with the service mesh
- Health checks and circuit breakers on every connection
- The order service must expose a GraphQL gateway compatible schema for the frontend team
- Retry with exponential backoff on transient failures
