package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/corvofeng/go-flarum/model"
	"github.com/corvofeng/go-flarum/model/flarum"
	"github.com/corvofeng/go-flarum/util"

	"github.com/go-redis/redis/v7"
	"goji.io/pat"
	"gorm.io/gorm"
)

type replyFilter struct {
	FT    filterType
	AID   uint64 // Comments of a post
	CID   uint64 // Information of a single comment
	UID   uint64 // Comments created by a certain user
	Page  uint64
	Limit uint64
	IDS   []uint64

	RenderLimit uint64 // The number of comments displayed on the current page, generally only a few

	LastReadPostNumber uint64
	NearNumber         uint64
	StartNumber        uint64
}

// Get comment information
// eArticle: Get comment information below a post
// eUserPost: Get the user's latest comments
// ePost: Get one comment information
func createFlarumPostAPIDoc(
	reqctx *ReqContext,
	gormDB *gorm.DB, redisDB *redis.Client,
	appConf model.AppConf,
	rf replyFilter,
	tz int,
) (flarum.CoreData, error) {
	var err error
	coreData := flarum.NewCoreData()
	apiDoc := &coreData.APIDocument
	inAPI := reqctx.inAPI
	currentUser := reqctx.currentUser
	logger := reqctx.GetLogger()
	siteInfo := model.GetSiteInfo(redisDB)

	rf.RenderLimit = 20
	// Current all comment resources: obtained from database
	// var comments []model.CommentListItem
	var comments []model.Comment
	// Current all comment resources: API return
	var flarumPosts []flarum.Resource

	// Information of all categories, used for the entire site information
	var flarumTags []flarum.Resource

	// Current topic information
	var curDisscussion *flarum.Resource

	// When using startNumber, load more data
	if rf.StartNumber < 10 {
		rf.StartNumber = 1
	} else {
		rf.StartNumber = rf.StartNumber - 10
	}
	logger.Debugf("Get comments with filter: %+v", rf)

	if rf.FT == eArticle { // Get all comments of a post
		comments, err = model.SQLCommentListByTopic(gormDB, redisDB, rf.AID, rf.Limit, tz)
	} else if rf.FT == ePost {
		comments, err = model.SQLCommentListByCID(gormDB, redisDB, rf.CID, rf.Limit, tz)
	} else if rf.FT == eUserPost {
		comments, err = model.SQLCommentListByUser(gormDB, redisDB, rf.UID, rf.Limit, tz)
	} else if rf.FT == ePosts { // Get comments based on post list
		comments, err = model.SQLCommentListByList(gormDB, redisDB, rf.IDS, tz)
		rf.RenderLimit = uint64(len(rf.IDS))
	} else {
		logger.Warningf("Can't process filter: `%s`", rf.FT)
		return coreData, fmt.Errorf("can't process filter: `%s`", rf.FT)
	}
	if err != nil {
		logger.Error("Get comments error with filter %+v, %s", rf, err)
	}

	commentsLen := uint64(len(comments))
	logger.Debugf("Get %d comments for %d", commentsLen, rf.AID)
	if commentsLen == 0 {
		logger.Errorf("Can't get any comment for %d", rf.AID)
	}

	// Get the proper commentsLen value
	if commentsLen < rf.RenderLimit {
		// logger.Warning("Can't get proper comments for", rf.AID)
		rf.RenderLimit = commentsLen
	}

	if rf.AID == 0 && commentsLen != 0 { // When there is no AID, supplement
		rf.AID = comments[0].AID
	}

	allUsers := make(map[uint64]bool)       // Used to save already added users, deduplicate
	allDiscussions := make(map[uint64]bool) // Used to save already added posts, deduplicate

	// Add current user, and session information
	if currentUser != nil {
		user := model.FlarumCreateCurrentUser(*currentUser)
		allUsers[user.GetID()] = true
		coreData.AddCurrentUser(user)
		if !inAPI { // When making API request, do not update csrf information, otherwise update
			coreData.AddSessionData(user, currentUser.RefreshCSRF(redisDB))
		}
	}

	hasUpdateComments := make(chan bool)

	// For a certain topic, add directly here
	for rf.FT == eArticle || rf.FT == ePost || rf.FT == ePosts {
		article, err := model.SQLArticleGetByID(gormDB, redisDB, rf.AID)
		logger.Debugf("Get article for %s", article.GetFormatedString())
		if err != nil {
			logger.Warning("Can't get article: ", rf.AID, err)
			break
		}

		diss := model.FlarumCreateDiscussion(article)
		curDisscussion = &diss
		apiDoc.AppendResources(*curDisscussion)
		allDiscussions[rf.AID] = true
		if rf.FT == eArticle || rf.FT == ePost { // When querying the current post information, update the post's comment information in redis, ePost is the operation of just adding the post
			go article.CacheCommentList(redisDB, comments, hasUpdateComments)
		}

		if article.BlogMetaData.ID != 0 {
			logger.Debugf("Create blog meta for article: %s", article.BlogMetaData.GetFormatedString())
			apiDoc.AppendResources(model.FlarumCreateBlogMeta(article.BlogMetaData, currentUser))
		}
		break
	}
	logger.Debugf("Get topic comments: %+v", func() []string {
		cs := []string{}
		for _, c := range comments {
			cs = append(cs, fmt.Sprintf("%d", c.ID))
		}
		return cs
	}())

	for _, comment := range comments {
		// lastReadPostNumber is only used to record the read position, no need to return comment information
		if rf.LastReadPostNumber != 0 {
			break
		}

		// Use lastReadPostNumber to mark the starting position
		if rf.StartNumber != 0 && comment.Number < rf.StartNumber {
			continue
		}

		// Use renderlimit to mark the end position
		if rf.RenderLimit == 0 {
			break
		}

		if _, ok := allUsers[comment.UID]; !ok {
			u, err := model.SQLUserGetByID(gormDB, comment.UID)
			if err != nil {
				logger.Warningf("Get user %d error: %s", comment.UID, err)
			} else {
				user := model.FlarumCreateUser(u)
				allUsers[comment.UID] = true
				coreData.AppendResources(user)
			}
		}

		if _, ok := allDiscussions[comment.AID]; !ok {
			article, err := model.SQLArticleGetByID(gormDB, redisDB, comment.AID)
			if err != nil {
				logger.Warning("Can't get article: ", comment.AID, err)
			} else {
				apiDoc.AppendResources(model.FlarumCreateDiscussion(article))
			}
			allDiscussions[comment.AID] = true
		}

		// Process user's like information
		for _, userID := range comment.Likes {
			if _, ok := allUsers[userID]; !ok {
				u, err := model.SQLUserGetByID(gormDB, userID)
				if err != nil {
					logger.Warningf("Get user %d error: %s", userID, err)
				} else {
					user := model.FlarumCreateUser(u)
					allUsers[user.GetID()] = true
					coreData.AppendResources(user)
				}
			}
		}

		post := model.FlarumCreatePost(comment, currentUser)
		logger.Debugf("Create comment %d post for %s", comment.ID, post.ID)
		apiDoc.AppendResources(post)
		flarumPosts = append(flarumPosts, post)

		rf.RenderLimit--
	}

	// For the current topic, complete its relationship information
	if curDisscussion != nil {
		if rf.FT == eArticle || rf.FT == ePost { // If querying all comments, wait a bit
			<-hasUpdateComments
		}
		article, _ := model.SQLArticleGetByID(gormDB, redisDB, rf.AID)
		postRelation := model.FlarumCreatePostRelations([]flarum.Resource{}, article.GetCommentIDList(redisDB))
		curDisscussion.BindRelations("Posts", postRelation)
	}

	// Add current site information
	tags, err := model.SQLGetTags(gormDB)
	if err != nil {
		logger.Error("Get all categories error", err)
	}
	for _, category := range tags {
		tag := model.FlarumCreateTag(category)
		coreData.AppendResources(tag)
		flarumTags = append(flarumTags, model.FlarumCreateTag(category))
	}
	coreData.AppendResources(model.FlarumCreateForumInfo(
		currentUser,
		appConf, siteInfo,
		flarumTags,
	))

	if rf.FT == eArticle {
		// apiDoc.SetData(flarumPosts) // Main information is all comments
		if rf.NearNumber != 0 {
			apiDoc.SetData(flarumPosts) // Main information is all comments
		} else {
			// if inAPI {
			// 	apiDoc.SetData(flarumPosts) // Main information is the current post
			// } else {
			// }
			apiDoc.SetData(*curDisscussion) // Main information is the current post
		}
	} else if rf.FT == ePost {
		// comment, err := model.SQLGetCommentByID(   redisDB, rf.CID, tz)
		// if err != nil {
		// 	logger.Error("Get comment error:", err)
		// }
		// commentListItem := model.CommentListItem{Comment: comment}
		// post := model.FlarumCreatePost(commentListItem, currentUser)
		if len(flarumPosts) >= 0 {
			apiDoc.SetData(flarumPosts[0]) // Main information is this comment
		}
	} else if rf.FT == eUserPost || rf.FT == ePosts {
		apiDoc.SetData(flarumPosts) // Main information is all comments
	}
	logger.Debugf("Update the api doc: %+v", apiDoc)
	// apiDoc.Links["first"] = "https://flarum.yjzq.fun/api/v1/flarum/discussions?sort=&page%5Blimit%5D=20"
	// apiDoc.Links["next"] = "https://flarum.yjzq.fun/api/v1/flarum/discussions?sort=&page%5Blimit%5D=20"
	model.FlarumCreateLocale(&coreData, reqctx.locale)

	return coreData, nil
}

// FlarumAPICreatePost flarum interface for commenting
func FlarumAPICreatePost(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h

	redisDB := h.App.RedisDB
	scf := h.App.Cf.Site
	// logger := ctx.GetLogger()

	type PostedReply struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				Content string `json:"content"`
			} `json:"attributes"`
			Relationships struct {
				Discussion struct {
					Data struct {
						Type string `json:"type"`
						ID   string `json:"id"`
					} `json:"data"`
				} `json:"discussion"`
			} `json:"relationships"`
		} `json:"data"`
	}

	reply := PostedReply{}
	err := json.NewDecoder(r.Body).Decode(&reply)
	if err != nil {
		h.flarumErrorMsg(w, "Parse json error:"+err.Error())
		return
	}
	aid, err := strconv.ParseUint(reply.Data.Relationships.Discussion.Data.ID, 10, 64)
	if err != nil {
		h.flarumErrorMsg(w, "Unable to get correct post information:"+err.Error())
		return
	}

	now := util.TimeNow()
	comment := model.Comment{
		Reply: model.Reply{
			AID:      aid,
			UID:      ctx.currentUser.ID,
			Content:  reply.Data.Attributes.Content,
			Number:   1,
			ClientIP: ctx.realIP,
			AddTime:  now,
		},
	}
	comment.Content = model.PreProcessUserMention(h.App.GormDB, redisDB, scf.TimeZone, comment.Content)

	if ok, err := comment.CreateFlarumComment(h.App.GormDB); !ok {
		h.flarumErrorMsg(w, "Error creating comment:"+err.Error())
		return
	}

	rf := replyFilter{
		FT:    ePost,
		AID:   comment.AID,
		CID:   comment.ID,
		Limit: comment.Number,
	}

	coreData, err := createFlarumPostAPIDoc(ctx, h.App.GormDB, redisDB, *h.App.Cf, rf, scf.TimeZone)
	if err != nil {
		h.flarumErrorMsg(w, "Error querying comment:"+err.Error())
		return
	}

	h.jsonify(w, coreData.APIDocument)
}

// FlarumConfirmUserAndPost confirm the current user's comment information
// FIXME: I only know that this function is called when commenting, @ other users, but I don't know the specific behavior of the interface
func FlarumConfirmUserAndPost(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	h := ctx.h
	scf := h.App.Cf.Site
	//
	// redisDB := h.App.RedisDB
	// logger := ctx.GetLogger()

	_filter := strings.TrimSpace(r.FormValue("filter[q]"))
	_pageLimit := r.FormValue("page[limit]")

	// filterData := strings.Split(_filter, "#")
	// if len(filterData) != 2 {
	// 	h.flarumErrorJsonify(w, createSimpleFlarumError("The given reply information is incorrect"))
	// 	return
	// }

	// pageLimit, err := strconv.ParseUint(_pageLimit, 10, 64)
	// if err != nil {
	// 	logger.Error(err)
	// 	h.flarumErrorJsonify(w, createSimpleFlarumError("Page limit information given error"))
	// 	return
	// }

	// postID, err := strconv.ParseUint(filterData[1], 10, 64)
	// if err != nil {
	// 	logger.Error(err)
	// 	h.flarumErrorJsonify(w, createSimpleFlarumError(fmt.Sprintf("Unable to parse comment information: %s", filterData)))
	// 	return
	// }
	// comment := model.SQLGetCommentByID(   redisDB, postID, scf.TimeZone)
	// if comment.UserName != filterData[0] {
	// 	logger.Warningf("User and comment information do not match: %s", filterData)
	// }
	coreData := flarum.NewCoreData()
	apiDoc := &coreData.APIDocument // Note: this returns a pointer

	apiDoc.Links["first"] = scf.MainDomain + model.FlarumAPIPath + "/users?" +
		fmt.Sprintf("filter%%5Bq%%5D=%s&page%%5Blimit%%5D=%s", url.QueryEscape(_filter), _pageLimit)
		// fmt.Sprintf("filter%%5Bq%%5D=%s%%23%d+&page%%5Blimit%%5D=%d", comment.UserName, comment.ID, pageLimit)

	h.jsonify(w, apiDoc)
	return
}

// FlarumPosts get comments
func FlarumPosts(w http.ResponseWriter, r *http.Request) {
	ctx := GetRetContext(r)
	logger := ctx.GetLogger()
	h := ctx.h

	parm := r.URL.Query()
	_userID := parm.Get("filter[user]")
	_authorID := parm.Get("filter[author]")
	_disscussionID := parm.Get("filter[discussion]")
	// _type := parm.Get("filter[type]")
	_limit := parm.Get("page[limit]")
	_ids := parm.Get("filter[id]")
	// _sort := parm.Get("sort")
	_near := parm.Get("page[near]")
	_postID := ""

	redisDB := h.App.RedisDB
	inAPI := ctx.inAPI

	var limit uint64
	var err error
	var user model.User
	var coreData flarum.CoreData

	if len(_limit) > 0 {
		limit, err = strconv.ParseUint(_limit, 10, 64)
		if err != nil {
			return
		}
	}
	limit = 20
	var rf replyFilter

	if _userID != "" {
		user, err = model.SQLUserGet(h.App.GormDB, _userID)
		if user.ID == 0 || err != nil {
			h.flarumErrorJsonify(w, createSimpleFlarumError("Can't get the user for: "+_userID+err.Error()))
			return
		}

		rf = replyFilter{
			FT:    eUserPost,
			UID:   user.ID,
			Limit: limit,
		}

	} else if _authorID != "" {
		user, err = model.SQLUserGetByName(h.App.GormDB, _authorID)
		if user.ID == 0 || err != nil {
			h.flarumErrorJsonify(w, createSimpleFlarumError("Can't get the user for: "+_userID+err.Error()))
			return
		}

		rf = replyFilter{
			FT:    eUserPost,
			UID:   user.ID,
			Limit: limit,
		}

	} else if _disscussionID != "" {
		aid, err := strconv.ParseUint(_disscussionID, 10, 64)
		if err != nil {
			logger.Error("Can't get discussion id for ", _disscussionID)
			h.flarumErrorJsonify(w, createSimpleFlarumError("Can't get the article for: "+_disscussionID+err.Error()))
			return
		}
		article, err := model.SQLArticleGetByID(h.App.GormDB, redisDB, aid)
		if err != nil {
			logger.Error("Can't get discussion id for ", aid)
			h.flarumErrorJsonify(w, createSimpleFlarumError("Can't get discussion for: "+_disscussionID+err.Error()))
			return
		}

		rf = replyFilter{
			FT:    eArticle,
			AID:   aid,
			Limit: article.CommentCount,
		}

		if _near != "" {
			near, err := strconv.ParseUint(_near, 10, 64)
			if err != nil {
				logger.Error("Can't get discussion id for ", _near)
				h.flarumErrorJsonify(w, createSimpleFlarumError("Can't get the page[near] for: "+_near+err.Error()))
				return
			}
			rf.NearNumber = near
		}

	} else if _ids != "" {
		postIds := strings.Split(_ids, ",")
		var _ids64 []uint64
		for _, _id := range postIds {
			_id64, err := strconv.ParseUint(_id, 10, 64)
			if err != nil {
				logger.Error("Can't get post id for", _id)
				continue
			}
			_ids64 = append(_ids64, _id64)
		}
		rf = replyFilter{
			FT:  ePosts,
			IDS: _ids64,
		}
	} else if _postID, err = h.safeGetParm(r, "cid"); _postID != "" && err == nil {
		postID, err := strconv.ParseUint(_postID, 10, 64)
		if err != nil {
			logger.Error("Can't get post id for ", _postID)
			h.flarumErrorJsonify(w, createSimpleFlarumError("Can't get postId for: "+err.Error()))
		}
		rf = replyFilter{
			FT:  ePost,
			CID: postID,
		}
	} else {
		logger.Warning("Can't process post api")
	}

	coreData, err = createFlarumPostAPIDoc(ctx, h.App.GormDB, redisDB, *h.App.Cf, rf, ctx.h.App.Cf.Site.TimeZone)
	if err != nil {
		h.flarumErrorJsonify(w, createSimpleFlarumError("Get api doc error"+err.Error()))
		return
	}

	// fmt.Println(userID, _type, _limit, _sort, limit, user, comments)
	// If it is an API request, return directly
	if inAPI {
		h.jsonify(w, coreData.APIDocument)
		return
	}
}

// FlarumPostsUtils some operations on comments
func FlarumPostsUtils(w http.ResponseWriter, r *http.Request) {
	var err error
	ctx := GetRetContext(r)
	logger := ctx.GetLogger()
	h := ctx.h
	_cid := pat.Param(r, "cid")
	cid, err := strconv.ParseUint(_cid, 10, 64)
	if err != nil {
		h.flarumErrorJsonify(w, createSimpleFlarumError("cid type error"))
		return
	}

	redisDB := h.App.RedisDB
	cobj, err := model.SQLCommentByID(h.App.GormDB, redisDB, cid, h.App.Cf.Site.TimeZone)
	if err != nil {
		h.flarumErrorJsonify(w, createSimpleFlarumError("Unable to get comment"))
		return
	}

	// User performed actions
	type CommentUtils struct {
		Data struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes map[string]interface{}
		} `json:"data"`
	}

	commentUtils := CommentUtils{}
	err = json.NewDecoder(r.Body).Decode(&commentUtils)
	if err != nil {
		h.flarumErrorJsonify(w, createSimpleFlarumError("json Decode err:"+err.Error()))
		return
	}

	if val, ok := commentUtils.Data.Attributes["isLiked"]; ok {
		cobj.DoLike(h.App.GormDB, redisDB, ctx.currentUser, val.(bool))
	}
	if val, ok := commentUtils.Data.Attributes["content"]; ok {
		logger.Errorf("Didn't apply the method to update the comment, Update content to ", val.(string))
		h.flarumErrorJsonify(w, createSimpleFlarumError("Didn't support modify comment"))
		return
	}

	rf := replyFilter{
		FT:  ePost,
		CID: cobj.ID,
		AID: cobj.AID,
	}

	coreData, err := createFlarumPostAPIDoc(ctx, h.App.GormDB, redisDB, *h.App.Cf, rf, ctx.h.App.Cf.Site.TimeZone)
	if err != nil {
		h.flarumErrorJsonify(w, createSimpleFlarumError("Get api doc error"+err.Error()))
		return
	}

	if ctx.inAPI {
		h.jsonify(w, coreData.APIDocument)
		return
	}

	h.flarumErrorJsonify(w, createSimpleFlarumError("This interface is only used in API"))
}
