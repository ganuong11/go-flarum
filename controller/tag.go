package controller

import (
	"net/http"

	"github.com/corvofeng/go-flarum/model"
	"github.com/corvofeng/go-flarum/model/flarum"

	"github.com/go-redis/redis/v7"
	"gorm.io/gorm"
)

func createFlarumTagAPIDoc(
	reqctx *ReqContext,
	gormDB *gorm.DB, redisDB *redis.Client,
	appConf model.AppConf,
	tz int,
) (flarum.CoreData, error) {
	var err error
	coreData := flarum.NewCoreData()
	currentUser := reqctx.currentUser
	inAPI := reqctx.inAPI
	siteInfo := model.GetSiteInfo(redisDB)
	apiDoc := &coreData.APIDocument // Note: this gets a pointer
	logger := reqctx.GetLogger()

	// Add current user's session information
	if currentUser != nil {
		user := model.FlarumCreateCurrentUser(*currentUser)
		coreData.AddCurrentUser(user)
		if !inAPI { // When making API requests, do not update CSRF information
			coreData.AddSessionData(user, currentUser.RefreshCSRF(redisDB))
		}
	}

	categories, err := model.SQLGetTags(gormDB)
	if err != nil {
		logger.Info("Can't get categories")
	}

	// Add all category information
	var flarumTags []flarum.Resource
	for _, category := range categories {
		tag := model.FlarumCreateTag(category)
		coreData.AppendResources(tag)
		flarumTags = append(flarumTags, tag)
	}
	// Add main site information
	coreData.AppendResources(model.FlarumCreateForumInfo(
		currentUser,
		appConf, siteInfo, flarumTags,
	))

	var res []flarum.Resource
	for _, category := range categories {
		tag := model.FlarumCreateTag(category)
		res = append(res, tag)
	}
	// article, err := model.SQLArticleGetByID(gormDB,    redisDB, 1)
	// if err != nil {
	// 	logger.Info("Can't get article", err.Error())
	// }
	// diss := model.FlarumCreateDiscussion(article)
	// apiDoc.AppendResources(diss)

	apiDoc.SetData(res)
	model.FlarumCreateLocale(&coreData, reqctx.locale)

	return coreData, err

}

// FlarumTagAll flarum homepage
func FlarumTagAll(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	inAPI := ctx.inAPI
	scf := h.App.Cf.Site

	redisDB := h.App.RedisDB
	logger := ctx.GetLogger()

	coreData, err := createFlarumTagAPIDoc(
		ctx, h.App.GormDB, redisDB, *h.App.Cf, scf.TimeZone)

	if err != nil {
		h.flarumErrorMsg(w, "Query tag information error:"+err.Error())
	}

	logger.Info(h.safeGetParm(r, "tag"))

	// If it is API, return directly
	if inAPI {
		h.jsonify(w, coreData.APIDocument)
		return
	}

	tpl := h.CurrentTpl(r)
	evn := InitPageData(r)
	evn.FlarumInfo = coreData
	h.Render(w, tpl, evn, "layout.html", "article.html")
}
