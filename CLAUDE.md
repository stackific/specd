Read AGENTS.md

# Rules

- When reference files exist (e.g. `ref/`), copy them directly instead of rewriting from scratch. Do not "adapt" or "simplify" — use the original as-is and only make the specific changes requested.
- Before modifying any file, read how it is actually used in context (imports, consumers, layout). Do not assume.
- Do not empty, gut, or zero out config values (like `social: {}`) unless explicitly asked to. Preserve existing data.
- Do not drop CSS files, imports, or structural elements from the original. If unsure whether something is needed, check the ref or ask.
- When something looks wrong, diff against the reference before guessing at a fix.
- Always use 2 spaces as indentation
