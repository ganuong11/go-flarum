package controller

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/corvofeng/go-flarum/model"
	"github.com/corvofeng/go-flarum/model/flarum"

	"github.com/dchest/captcha"
	"github.com/go-redis/redis/v7"
	"github.com/rs/xid"
	"goji.io/pat"
	"gorm.io/gorm"
)

// UserLogin user login and registration page
func (h *BaseHandler) UserLogin(w http.ResponseWriter, r *http.Request) {
	type pageData struct {
		BasePageData
		Act       string
		Token     string
		CaptchaID string
	}
	act := strings.TrimLeft(r.RequestURI, "/")
	title := "Login"
	if act == "register" {
		title = "Register"
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = title
	evn.Keywords = ""
	evn.Description = ""
	evn.IsMobile = tpl == "mobile"

	evn.ShowSideAd = true
	evn.PageName = "user_login_register"

	evn.Act = act
	evn.CaptchaID = model.NewCaptcha(filepath.Join(h.App.Cf.Main.StaticDir, "captcha"))

	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		token := xid.New().String()
		h.SetCookie(w, "token", token, 1)
	}

	h.Render(w, tpl, evn, "layout.html", "userlogin.html")
}

// NewCaptcha gets new captcha
func NewCaptcha(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	// Return and carry new captcha
	type captchaData struct {
		response
		NewCaptchaID string `json:"newCaptchaID"`
	}
	respCaptcha := captchaData{
		response{200, "success"},
		model.NewCaptcha(filepath.Join(h.App.Cf.Main.StaticDir, "captcha")),
	}
	h.jsonify(w, respCaptcha)
}

// FlarumUserRegister user registration
func FlarumUserRegister(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	rsp := response{}

	type recForm struct {
		Name            string `json:"username"`
		Password        string `json:"password"`
		CaptchaID       string `json:"captcha-id"`
		CaptchaSolution string `json:"captcha-solution"`
		Email           string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	var rec recForm
	if err := decoder.Decode(&rec); err != nil {
		rsp = normalRsp{
			400,
			"Form parsing error:" + err.Error(),
		}
		h.jsonify(w, rsp)
		return
	}
	defer r.Body.Close()

	// Return and carry new captcha
	type captchaData struct {
		response
		NewCaptchaID string `json:"newCaptchaID"`
	}

	var respCaptcha captchaData

	if !captcha.VerifyString(rec.CaptchaID, rec.CaptchaSolution) {
		respCaptcha = captchaData{
			response{405, "Captcha error"},
			model.NewCaptcha(filepath.Join(h.App.Cf.Main.StaticDir, "captcha")),
		}
		h.jsonify(w, respCaptcha)
		return
	}

	if _, err := model.SQLUserRegister(h.App.GormDB, rec.Name, rec.Email, rec.Password); err != nil {
		rsp = normalRsp{
			400,
			"Registration failed:" + err.Error(),
		}
		h.jsonify(w, rsp)
		return
	}

	rsp.Retcode = 200
	rsp.Retmsg = "Registration successful"

	h.jsonify(w, rsp)
}

// FlarumUserLogin flarum user login
func FlarumUserLogin(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	rsp := response{}
	logger := ctx.h.App.Logger
	type recForm struct {
		Identification  string `json:"identification"`
		Password        string `json:"password"`
		CaptchaID       string `json:"captcha-id"`
		CaptchaSolution string `json:"captcha-solution"`
	}
	decoder := json.NewDecoder(r.Body)
	var rec recForm
	err := decoder.Decode(&rec)
	if err != nil {
		rsp = normalRsp{400, "Data filling error:" + err.Error()}
		h.jsonify(w, rsp)
		return
	}
	if rec.Identification == "" || rec.Password == "" {
		rsp = normalRsp{400, "Please fill in login information and password"}
		h.jsonify(w, rsp)
		return
	}
	defer r.Body.Close()

	// Return and carry new captcha
	type captchaData struct {
		response
		NewCaptchaID string `json:"newCaptchaID"`
	}
	var respCaptcha captchaData
	if !captcha.VerifyString(rec.CaptchaID, rec.CaptchaSolution) {
		rsp = normalRsp{405, "Captcha error"}
		h.jsonify(w, rsp)
		return
	}

	redisDB := h.App.RedisDB

	uobj, err := model.SQLUserGetByName(h.App.GormDB, rec.Identification)
	if err != nil {
		rsp = normalRsp{405, "Login failed, please check username and password"}
		h.jsonify(w, respCaptcha)
		return
	}
	if uobj.Password != rec.Password {
		logger.Debugf("For user %s, want %s but get %s", uobj.Name, uobj.Password, rec.Password)
		rsp = normalRsp{405, "Login failed, please check username and password"}
		h.jsonify(w, rsp)
		return
	}
	sessionid := xid.New().String()
	// uobj.LastLoginTime = timeStamp
	uobj.Session = sessionid

	uobj.CachedToRedis(redisDB)
	h.SetCookie(w, "SessionID", uobj.StrID()+":"+sessionid, 365)

	rsp.Retcode = 200
	rsp.Retmsg = "Login successful"
	h.jsonify(w, rsp)
}

func userLogout(user model.User, h *BaseHandler, w http.ResponseWriter, r *http.Request) {
	redisDB := h.App.RedisDB
	cks := []string{"SessionID", "QQURLState", "WeiboURLState", "token"}
	for _, k := range cks {
		h.DelCookie(w, k)
	}
	user.CleareRedisCache(redisDB)
}

func createFlarumUserAPIDoc(
	reqctx *ReqContext,
	gormDB *gorm.DB,
	redisDB *redis.Client,
	appConf model.AppConf,
	tz int,
) (flarum.CoreData, error) {
	var err error
	coreData := flarum.NewCoreData()
	inAPI := reqctx.inAPI
	currentUser := reqctx.currentUser
	logger := reqctx.GetLogger()
	siteInfo := model.GetSiteInfo(redisDB)

	// All category information, used for the entire site information
	var flarumTags []flarum.Resource

	// Add current user's session information
	if currentUser != nil {
		user := model.FlarumCreateCurrentUser(*currentUser)
		coreData.AddCurrentUser(user)
		if !inAPI { // When making API requests, do not update CSRF information
			coreData.AddSessionData(user, currentUser.RefreshCSRF(redisDB))
		}
	}
	// Add current site information
	categories, err := model.SQLGetTags(gormDB)
	if err != nil {
		logger.Error("Get all categories error", err)
	}
	for _, category := range categories {
		flarumTags = append(flarumTags, model.FlarumCreateTag(category))
	}
	coreData.AppendResources(model.FlarumCreateForumInfo(
		currentUser,
		appConf, siteInfo, flarumTags,
	))
	model.FlarumCreateLocale(&coreData, reqctx.locale)

	return coreData, err
}

// FlarumUserLogout flarum user logout
func FlarumUserLogout(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	rsp := response{}
	redisDB := h.App.RedisDB

	token := r.FormValue("token")
	if token == "" {
		rsp = normalRsp{400, "Form parameter parsing error"}
		h.jsonify(w, rsp)
		return
	}
	user, err := h.CurrentUser(w, r)
	if err != nil {
		rsp = normalRsp{400, "User not logged in:" + err.Error()}
		h.jsonify(w, rsp)
		return
	}

	if !user.VerifyCSRFToken(redisDB, token) {
		rsp = normalRsp{400, "CSRF error"}
		h.jsonify(w, rsp)
		return
	}

	userLogout(user, h, w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	rsp.Retcode = 200
	rsp.Retmsg = "Logout successful"
	h.jsonify(w, rsp)
}

// FlarumUser flarum user query
func FlarumUser(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	//
	inAPI := ctx.inAPI

	_userID := pat.Param(r, "uid")
	user, err := model.SQLUserGet(h.App.GormDB, _userID)

	if err != nil {
		h.flarumErrorJsonify(w, createSimpleFlarumError("Get user information error: "+err.Error()))
		return
	}

	coreData := flarum.NewCoreData()
	apiDoc := &coreData.APIDocument
	apiDoc.SetData(model.FlarumCreateCurrentUser(user))

	if inAPI {
		h.jsonify(w, apiDoc)
		return
	}
	tpl := h.CurrentTpl(r)
	evn := InitPageData(r)
	evn.FlarumInfo = coreData

	h.Render(w, tpl, evn, "layout.html", "index.html")
}

// FlarumUserSettings flarum user query
func FlarumUserSettings(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h

	redisDB := h.App.RedisDB
	gormDB := h.App.GormDB
	scf := h.App.Cf.Site
	tpl := h.CurrentTpl(r)

	coreData, err := createFlarumUserAPIDoc(ctx, gormDB, redisDB, *h.App.Cf, scf.TimeZone)
	if err != nil {
		h.flarumErrorMsg(w, "Query user information error:"+err.Error())
	}
	evn := InitPageData(r)
	evn.FlarumInfo = coreData

	h.Render(w, tpl, evn, "layout.html", "index.html")
}

// FlarumUserPage flarum user query
func FlarumUserPage(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	inAPI := ctx.inAPI
	logger := h.App.Logger

	username := pat.Param(r, "username")
	user, err := model.SQLUserGetByName(h.App.GormDB, username)

	if err != nil {
		h.flarumErrorJsonify(w, createSimpleFlarumError("Get user information error"+err.Error()))
		return
	}

	redisDB := h.App.RedisDB
	scf := h.App.Cf.Site
	df := dissFilter{
		FT:        eUserPost,
		UID:       user.ID,
		pageLimit: uint64(h.App.Cf.Site.HomeShowNum),
	}
	logger.Info("Get user for ", user.ID, df)

	coreData, err := createFlarumPageAPIDoc(ctx, redisDB, h.App.GormDB, *h.App.Cf, df, scf.TimeZone)
	if err != nil {
		h.flarumErrorMsg(w, "Unable to get post information")
		return
	}

	// Add main site information
	si := model.GetSiteInfo(redisDB)

	coreData.AppendResources(model.FlarumCreateForumInfo(
		ctx.currentUser,
		*h.App.Cf, si,
		[]flarum.Resource{},
	))

	apiDoc := &coreData.APIDocument

	u := model.FlarumCreateCurrentUser(user)
	coreData.AppendResources(u)
	apiDoc.SetData(u)
	currentUser := ctx.currentUser
	// Add current user's session information
	if currentUser != nil {
		user := model.FlarumCreateCurrentUser(*currentUser)
		coreData.AddCurrentUser(user)
		if !inAPI { // When making API requests, do not update CSRF information
			coreData.AddSessionData(user, currentUser.RefreshCSRF(redisDB))
		}
	}

	apiDoc.Links["first"] = ""
	apiDoc.Links["next"] = ""

	tpl := h.CurrentTpl(r)
	evn := InitPageData(r)
	evn.FlarumInfo = coreData

	h.Render(w, tpl, evn, "layout.html", "index.html")
}

// FlarumUserUpdate flarum user update configuration information
func FlarumUserUpdate(w http.ResponseWriter, r *http.Request) {
	_uid := pat.Param(r, "uid")
	ctx := GetRetContext(r)
	h := ctx.h
	if ctx.currentUser.StrID() != _uid {
		h.flarumErrorMsg(w, "Currently only allows modifying own configuration")
		return
	}

	type UserUpdate struct {
		Data struct {
			Type string `json:"type"`
			ID   string `json:"id"`

			Attributes struct {
				Preferences flarum.Preferences `json:"preferences"`
			} `json:"attributes"`
		} `json:"data"`
	}
	userUpdateInfo := UserUpdate{}
	err := json.NewDecoder(r.Body).Decode(&userUpdateInfo)
	if err != nil {
		h.flarumErrorMsg(w, "Parse JSON error:"+err.Error())
		return
	}
	ctx.currentUser.SetPreference(
		h.App.GormDB,
		userUpdateInfo.Data.Attributes.Preferences,
	)
	coreData := flarum.NewCoreData()
	apiDoc := &coreData.APIDocument
	apiDoc.SetData(model.FlarumCreateCurrentUser(*ctx.currentUser))

	if ctx.inAPI {
		h.jsonify(w, apiDoc)
		return
	}
}
