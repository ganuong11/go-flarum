package model

import (
	"database/sql"
	"fmt"

	"sync"

	"github.com/corvofeng/go-flarum/util"

	"github.com/go-redis/redis/v7"
	"gorm.io/gorm"

	"strconv"
)

// WeightAble is a structure that can get weight
type WeightAble interface {
	GetWeight() uint64
}

// ArticleRankItem records the weight of each topic
type ArticleRankItem struct {
	AID     uint64 `json:"a_id"`
	Weight  uint64
	SQLDB   *sql.DB
	RedisDB *redis.Client
}

// CategoryRankData sorting data under a category
type CategoryRankData struct {
	CID       uint64     `json:"c_id"`
	mtx       sync.Mutex // At the same time, only one goroutine is allowed to manipulate the records of this category
	maxID     uint64     // Database cursor, records the maximum value of currently read data, used when reading new data from the database
	topicData []ArticleRankItem
}

// RankMap TTL map
type RankMap struct {
	m       map[uint64]*CategoryRankData
	mtx     sync.Mutex // At the same time, only one goroutine is allowed to manipulate the map
	GormDB  *gorm.DB
	SQLDB   *sql.DB
	RedisDB *redis.Client
}

func getWeight(rankMap *RankMap, aid uint64) float64 {
	// topic, err := SQLArticleGetByID(rankMap.GormDB, rankMap.SQLDB, rankMap.RedisDB, aid)
	// if util.CheckError(err, "Query post") {
	// 	return 0
	// }
	// return topic.GetWeight(
	// 	rankMap.SQLDB,
	// 	rankMap.RedisDB,
	// )
	// return nil
	return 0
}

var rankMap *RankMap
var rankRedisDB *redis.Client

func cid2Key(cid uint64) string {
	return fmt.Sprintf("rank-category-%d", cid)
}

// TimelyResort refreshes each post's weight in Redis
func TimelyResort() {
	// Refresh topics for all categories
	categoryList, err := SQLGetTags(rankMap.GormDB)
	logger := util.GetLogger()
	if util.CheckError(err, "Get all nodes") {
		return
	}
	categoryList = append(categoryList, Tag{ID: 0, Name: "All nodes"})

	for _, v := range categoryList {
		logger.Debugf("Start refresh category %d(%s)", v.ID, v.Name)

		// Remove invalid posts from Redis
		sqlDataDel, err := sqlGetAllArticleWithCID(v.ID, false)
		if util.CheckError(err, fmt.Sprintf("Get the list of invalid posts under node %d", v.ID)) {
			return
		}
		for _, t := range sqlDataDel {
			_, err := rankRedisDB.ZRem(cid2Key(v.ID), fmt.Sprintf("%d", t.ID)).Result()
			logger.Debug("Delete not active topic", t.ID)
			util.CheckError(err, "Delete invalid posts")
		}

		// Update all valid posts to Redis
		sqlDataAdd, err := sqlGetAllArticleWithCID(v.ID, true)
		if util.CheckError(err, fmt.Sprintf("Get the list of valid posts under node %d", v.ID)) {
			return
		}

		//     First fetch valid IDs from DB
		for _, t := range sqlDataAdd {
			_, err := rankRedisDB.ZAddNX(cid2Key(v.ID), &redis.Z{
				Score:  getWeight(rankMap, t.ID),
				Member: fmt.Sprintf("%d", t.ID)},
			).Result()
			util.CheckError(err, "Update current post")
		}

		// Refresh weights
		rdsData, _ := rankRedisDB.ZRevRange(cid2Key(v.ID), 0, -1).Result()
		for _, topicID := range rdsData {
			aid, _ := strconv.ParseUint(topicID, 10, 64)
			rankRedisDB.ZAddXX(cid2Key(v.ID), &redis.Z{
				Score:  float64(getWeight(rankMap, aid)),
				Member: fmt.Sprintf("%d", aid)},
			)
		}
	}
}

func newRankMap() (m *RankMap) {
	m = &RankMap{m: make(map[uint64]*CategoryRankData)}
	return m
}

// RankMapInit init a ttl map
func RankMapInit(gormDB *gorm.DB, redisDB *redis.Client) {
	rankMap = newRankMap()
	rankMap.GormDB = gormDB
	rankMap.RedisDB = redisDB

	rankRedisDB = redisDB
}

// GetRankMap you can get ttlmap by this.
func GetRankMap() *RankMap {
	return rankMap
}

func min(a, b uint64) uint64 {
	if a <= b {
		return a
	}
	return b
}

// GetTopicListByPageNum finds the ID values of topics through the given page number
func GetTopicListByPageNum(cid uint64, page uint64, limit uint64) []uint64 {
	var retData []uint64

	start := (page - 1) * limit
	end := (page)*limit - 1
	data, _ := rankRedisDB.ZRevRange(cid2Key(cid), int64(start), int64(end)).Result()
	for _, val := range data {
		aid, _ := strconv.ParseUint(val, 10, 64)
		retData = append(retData, aid)
	}
	return retData
}

// AddNewArticleList adds topics to a certain category
func AddNewArticleList(cid uint64, rankItems []ArticleRankItem) {
	m := GetRankMap()
	if _, ok := m.m[cid]; !ok { // At the same time, only one goroutine may operate
		func() {
			m.mtx.Lock()
			defer m.mtx.Unlock()
			if _, ok := m.m[cid]; !ok { // double-check
				m.m[cid] = &CategoryRankData{CID: cid}
			}
		}()
	}

	var maxID uint64
	// fmt.Printf("Add rank item", rankItems)
	for _, d := range rankItems {
		if d.AID > maxID {
			maxID = d.AID
		}

		rankRedisDB.ZAdd(cid2Key(cid), &redis.Z{
			Score:  float64(d.Weight),
			Member: fmt.Sprintf("%d", d.AID)})
	}

	crd := m.m[cid] // categoryRankData
	func() {
		crd.mtx.Lock()
		defer crd.mtx.Unlock()
		crd.topicData = append(crd.topicData, rankItems...) // Directly add, without any processing
		crd.maxID = maxID
	}()
}

// GetCIDArticleMax gets the offset value of the current category
func GetCIDArticleMax(cid uint64) uint64 {
	m := GetRankMap()
	if _, ok := m.m[cid]; ok {
		return m.m[cid].maxID
	}
	return 0
}
