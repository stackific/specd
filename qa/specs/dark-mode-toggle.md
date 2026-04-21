---
title: Dark Mode Toggle
summary: Allow users to switch between light and dark color themes
type: functional
---

## Overview

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
- No white flash on page load in dark mode
