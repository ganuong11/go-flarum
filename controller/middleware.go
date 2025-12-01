package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/corvofeng/go-flarum/model"
	"github.com/corvofeng/go-flarum/model/flarum"
	"github.com/corvofeng/go-flarum/util"
)

// Functions related to middleware

type (
	// HTTPHandleFunc is used to handle HTTP request functions
	HTTPHandleFunc func(w http.ResponseWriter, r *http.Request)

	// HTTPMiddleWareFunc middleware function
	HTTPMiddleWareFunc func(inner HTTPHandleFunc) HTTPHandleFunc
)

// MiddlewareArrayToChains organizes middleware into chained function calls
/* When we use multiple middlewares for a certain route, we can conveniently integrate them:
sp.HandleFunc(pat.Get("/"), controller.ArrayToChains(
	[]controller.ReqMiddle{
	controller.TestMiddleware,
	controller.TestMiddleware2,
	},
	h.FlarumIndex,
))

It will return a function wrapped by middleware in the following form:
controller.TestMiddleware(controller.TestMiddleware2(h.FlarumIndex))
*/
func MiddlewareArrayToChains(reqProcessFuncs []HTTPMiddleWareFunc, req HTTPHandleFunc) (rp HTTPHandleFunc) {
	rp = req
	rpfs := reqProcessFuncs
	for i := len(rpfs) - 1; i >= 0; i-- {
		rp = rpfs[i](rp)
	}
	return
}

// InitMiddlewareContext initializes the data structure needed by middleware
/*
Data transfer in middleware depends on context design. The current context exists as a struct,
a corresponding struct is created for each request and stores relevant information.
When actually processing the request, get the struct and obtain the information passed by the middleware.
*/
func (h *BaseHandler) InitMiddlewareContext(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		reqCtx := &ReqContext{}
		reqCtx.h = h
		r = r.WithContext(
			context.WithValue(r.Context(), ckRequest, reqCtx),
		)
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

// AdjustLocaleMiddleware adjusts user's language settings
func AdjustLocaleMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		reqCtx.locale = "en"

		// For users who have already logged in, according to their own configuration
		user := reqCtx.currentUser
		if user != nil {
			obj := flarum.NewResource(flarum.ECurrentUser, user.ID)
			data := obj.Attributes.(*flarum.CurrentUser)
			err := json.Unmarshal(user.Preferences, &data.Preferences)
			if err != nil || data.Preferences.Locale == "" {
				util.GetLogger().Errorf("Can't get preferences for user %d", user.ID)
			} else {
				reqCtx.locale = data.Preferences.Locale
			}
		} else if cookie, err := r.Cookie("locale"); err == nil {
			// For non-logged-in users, select based on cookie
			reqCtx.locale = cookie.Value
		}
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

// TrackerMiddleware records request time
func TrackerMiddleware(inner http.Handler) http.Handler {
	logger := util.GetLogger()
	mw := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		inner.ServeHTTP(w, r)
		logger.Noticef("Track [%6s] %s %s", r.Method, r.URL.Path, time.Since(start))
	}
	return http.HandlerFunc(mw)
}

// OriginMiddleware handles CORS issues
func (h *BaseHandler) OriginMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if strings.Contains(origin, h.App.Cf.Main.Domain) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		}
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

// AuthMiddleware validates user
func (h *BaseHandler) AuthMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		currentUser, err := h.CurrentUser(w, r)
		if err != nil {
			reqCtx.currentUser = nil
		} else {
			reqCtx.currentUser = &currentUser
		}
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

// InAPIMiddleware decorated with this indicates that the current request is an API request
func InAPIMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		reqCtx.inAPI = true
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

// InAdminMiddleware decorated with this indicates that the current request is an admin request
func InAdminMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		reqCtx.inAdmin = true
		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}

// MustAuthMiddleware requires user to be logged in
func MustAuthMiddleware(inner HTTPHandleFunc) HTTPHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		if reqCtx.currentUser == nil || reqCtx.currentUser.ID == 0 {
			w.WriteHeader(http.StatusForbidden)
			reqCtx.h.jsonify(w, response{
				Retcode: 403,
				Retmsg:  "User needs to log in",
			})
		} else {
			inner(w, r)
		}
	}
}

// MustCSRFMiddleware checks CSRF token
func MustCSRFMiddleware(inner HTTPHandleFunc) HTTPHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		h := reqCtx.h
		user := reqCtx.currentUser
		csrf := r.Header.Get("X-CSRF-Token")
		redisDB := h.App.RedisDB
		if !user.VerifyCSRFToken(redisDB, csrf) {
			w.WriteHeader(http.StatusForbidden)
			reqCtx.h.jsonify(w, response{
				Retcode: 403,
				Retmsg:  "User CSRF token error, please refresh the page and try again",
			})
		} else {
			inner(w, r)
		}
	}
}

// MustAdminUser must be an administrator to operate
func MustAdminUser(inner HTTPHandleFunc) HTTPHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		user := reqCtx.currentUser
		if !user.IsAdmin() {
			w.WriteHeader(http.StatusForbidden)
			reqCtx.h.jsonify(w, response{
				Retcode: 403,
				Retmsg:  "This action is only allowed for administrators",
			})
		}
		inner(w, r)
	}
}

// IsInAdmin in admin page
func IsInAdmin(inner HTTPHandleFunc) HTTPHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		reqCtx.inAdmin = true
		inner(w, r)
	}
}

func readUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

// RealIPMiddleware gets the user's real IP
func RealIPMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		reqCtx := GetRetContext(r)
		reqCtx.realIP = readUserIP(r)
		inner.ServeHTTP(w, r)
	}

	return http.HandlerFunc(mw)
}

// Record user actions
// If you want to make records, here is an example:
//
//	ctx.actionRecords = "Create a new discussion"
func ActionRecordsMiddleware(inner HTTPHandleFunc) HTTPHandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inner(w, r)
		reqCtx := GetRetContext(r)
		uid := uint64(0)
		if reqCtx.currentUser != nil {
			uid = reqCtx.currentUser.ID
		}
		if reqCtx.actionRecords == "" {
			return
		}

		logger := reqCtx.h.App.Logger
		if err := model.CreateActionRecord(
			reqCtx.h.App.GormDB,
			uid,
			reqCtx.actionRecords,
		); err != nil {
			logger.Error(
				"Can't create action record %s for %s", err, reqCtx.actionRecords)
		}
		logger.Debugf("Create action record %s for %d", reqCtx.actionRecords, uid)
	}
}
