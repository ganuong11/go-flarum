package model

import (
	"time"

	"github.com/go-redis/redis/v7"
)

const (
	// FlarumAPIPath flarum api location
	FlarumAPIPath      = "/api/v1/flarum"
	FlarumAdminPath    = "/admin"
	FlarumExtensionAPI = "/api/extensions" // For website administrators
)

type S3ConfigConf struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
	Region          string
	Path            string
	S3BaseURL       string
}

// MainConf Main configuration
type MainConf struct {
	HTTPPort int
	Domain   string // If https is enabled, this domain is the registered domain, eg: domain.com, www.domain.com

	BaseURL string

	// Database address
	DB          string
	MySQLURL    string
	PostgresURL string
	MongoURL    string
	RedisURL    string

	StaticDir     string
	WebpackDir    string
	LocaleDir     string
	ExtensionsDir string
	ViewDir       string
	UploadDir     string // Upload file directory
	Debug         bool

	ServerName     string
	CanServeAdmin  bool
	CookieSecure   bool
	CookieHttpOnly bool
	OldSiteDomain  string
	TLSCrtFile     string
	TLSKeyFile     string

	S3Config S3ConfigConf // S3 configuration

	// secure cookie needed during initialization
	SCHashKey  string
	SCBlockKey string
}

// SiteConf Site configuration
type SiteConf struct {
	GoVersion  string
	MD5Sums    string
	Name       string
	Desc       string
	AdminEmail string
	MainDomain string // Add URL prefix after uploading image, eg: http://domain.com, http://234.21.35.89:8082

	CDNBaseURL string // Static file CDN address

	MainNodeIDs       string
	TimeZone          int
	HomeShowNum       int
	TitleMaxLen       int
	ContentMaxLen     int
	PostInterval      int
	CommentListNum    int
	CommentInterval   int
	Authorized        bool
	RegReview         bool
	CloseReg          bool
	AutoDataBackup    bool
	AutoGetTag        bool
	UploadSuffix      string
	UploadImgOnly     bool
	UploadImgResize   bool
	AllowSignup       bool
	UploadMaxSize     int
	UploadMaxSizeByte int64

	WelcomeMessage string
	WelcomeTitle   string

	// Google tracking code id
	TrackingCodeID string

	GithubClientID     string
	GithubClientSecret string
}

// AppConf Application configuration file
type AppConf struct {
	Main *MainConf
	Site *SiteConf
}

// SiteInfo Some collection class information of the current site
type SiteInfo struct {
	Days     uint64 // Days created
	UserNum  uint64 // Number of users
	NodeNum  uint64 // Number of nodes
	TagNum   uint64 // Number of tags
	PostNum  uint64 // Number of posts
	ReplyNum uint64 // Number of replies
}

// GetDays Get the number of days from the site creation to now, used for display on the homepage
func GetDays(redisDB *redis.Client) uint64 {

	siteCreateTime, err := redisDB.Get("site_create_time").Uint64()
	if err != nil {
		siteCreateTime = 1557585456 // 2019-05-11 22:37:36 +0800 HKT
	}
	then := time.Unix(int64(siteCreateTime), 0)
	diff := time.Now().UTC().Sub(then)
	return uint64(diff.Hours()/24) + 1
}

// GetSiteInfo Directly get website information
func GetSiteInfo(redisDB *redis.Client) SiteInfo {
	si := SiteInfo{}
	si.Days = GetDays(redisDB)
	// si.UserNum = db.Hsequence("user")
	// si.NodeNum = db.Hsequence("category")
	// si.TagNum = db.Hsequence("tag")
	// si.PostNum = db.Hsequence("article")
	// si.ReplyNum = db.Hget("count", []byte("comment_num")).Uint64()

	return si
}
