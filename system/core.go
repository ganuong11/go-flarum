package system

import (
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	"net/url"
	"strings"

	"database/sql"

	"github.com/corvofeng/go-flarum/model"
	"github.com/corvofeng/go-flarum/util"

	"github.com/gorilla/securecookie"
	logging "github.com/op/go-logging"
	"github.com/weint/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	ormLogger "gorm.io/gorm/logger"

	"github.com/go-redis/redis/v7"
)

// Application Application database and external services
type Application struct {
	Cf      *model.AppConf
	RedisDB *redis.Client
	MySQLdb *sql.DB
	// MongoDB *mongo.Client
	Sc *securecookie.SecureCookie
	// QnZone  *storage.Zone
	Logger *logging.Logger
	Rand   *rand.Rand // responsible for random number generation
	GormDB *gorm.DB
}

// LoadConfig Initialize program configuration from file
func LoadConfig(filename string) *config.Engine {
	c := &config.Engine{}
	err := c.Load(filename)
	logger := util.GetLogger()
	if err != nil {
		logger.Error("Failed to read configuration file:", err)
	}
	return c
}

// Init, connect to database
func (app *Application) Init(c *config.Engine, currentFilePath string) {
	// .. version_changed: 2019-11-09
	// Add redis, currently redis is only used for caching data, theoretically cannot contain data structures

	mcf := &model.MainConf{}
	c.GetStruct("Main", mcf)
	logger := util.GetLogger()
	app.Logger = logger

	app.Rand = rand.New(rand.NewSource(time.Now().Unix())) // Responsible for handling random numbers

	// check domain
	if strings.HasPrefix(mcf.Domain, "http") {
		dm, err := url.Parse(mcf.Domain)
		if err != nil {
			logger.Fatal("domain fmt err", err)
		}
		mcf.Domain = dm.Host
	} else {
		mcf.Domain = strings.Trim(mcf.Domain, "/")
	}

	scf := &model.SiteConf{}
	c.GetStruct("Site", scf)
	scf.GoVersion = runtime.Version()
	fMd5, _ := util.HashFileMD5(currentFilePath)
	scf.MD5Sums = fMd5
	scf.MainDomain = strings.Trim(scf.MainDomain, "/")
	if scf.TimeZone < -12 || scf.TimeZone > 12 {
		scf.TimeZone = 0
	}
	if scf.UploadMaxSize < 1 {
		scf.UploadMaxSize = 1
	}
	scf.UploadMaxSizeByte = int64(scf.UploadMaxSize) << 20

	app.Cf = &model.AppConf{mcf, scf}

	logger.Debugf("Get redis db url: %s", mcf.RedisURL)
	opt, err := redis.ParseURL(mcf.RedisURL)
	if err != nil {
		panic(err)
	}
	rdsClient := redis.NewClient(opt)
	pong, err := rdsClient.Ping().Result()
	if err != nil {
		logger.Errorf("Connect redis error, %s", err)
		return
	}
	logger.Debug(pong, err)

	logger.Debugf("Get mongo db url: %s", mcf.MongoURL)
	logger.Debugf("Get S3 config: %+v", mcf.S3Config)
	// mongoClient, err := mongo.NewClient(options.Client().ApplyURI(mcf.MongoURL))
	// if err != nil {
	// 	logger.Errorf("Connect mongo error, %s", err)
	// 	return
	// }

	// logger.Debugf("Get mysql db url: %s", mcf.MySQLURL)
	// sqlDb, err := sql.Open("mysql", mcf.MySQLURL)
	// if err != nil {
	// 	logger.Errorf("Connect mysql error, %s", err)
	// 	return
	// }
	// sqlDb.SetConnMaxLifetime(time.Minute * 10)

	app.RedisDB = rdsClient
	ormLogLevel := ormLogger.Silent
	if mcf.Debug {
		ormLogLevel = ormLogger.Info
	}

	gormConfig := gorm.Config{
		Logger: ormLogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			ormLogger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  ormLogLevel, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      false,       // Don't include params in the SQL log
				Colorful:                  false,       // Disable color
			},
		),
	}
	if mcf.DB == "mysql" {
		logger.Debugf("Get mysql db url: %s", mcf.MySQLURL)
		gormConfig.Dialector = mysql.New(mysql.Config{
			DSN:                       mcf.MySQLURL, // DSN data source name
			DefaultStringSize:         256,          // Default length of string type fields
			DisableDatetimePrecision:  true,         // Disable datetime precision
			DontSupportRenameIndex:    true,         // Rename index not supported
			DontSupportRenameColumn:   true,         // Rename column not supported
			SkipInitializeWithVersion: false,        // Automatically configure based on current MySQL version
		})
	} else if mcf.DB == "postgres" {
		logger.Debugf("Get postgres db url: %s", mcf.PostgresURL)
		gormConfig.Dialector = postgres.New(postgres.Config{
			DSN:                  mcf.PostgresURL, // DSN data source name
			PreferSimpleProtocol: true,            // disables implicit prepared statement usage
		})
	} else {
		logger.Fatalf("Unsupported database type: %s", mcf.DB)
	}

	app.GormDB, err = gorm.Open(&gormConfig)
	util.CheckError(err, "gorm open error")
	app.Sc = securecookie.New(
		[]byte(app.Cf.Main.SCHashKey),
		[]byte(app.Cf.Main.SCBlockKey),
	)
}

func (app *Application) CanServeAdmin() bool {
	return app.Cf.Main.CanServeAdmin
}

// Close Clean up program connections
func (app *Application) Close() {
	if app.RedisDB != nil {
		app.RedisDB.Close()
		app.RedisDB = nil
	}
	app.Logger.Info("db closed")
}
