---
title: User Authentication
summary: Implement OAuth2 login with Google and GitHub providers
type: functional
---

## Overview

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
- Existing users are matched by email
