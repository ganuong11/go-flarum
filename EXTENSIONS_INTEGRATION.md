# Extensions Integration

This document explains how Flarum extensions work within this custom Go-Flarum implementation.

## The Challenge

Standard Flarum extensions are typically composed of two parts:
1.  **Frontend (JS)**: Mithril.js code that modifies the UI.
2.  **Backend (PHP)**: PHP code that modifies API responses, adds database tables, or handles new routes.

Since this repository replaces the PHP backend with **Go**, **standard Flarum extensions will NOT work out of the box.**

## How Extensions Work Here

### 1. Frontend Integration
The frontend JavaScript for extensions is **manually integrated**.
*   **Location**: `view/extensions/`.
*   **Registration**: Extensions are not dynamically discovered. They are explicitly imported in the main entry file (`view/extensions/forum.js`).
    *   *Example*: `import * as flarum_emoji from '../framework/extensions/emoji/js/forum';`
*   **Build**: You must rebuild the frontend (`yarn build`) after adding or modifying an extension.

### 2. Backend Integration
The backend logic for extensions is **manually re-implemented in Go**.
*   There is no "plugin system" that loads external Go code.
*   **Routing**: New routes required by extensions must be hardcoded in `router/router.go`.
    *   *Example*: `fof/upload` extension has specific routes added:
        ```go
        apiSP.HandleFunc(pat.Post("/fof/upload"), ct.FlarumUpload)
        ```
*   **Logic**: The business logic (e.g., handling the file upload, saving to S3/Disk) is written in Go controllers (`controller/`).
*   **Database**: Any new tables or columns required by the extension must be added to the Go `model` structs and migration script.

## Supported Extensions
Based on the codebase analysis, the following extensions appear to have some level of support (either frontend-only or full-stack):

*   **flarum-emoji**: Frontend only.
*   **flarum-tags**: Core feature, fully implemented in Go.
*   **flarum-markdown**: Frontend rendering.
*   **flarum-mentions**: Frontend logic + likely backend parsing (supported in `model`).
*   **fof-upload**: **Fully implemented**. Has backend routes and controller logic for file uploads.
*   **auth-github**: **Fully implemented**. Has backend routes for OAuth flow.
*   **flarum-likes**: Supported in `model` (ReplyLikes).
*   **flarum-sticky**: Supported in `model` (Topic.IsSticky).

## Adding a New Extension

To add a new Flarum extension to this repo, you must:

1.  **Copy Frontend Code**: Place the extension's JS source in `view/extensions/my-extension`.
2.  **Register Frontend**: Import and register it in `view/extensions/forum.js`.
3.  **Analyze Backend Needs**: Does it need new API endpoints? Database changes?
4.  **Implement Backend**:
    *   Add routes in `router/router.go`.
    *   Add logic in `controller/`.
    *   Update `model/` if DB changes are needed.
5.  **Rebuild**: Run `yarn build` and recompile the Go server.
