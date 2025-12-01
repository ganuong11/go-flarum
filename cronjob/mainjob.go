package cronjob

import (
	"time"

	"github.com/go-redis/redis/v7"

	"github.com/corvofeng/go-flarum/model"
	"github.com/corvofeng/go-flarum/system"

	logging "github.com/op/go-logging"
)

// BaseHandler I do not know
type BaseHandler struct {
	App *system.Application
}

// MainCronJob my job
func (h *BaseHandler) MainCronJob() {

	// cntDB := h.App.Db
	// scf := h.App.Cf.Site
	logger := h.App.Logger
	redisDB := h.App.RedisDB
	// tick1 := time.Tick(3600 * time.Second)
	// tick2 := time.Tick(120 * time.Second)
	// tick3 := time.Tick(30 * time.Minute)
	// tick4 := time.Tick(31 * time.Second)
	// tick5 := time.Tick(1 * time.Minute)
	// tickRefreshOrder := time.Tick(3 * time.Second)
	// daySecond := int64(3600 * 24)

	// Refresh the sorting data in Redis to the database every 3 hours
	tickResortRankMap := time.Tick(3 * time.Hour)

	// Flush click counts to database every hour
	tickStoreHitToMySQL := time.Tick(1 * time.Hour)

	// Refresh sorting every ten minutes
	// tickRefreshOrder := time.Tick(10 * time.Minute)

	logger.Info("Start cron job")
	syncWithMySQL(logger, redisDB)
	refreshRankMap(logger)

	for {
		select {
		case <-tickStoreHitToMySQL:
			syncWithMySQL(logger, redisDB)
		case <-tickResortRankMap:
			refreshRankMap(logger)
		}
	}
}

// refreshRankMap refresh the sorting data stored in redis
func refreshRankMap(logger *logging.Logger) {
	logger.Info("===== Start refresh rank map =====")
	model.TimelyResort()
	logger.Info("=====  End  refresh rank map =====")
}

// syncWithMySQL synchronize the statistical data in Redis to mysql
func syncWithMySQL(logger *logging.Logger, redisDB *redis.Client) {
	logger.Info("===== start sync hits with the mysql =====")
	// data, _ := redisDB.HGetAll("article_views").Result()
	// for aid, clickCnt := range data {
	// _aid, _ := strconv.ParseUint(aid, 10, 64)
	// _clickCnt, _ := strconv.ParseUint(clickCnt, 10, 64)
	// logger.Debugf("Set %4d with %d", _aid, _clickCnt)
	// model.SQLArticleSetClickCnt(   _aid, _clickCnt)
	// }
	logger.Info("=====  end  sync hits with the mysql =====")
}
