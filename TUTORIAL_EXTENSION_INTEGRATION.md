# Tutorial: Integrating a Complex Extension into Go-Flarum

This tutorial guides you through creating and integrating a complex, full-stack extension into the `go-flarum` repository.

**Target Extension:** `daily-checkin`
**Features:**
*   **Frontend**: A "Check In" button on the forum homepage.
*   **Backend**: API to handle check-ins.
*   **Database**: Store check-in records.
*   **Caching**: Use Redis to show a global "Check-ins Today" counter.
*   **Admin**: A settings page to customize the button text.

---

## Part 1: Frontend (Mithril.js)

Since `go-flarum` uses a custom webpack build, we will manually create the extension structure in `view/extensions/`.

### 1. Create Directory Structure
Create a folder `view/extensions/daily-checkin` with the following structure:
```
view/extensions/daily-checkin/
├── js/
│   ├── src/
│   │   ├── admin/
│   │   │   └── index.js
│   │   └── forum/
│   │       ├── components/
│   │       │   └── CheckInWidget.js
│   │       └── index.js
│   └── package.json (Optional, for reference)
```

### 2. Implement Forum Logic
We will add a widget to the side pane.

**`view/extensions/daily-checkin/js/src/forum/components/CheckInWidget.js`**
```javascript
import Component from 'flarum/common/Component';
import Button from 'flarum/common/components/Button';
import app from 'flarum/forum/app';

export default class CheckInWidget extends Component {
  oninit(vnode) {
    super.oninit(vnode);
    this.loading = false;
    this.hasCheckedIn = false;
    this.count = 0;

    this.loadStatus();
  }

  view() {
    const label = app.forum.attribute('dailyCheckinLabel') || 'Daily Check-In';

    return (
      <div className="CheckInWidget" style="background: #fff; padding: 15px; border-radius: 5px; margin-bottom: 10px;">
        <h3>{label}</h3>
        <p>Total today: {this.count}</p>

        {this.hasCheckedIn ? (
          <div className="Button Button--primary Button--block" disabled>
            Already Checked In!
          </div>
        ) : (
          <Button
            className="Button Button--primary Button--block"
            loading={this.loading}
            onclick={this.handleCheckIn.bind(this)}
          >
            Check In Now
          </Button>
        )}
      </div>
    );
  }

  loadStatus() {
    // Call our custom Go API
    app.request({
      method: 'GET',
      url: app.forum.attribute('apiUrl') + '/daily-checkin/status'
    }).then(result => {
      this.hasCheckedIn = result.checked_in;
      this.count = result.count;
      m.redraw();
    });
  }

  handleCheckIn() {
    this.loading = true;

    app.request({
      method: 'POST',
      url: app.forum.attribute('apiUrl') + '/daily-checkin/do'
    }).then(result => {
      this.hasCheckedIn = true;
      this.count = result.count;
      this.loading = false;
      m.redraw();
    });
  }
}
```

**`view/extensions/daily-checkin/js/src/forum/index.js`**
```javascript
import { extend } from 'flarum/common/extend';
import IndexPage from 'flarum/forum/components/IndexPage';
import CheckInWidget from './components/CheckInWidget';

app.initializers.add('daily-checkin', () => {
  extend(IndexPage.prototype, 'sidebarItems', function(items) {
    items.add('dailyCheckin', <CheckInWidget />, 20);
  });
});
```

### 3. Implement Admin Logic
Add a setting field.

**`view/extensions/daily-checkin/js/src/admin/index.js`**
```javascript
import app from 'flarum/admin/app';

app.initializers.add('daily-checkin', () => {
  app.extensionData.for('daily-checkin')
    .registerSetting({
      setting: 'daily-checkin.label', // Key matches Go config
      label: 'Check-In Button Label',
      type: 'text',
    });
});
```

### 4. Register the Extension
Now, tell the main build system about your new extension.

**Edit `view/extensions/forum.js`**:
```javascript
// ... existing imports
import * as daily_checkin from './daily-checkin/js/src/forum';

// ... existing code
flarum.extensions['daily-checkin'] = daily_checkin;
```

**Edit `view/extensions/admin.js`**:
```javascript
// ... existing imports
import * as daily_checkin_admin from './daily-checkin/js/src/admin';

// ...
flarum.extensions['daily-checkin'] = daily_checkin_admin;
```

---

## Part 2: Backend (Go - Database Model)

We need a table to store who checked in and when.

**Create `model/checkin.go`**:
```go
package model

import (
	"time"
	"gorm.io/gorm"
)

type DailyCheckin struct {
	gorm.Model
	UserID    uint64 `gorm:"index"`
	CheckDate string `gorm:"size:10;index"` // Format YYYY-MM-DD
}

// SQLHasCheckedIn checks if user checked in today
func SQLHasCheckedIn(db *gorm.DB, userID uint64) bool {
	var count int64
	today := time.Now().Format("2006-01-02")
	db.Model(&DailyCheckin{}).Where("user_id = ? AND check_date = ?", userID, today).Count(&count)
	return count > 0
}

// SQLDoCheckin performs the checkin
func SQLDoCheckin(db *gorm.DB, userID uint64) error {
	today := time.Now().Format("2006-01-02")
	checkin := DailyCheckin{
		UserID:    userID,
		CheckDate: today,
	}
	return db.Create(&checkin).Error
}

// SQLGetTodayCount gets total checkins for today
func SQLGetTodayCount(db *gorm.DB) int64 {
	var count int64
	today := time.Now().Format("2006-01-02")
	db.Model(&DailyCheckin{}).Where("check_date = ?", today).Count(&count)
	return count
}
```

**Run Migrations**:
Edit `cmd/migration/main.go` to include the new model:
```go
// In main() function
util.CheckError(app.GormDB.AutoMigrate(model.DailyCheckin{}), "migrate daily checkin")
```

---

## Part 3: Backend (Go - Controller & Caching)

We need to handle the API requests and use Redis for caching the counter.

**Create `controller/checkin.go`**:
```go
package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/corvofeng/go-flarum/model"
	"github.com/corvofeng/go-flarum/util"
)

// CheckinStatusResponse JSON response
type CheckinStatusResponse struct {
	CheckedIn bool  `json:"checked_in"`
	Count     int64 `json:"count"`
}

func (h *BaseHandler) GetCheckinStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Get current logged in user (if any)
	currentUser, ok := ctx.Value(KeyCurrentUser).(model.User)

	isCheckedIn := false
	if ok && currentUser.ID > 0 {
		isCheckedIn = model.SQLHasCheckedIn(h.App.GormDB, currentUser.ID)
	}

	// Caching: Get today's count from Redis first
	today := time.Now().Format("2006-01-02")
	cacheKey := fmt.Sprintf("checkin_count_%s", today)

	count, err := h.App.RedisDB.Get(cacheKey).Int64()
	if err != nil {
		// Cache miss, query DB
		count = model.SQLGetTodayCount(h.App.GormDB)
		// Set cache (expire in 1 hour)
		h.App.RedisDB.Set(cacheKey, count, time.Hour)
	}

	util.WriteJSON(w, CheckinStatusResponse{
		CheckedIn: isCheckedIn,
		Count:     count,
	})
}

func (h *BaseHandler) DoCheckin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	currentUser := ctx.Value(KeyCurrentUser).(model.User) // Middleware ensures auth

	// Check if already done
	if model.SQLHasCheckedIn(h.App.GormDB, currentUser.ID) {
		util.WriteError(w, 400, "Already checked in today")
		return
	}

	// Perform DB Write
	if err := model.SQLDoCheckin(h.App.GormDB, currentUser.ID); err != nil {
		util.WriteError(w, 500, "Database error")
		return
	}

	// Caching: Increment Redis counter
	today := time.Now().Format("2006-01-02")
	cacheKey := fmt.Sprintf("checkin_count_%s", today)
	newCount, _ := h.App.RedisDB.Incr(cacheKey).Result()

	util.WriteJSON(w, CheckinStatusResponse{
		CheckedIn: true,
		Count:     newCount,
	})
}
```

---

## Part 4: Backend (Go - Router)

Finally, wire up the routes.

**Edit `router/router.go`**:

Find `NewFlarumAPIRouter` and add:

```go
func NewFlarumAPIRouter(app *system.Application, apiSP *goji.Mux) *goji.Mux {
    // ... existing code ...

    // Daily Checkin Extension Routes
    apiSP.HandleFunc(pat.Get("/daily-checkin/status"), ct.MiddlewareArrayToChains(
        []ct.HTTPMiddleWareFunc{
             // Optional: ct.MustAuthMiddleware if you only want logged in users to see it
        },
        ct.GetCheckinStatus, // You need to export this or add it to BaseHandler properly
    ))

    apiSP.HandleFunc(pat.Post("/daily-checkin/do"), ct.MiddlewareArrayToChains(
        []ct.HTTPMiddleWareFunc{
            ct.MustAuthMiddleware, // Ensure user is logged in
        },
        ct.DoCheckin,
    ))

    // ...
}
```
*Note: You'll need to make sure `GetCheckinStatus` and `DoCheckin` are attached to `BaseHandler` or called correctly in `router.go`.*

To make them accessible in `router.go`, in `controller/checkin.go`, ensure they are methods of a struct or standalone functions accepting `*system.Application`. The example above uses `(h *BaseHandler)`. You might need to instantiate a handler in `router.go`:

```go
// In router/router.go
checkinHandler := &ct.BaseHandler{App: app}
apiSP.HandleFunc(pat.Get("/daily-checkin/status"), checkinHandler.GetCheckinStatus)
```

---

## Part 5: Admin Settings (Configuration)

Flarum stores settings in a `settings` table (key-value). In this Go implementation, you need to expose these settings to the frontend.

1.  **Frontend**: We already added `app.extensionData.registerSetting`.
2.  **Backend**: The standard Flarum settings API might be partially implemented. Check `controller/admin.go` or `model/config.go`.
    *   If `api/settings` endpoint exists, it usually returns all settings.
    *   To inject your setting (`daily-checkin.label`) into the Forum app payload (so `app.forum.attribute('dailyCheckinLabel')` works), you need to modify the `FlarumIndex` controller.

**Edit `controller/index.go` (or wherever `FlarumIndex` is)**:
Look for where `attributes` map is constructed.
```go
// Pseudocode for FlarumIndex
attributes := map[string]interface{}{
    "apiUrl": "/api/v1",
    // Add your setting
    "dailyCheckinLabel": model.GetSetting(db, "daily-checkin.label", "Daily Check-In"),
}
```
*You will need to implement a helper `model.GetSetting` that queries the `settings` table.*

---

## Summary of Workflow

1.  **Code**: Write JS in `view/extensions/` and Go in `model/`, `controller/`.
2.  **Build Frontend**: Run `yarn build` in the root directory. This bundles your new JS.
3.  **Build Backend**: Run `go build -o server cmd/server/main.go`.
4.  **Migrate**: Run `./server -initdb` (or just start it if auto-migrate is on).
5.  **Run**: Start the server and test!
