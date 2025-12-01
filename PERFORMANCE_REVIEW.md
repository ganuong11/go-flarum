# Performance Review & Recommendations

This document outlines performance bottlenecks identified in the codebase and provides concrete recommendations for improvement.

## 1. Database Query Patterns (Critical)

### The "N+1" Problem in Ranking
**File:** `model/rank.go`
**Function:** `TimelyResort`

The current implementation iterates through every category, and then through **every single topic** to update its score in Redis.

```go
// Current Logic (Pseudocode)
categories := GetTags()
for _, cat := range categories {
    topics := GetAllTopics(cat.ID) // Query 1
    for _, topic := range topics {
        // ... Redis operations ...
        weight := getWeight(topic.ID) // Calls SQLArticleGetByID -> Query 2 (N times!)
    }
}
```
If you have 1,000 topics, this function performs 1,000+ separate database queries every time it runs. This will cripple the database as the forum grows.

**Recommendation:**
1.  **Batch Data Fetching:** Fetch all necessary data (view counts, comment counts) in a single SQL query or a few batch queries.
2.  **Calculate in Memory:** Perform the weight calculation in Go using the fetched data, rather than querying the DB for each item.

### Missing Indexes
**File:** `model/topic.go`
**Function:** `SQLGetTopicByTag`

This function joins `topic` and `tags` (via `topic_tags`).
```go
// Current GORM logic
err = ormFilter.Model(&tag).Association("Topics").Find(&topics)
```
Ensure that the `topic_tags` table has a composite index on `(tag_id, topic_id)` and `(topic_id, tag_id)`. Without these, filtering by tag becomes a full table scan.
*Based on `cmd/migration/main.go`, the primary key is `(topic_id, tag_id)`, which acts as an index, but an index starting with `tag_id` is needed for efficient "Get topics by tag" queries.*

## 2. Redis Usage & Ranking Algorithm

### Inefficient Redis Calls
**File:** `model/rank.go`

The `TimelyResort` function makes individual Redis calls (`ZRem`, `ZAddNX`, `ZAddXX`) for every topic. Network latency (RTT) will make this extremely slow.

**Recommendation:**
**Use Redis Pipelines.** A pipeline allows you to send hundreds of commands in a single network round-trip.
```go
pipe := redisDB.Pipeline()
for _, t := range topics {
    pipe.ZAddNX(...)
}
_, err := pipe.Exec()
```

### Complexity vs. Value
The `TimelyResort` function calculates a "weight" for every topic to create a custom sort order. However, the `getWeight` function currently returns `0` (hardcoded).
**Recommendation:** If the custom ranking algorithm is not actively used, **disable `TimelyResort` completely** to save massive resources. If it is needed, optimize it as described above.

## 3. Search Service Performance

### Search Result Highlighting (N+1)
**File:** `whooshsearch.py`
**Function:** `highlight_for_hit`

When a search returns results, the script iterates through them and queries MySQL *individually* for each result to fetch content if needed.
```python
with db.cursor() as cursor:
    cursor.execute("select * from topic where id = %s ...", (rlt['id']))
```
**Recommendation:**
After getting the list of IDs from Whoosh, perform a single MySQL query: `SELECT * FROM topic WHERE id IN (1, 2, 3...)`.

### Connection Management
The Python script opens a single global MySQL connection at startup.
**Risk:** This connection will time out (the "MySQL has gone away" error) if the search service sits idle for too long.
**Recommendation:** Use a connection pool (like `DBUtils` in Python) or ensure the code handles reconnection logic before executing queries.

## 4. General Code & Concurrency

### Cron Job Safety
**File:** `cmd/server/main.go`
**Line:** `go cr.MainCronJob()`

This starts the cron worker in a goroutine. If `MainCronJob` panics, the entire Go server (including the web server) will crash.
**Recommendation:** Wrap the goroutine in a function that uses `defer recover()` to log the panic and prevent a server crash.

### HTTP Timeout
**File:** `cmd/server/main.go`
**Line:** `srv := &http.Server{...}`

The server is initialized without explicit `ReadTimeout` or `WriteTimeout`. In Go, these default to "no timeout".
**Risk:** Slow clients (or malicious "Slowloris" attacks) can keep connections open indefinitely, consuming file descriptors and RAM.
**Recommendation:** Set reasonable timeouts:
```go
srv := &http.Server{
    Addr:         ":" + strconv.Itoa(mcf.HTTPPort),
    Handler:      root,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
}
```

## Summary of Immediate Actions

1.  **Disable `TimelyResort`** if you aren't using the custom ranking logic yet.
2.  **Add `ReadTimeout/WriteTimeout`** to the HTTP server in `main.go`.
3.  **Optimize Search Queries** to batch fetch MySQL data.
4.  **Add Indexes** for `topic_tags` (specifically `tag_id` first).
