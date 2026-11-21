# Recommendation Data — Handoff

Owner: recommendation service. The website team owns the schema and the storefront; this document defines what you import, what you produce, and the contract between us.

## 1. Dataset

**Amazon Reviews'23** (UCSD), `Books` category — https://amazon-reviews-2023.github.io/

| | Users | Items | Ratings |
|---|---|---|---|
| Books | 10.3M | 4.4M | 29.5M |

Two files:

- `meta_Books.jsonl` — item metadata
- `Books.jsonl` — user reviews

### Fields — item metadata

| Field | Type | Notes |
|---|---|---|
| `parent_asin` | str | Product id, groups variants of the same title. **This is our item key** |
| `title` | str | Book title |
| `main_category` | str | Top-level category |
| `categories` | list | Hierarchical category path |
| `average_rating` | float | Aggregate shown on the product page |
| `rating_number` | int | Number of ratings |
| `price` | float | USD at crawl time, often missing |
| `features` | list | Bullet points |
| `description` | list | Paragraphs |
| `images` | list | `thumb` / `large` / `hi_res` variants |
| `videos` | list | Title + URL |
| `store` | str | Publisher or seller |
| `details` | dict | Free-form: Author, ISBN, Publisher, publication date, language |
| `bought_together` | list | Bundles; usually null |

### Fields — user reviews

| Field | Type | Notes |
|---|---|---|
| `user_id` | str | Reviewer id |
| `asin` / `parent_asin` | str | Join `parent_asin` to item metadata |
| `rating` | float | 1.0–5.0 |
| `title` | str | Review headline |
| `text` | str | Review body — the bulk of the data volume |
| `timestamp` | int | Unix ms |
| `verified_purchase` | bool | |
| `helpful_vote` | int | |

## 2. Size limit — read this before importing

The full category does not fit the budget: 29.5M reviews with text is tens of GB, and production runs on **Azure SQL Database serverless, 1 vCore**.

Import a **5-core subset** (keep only users and books with at least 5 interactions). Suggested target:

| Table | Target rows |
|---|---|
| `books` | 100K–200K |
| `users` (source = `amazon`) | 300K–500K |
| `reviews` | 3M–5M |
| `review_contents` | optional — see below |

`review_contents` holds the review body. Skip it entirely if the model is collaborative-filtering only; that saves roughly 80% of the storage. Import it (optionally truncated to ~1000 characters) only if you do content-based or sentiment work.

Load with `BULK INSERT` or `bcp`, not row-by-row inserts. Drop non-clustered indexes before a large load and rebuild them afterwards.

## 3. Tables you write

Schema lives in `apps/web/prisma/schema.prisma`. Do not edit it — request changes from the website team so the migration stays in one place.

### Seeding (one-off ETL)

| Table | Source | Key mapping rule |
|---|---|---|
| `books` | `meta_Books.jsonl` | `books.external_id = parent_asin` (unique). Generate `slug` from the title |
| `authors`, `book_authors` | `details.Author` | Normalize the free-form string before inserting |
| `categories`, `book_categories` | `categories[]` | Build the tree; `path` is the materialized ancestor path, e.g. `1/14/58` |
| `book_images` | `images[]` | Also copy the primary image into `books.image_url` |
| `users` | distinct `user_id` | `source = 'amazon'`, `external_id = user_id`, `email` stays NULL |
| `reviews` | `Books.jsonl` | `source = 'amazon'`, unique on `(user_id, book_id)` — deduplicate before loading |
| `review_contents` | `review.text` | Optional |

Imports must be **idempotent**: match on `external_id` and upsert, so a re-run does not duplicate rows.

After seeding, recompute `books.average_rating` and `books.rating_count` from `reviews` rather than trusting the dataset's `average_rating` — the subset changes the aggregates.

### Serving tables (refreshed every training run)

| Table | Purpose |
|---|---|
| `model_versions` | One row per trained model: `name`, `version`, `metrics_json`, `is_active`. Exactly one active row at a time |
| `user_recommendations` | Per-user top-N: `user_id`, `book_id`, `score`, `rank`, `model_version_id` |
| `item_similarities` | "Readers also liked": `book_id`, `similar_book_id`, `score` |
| `popular_books` | Cold-start fallback: `book_id`, `category_id` (nullable = global), `score`, `period` |

Write a new `model_version` row first, insert its rows, then flip `is_active` in one transaction. The website reads whichever version is active, so a half-written batch is never served.

## 4. Tables you read

`interaction_events` — written by the website in real time. This is the live signal; `reviews` is frozen history.

| Column | Meaning |
|---|---|
| `user_id` | Nullable: anonymous visitors have only `session_id` |
| `book_id` | |
| `event_type` | `view`, `click`, `search`, `add_to_cart`, `remove_from_cart`, `purchase`, `wishlist` |
| `session_id` | Groups an anonymous visit |
| `occurred_at` | |

Weighting implicit feedback is your decision (a common starting point is view 1, add_to_cart 3, purchase 5). The table stores raw events; it does not store weights.

## 5. API contract

The website calls the service over the Container Apps internal network:

```
GET /api/v1/recommendations/{userId}?limit=10
→ 200 [{ "productId": "12345", "score": 0.87 }]
```

- `productId` is `books.id` as a string, not `parent_asin`.
- The website applies a 3 second timeout and falls back to `popular_books` on error, so a slow response degrades the page rather than breaking it.
- `GET /health` must return 200 — Container Apps uses it as the liveness probe.

Reading `user_recommendations` directly is the fast path. Do not run model inference inside the request.

## 6. Cold start

A visitor who just registered has no history. Resolution order:

1. `user_recommendations` for that user
2. `item_similarities` for books in the current session (`interaction_events`)
3. `popular_books` for the category being browsed
4. `popular_books` global

## 7. Evaluation

Store offline metrics in `model_versions.metrics_json` (precision@k, recall@k, NDCG@k) for every trained version. Split by time — train on reviews before a cutoff timestamp, test after it — not randomly; a random split leaks the future into training.
