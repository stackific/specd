# How Search Works in specd

specd uses a hybrid search strategy combining two complementary techniques to find related content across specs, tasks, and knowledge base chunks.

## The Two Search Engines

### BM25 (Primary)

BM25 is a proven text ranking algorithm used by search engines. specd uses it through SQLite's FTS5 extension with porter stemming enabled.

When a spec is created, its title, summary, and body are automatically indexed into an FTS5 virtual table (via database triggers). The porter stemmer normalizes words to their root form — "authentication", "authenticating", and "authenticated" all match each other.

A search query like "user authentication session" is broken into individual word tokens, each quoted for safety: `"user" "authentication" "session"`. FTS5 requires all tokens to appear in the document (implicit AND). Results are ranked by BM25 relevance score — documents where the search terms appear more frequently and more prominently score higher.

BM25 is fast and precise. Its weakness is that it only matches whole word tokens. A search for "auth" will not find documents containing "authentication" because they are different tokens after stemming.

### Trigram (Fallback)

Trigram search is a substring matching technique. It splits text into overlapping 3-character windows and matches any document containing the query as an exact substring.

specd maintains a separate FTS5 virtual table using the trigram tokenizer. The same content (title + summary + body) is indexed here via triggers. A search for "auth" will find any document containing that substring anywhere — "authentication", "unauthorized", "OAuth".

Trigram is slower than BM25 and doesn't produce relevance scores, but it catches matches that BM25 misses: partial words, technical identifiers with special characters, and exact phrases.

## How They Work Together

When you search, specd runs BM25 first. If BM25 returns 3 or more results for a given content type, those results are returned as-is, ranked by score.

If BM25 returns fewer than 3 results, trigram search runs as a supplement. Trigram results are appended after the BM25 results (which have scores) and are deduplicated — if the same document was already found by BM25, the trigram hit is skipped.

There is one exception: if the search query contains special characters (hyphens, slashes, dots), trigram always runs alongside BM25 regardless of how many BM25 results were found. This is because BM25 strips special characters during tokenization — a search for "/etc/specd/config.yaml" would lose the path structure entirely. Trigram preserves it.

## What Gets Searched

specd searches three kinds of content:

- **Specs** — indexed by title, summary, and body via `specs_fts`
- **Tasks** — indexed by title, summary, and body via `tasks_fts`
- **KB chunks** — indexed by text content via `kb_chunks_fts`

Enum fields like `type` (on specs) and `status` (on tasks) are deliberately excluded from the FTS indexes. They are short slugs like "functional" or "backlog" — indexing them would add noise without improving search relevance. To filter by type or status, use SQL WHERE clauses on the base table, not full-text search.

All three also have entries in a shared `search_trigram` table for substring matching. This table stores concatenated text (title + summary + body) per record, keyed by kind and ID.

The caller can search all three kinds at once or filter to a single kind.

## Query Sanitization

User input never reaches FTS5 directly. Two sanitization functions transform the raw query:

For BM25, only alphanumeric word tokens are extracted and each is individually quoted. This prevents FTS5 syntax errors and eliminates injection — a title containing " AND " or " NOT " is treated as regular words, not operators.

For trigram, the raw query is wrapped in double quotes for exact phrase matching. Internal double quotes are escaped. Queries shorter than 3 characters are rejected since the trigram tokenizer cannot produce any trigrams from them.

## Deduplication and Exclusion

Every search accepts an exclude ID. When creating a new spec, the search excludes that spec's own ID so it doesn't appear in its own "related content" list. This exclusion applies to both BM25 and trigram paths.

A seen map tracks which IDs have already been added to results. When trigram runs as a fallback, any ID already found by BM25 is skipped. This prevents the same document from appearing twice with different match types.

## Scoring and Column Weights

BM25 results carry a relevance score. SQLite's `bm25()` function returns negative values where more negative means more relevant. specd flips the sign so that higher scores mean better matches — this is what consumers see in the JSON output.

For specs and tasks, the FTS5 columns are indexed in this order: title, summary, body. The BM25 scorer applies per-column weights to prioritize where a match occurs:

- **Title**: weight 10 — a match in the title is the strongest signal
- **Summary**: weight 5 — a match in the summary is moderately strong
- **Body**: weight 1 — a match deep in the body is the weakest signal

This means a spec titled "User Authentication" will rank above a spec that merely mentions "authentication" once in a long body, even if both contain the search term. The weights are hardcoded in the BM25 SQL queries.

KB chunks have a single text column, so no column weighting applies.

Trigram results have a score of zero. They are useful for catching matches that BM25 missed, but they don't indicate relative relevance among themselves.

## Where Search Is Used

Currently, search runs automatically when a new spec is created via `specd new-spec`. The command searches for existing specs and KB chunks related to the new spec's title and summary. The results are included in the JSON output so the AI skill can decide which existing content to link.

The number of results returned per kind is configurable per project via `top_search_results` in `.specd.json` (default: 5).

## Index Maintenance

All FTS indexes are maintained automatically by SQLite triggers. When a spec, task, or KB chunk is inserted, updated, or deleted, the corresponding FTS5 and trigram entries are updated in the same transaction. There is no manual reindexing step.
