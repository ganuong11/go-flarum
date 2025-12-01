# Deep Dive: Component Analysis

## 1. Backend Architecture & Routing

The backend is built with **Go** and uses the `goji.io` router.

### Structure
*   **Entry Point**: `cmd/server/main.go`. This initializes the `system.Application` (which holds DB connections and config), starts the cron jobs, and sets up the HTTP server.
*   **Router**: `router/router.go` defines the routing tree.
    *   It separates routes into:
        *   **Static**: `/static/*`, `/webpack/*`
        *   **Web**: Standard Flarum pages (e.g., `/d/123-slug`) -> handled by `ct.FlarumIndex` which serves the HTML shell.
        *   **API**: `/api/v1/*` -> handled by specific controllers (e.g., `ct.FlarumAPIDiscussions`).
        *   **Admin**: `/admin` routes.
*   **Middleware**: Located in `controller/middleware.go` (implied or inline). Uses chains for:
    *   `MustAuthMiddleware`: Ensures user is logged in.
    *   `MustCSRFMiddleware`: CSRF protection.
    *   `TrackerMiddleware`: Request logging/tracking.

### Request Handling
Requests are routed to **Controllers** (`controller/` package). A typical flow:
1.  Route match (e.g., `GET /api/v1/discussions`).
2.  Middleware checks (Auth, CSRF).
3.  Controller function (`FlarumAPIDiscussions`) executes.
4.  Controller calls `model` to fetch data (MySQL/Redis).
5.  Controller serializes data to JSON and responds.

## 2. Database Schema

The application uses **MySQL** as the primary source of truth, managed via **GORM**.

### Key Models (`model/`)
*   **Topic**: The core discussion unit. Contains `Title`, `Content` (of the first post?), `UserID`, and stats like `CommentCount`.
*   **Reply (Comment)**: Individual posts within a topic. Linked via `TopicID`.
*   **User**: User accounts.
*   **Tag**: Categories/Tags for topics. Many-to-Many relationship with `Topic` via `topic_tags` table.
*   **BlogMeta**: Additional metadata for blog-like usage of Flarum topics.

### Migration
*   **Tool**: `cmd/migration/main.go`.
*   **Process**: Uses `gorm.AutoMigrate` to automatically create/update tables based on struct definitions. It also inserts default data (like the `root` user and `Welcome` topic) if the DB is empty.

## 3. Caching Mechanism

The project relies heavily on **Redis** for performance and specific feature implementations.

### Ranking System (`model/rank.go`)
A complex "weight" system is implemented to order topics, likely to support a "Hot" or "Top" sorting algorithm different from standard "Latest".
*   **Structure**: Redis Sorted Sets (`ZSET`).
*   **Key**: `rank-category-{cid}`.
*   **Logic**: Topics are assigned a score (weight). `TimelyResort` function recalculates and updates these scores in Redis.
*   **Pagination**: `GetTopicListByPageNum` fetches a slice of Topic IDs from the Redis ZSET, avoiding heavy `ORDER BY` queries in MySQL.

### Data Caching
*   **Article Views**: Stored in a Redis Hash (`article_views`) to allow high-concurrency increments without locking the MySQL row.
*   **Comments**: `Topic.CacheCommentList` stores the *order* of comments (Reply IDs) for a topic in a Redis ZSET. This helps in quickly retrieving the IDs for a specific page of comments without querying MySQL with offsets.

## 4. Frontend Build Process

The frontend uses the standard **Mithril.js** framework found in Flarum, but the build process is adapted for this repo.

### Build Tools
*   **Webpack 5**: The primary bundler.
*   **flarum-webpack-config**: A package used to inherit standard Flarum build configurations.

### Configuration (`webpack.config.js`)
*   **Entry Points**:
    *   `forum`: The main forum app.
    *   `admin`: The admin interface.
    *   **Extensions**: Explicit entry points are defined for extensions located in `view/extensions/`.
*   **Output**: Files are compiled to `static/flarum/`.
*   **Styling**: Uses LESS, compiled via `mini-css-extract-plugin`.

### Workflow
1.  Developer modifies `.js` or `.less` files in `view/`.
2.  Runs `yarn build` (or `npm run build`).
3.  Webpack bundles dependencies and source code.
4.  Go server serves the resulting `dist` files.
