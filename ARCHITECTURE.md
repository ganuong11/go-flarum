# High-Level Architecture

This project is a custom implementation of the [Flarum](https://flarum.org/) forum software backend using **Go (Golang)** instead of the standard PHP. It retains the Flarum frontend (Mithril.js) but replaces the entire server-side logic.

## Core Components

The architecture consists of the following key components:

1.  **Go Backend Server (`cmd/server`)**:
    *   The core of the application.
    *   Handles HTTP requests, API calls, business logic, and authentication.
    *   Replaces the Flarum PHP core.
    *   Serves the compiled frontend assets.

2.  **Frontend (Mithril.js)**:
    *   Located in `view/`.
    *   Uses standard Flarum frontend code (Mithril.js based).
    *   Communicates with the Go backend via a RESTful API.
    *   Built using Webpack.

3.  **Data Storage**:
    *   **MySQL**: Primary database for storing users, topics, posts, and configuration. Uses GORM for object-relational mapping.
    *   **Redis**: Used heavily for caching, specifically for:
        *   **Ranking System**: Storing weighted scores of topics.
        *   **View Counts**: Fast increments of topic views.
        *   **Comment Lists**: Caching ordered lists of comments for performance.
        *   **Captcha & Sessions**: Storing temporary authentication data.

4.  **Search Service (`whooshsearch.py`)**:
    *   A standalone Python script using the `Whoosh` library.
    *   Provides full-text search capabilities (likely because the Go backend implementation of search was either complex or this was a preferred microservice approach for Chinese language support via `jieba`).
    *   Connects to the MySQL database to index content.

## How It Fits Together

1.  **Request Flow**:
    *   User accesses the site -> Go Server handles the request.
    *   **Static Assets**: If searching for JS/CSS/Images, the Go server serves files from `static/` or `webpack/`.
    *   **API Calls**: The frontend (Mithril.js) makes AJAX requests to `/api/v1/...`.
    *   **Business Logic**: The Go server processes these requests (e.g., creating a post, logging in) using logic in `controller/`.
    *   **Database Interactions**: The controller uses `model/` to read/write to MySQL and Redis.

2.  **Extension Integration**:
    *   Unlike standard Flarum, extensions are **not** plug-and-play.
    *   **Frontend**: Extension JS code must be manually imported and bundled into the main JS file (`view/extensions/forum.js`).
    *   **Backend**: Extension backend logic (normally PHP) must be **re-implemented in Go** and manually wired into the router (`router/router.go`).

## Directory Structure Overview

*   `cmd/`: Entry points for applications (server, migration tool).
*   `config/`: Configuration templates.
*   `controller/`: HTTP handlers for routes.
*   `model/`: Database structs and logic (GORM).
*   `router/`: Route definitions (Goji).
*   `system/`: Global application state (DB connections, Config).
*   `view/`: Frontend source code.
*   `whooshsearch.py`: Python search service.
