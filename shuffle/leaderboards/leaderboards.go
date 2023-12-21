package leaderboards

import (
	"log"
	"sort"
	"time"

	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/utility"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	usersCache              []database.UserProfile
	usersCacheByTotalVolume []database.UserProfile
	usersCacheByTodayVolume []database.UserProfile
)

func Routine() {
	go func() {
		routine()
	}()
}

func routine() {
	for {
		err := database.Find("users", bson.M{}, &usersCache)
		if err != nil {
			log.Println(err)
			continue
		}
		usersCacheByTotalVolume = make([]database.UserProfile, len(usersCache))
		usersCacheByTodayVolume = make([]database.UserProfile, len(usersCache))
		copy(usersCacheByTotalVolume, usersCache)
		copy(usersCacheByTodayVolume, usersCache)
		sort.Slice(usersCacheByTotalVolume, func(i, j int) bool {
			return usersCacheByTotalVolume[i].Stats.TotalVolume > usersCacheByTotalVolume[j].Stats.TotalVolume
		})

		for i, user := range usersCacheByTodayVolume {
			if user.Stats.Volumes == nil {
				user.Stats.Volumes = make(map[string]uint64)
			}
			if _, ok := user.Stats.Volumes[utility.FormatDate(time.Now())]; !ok {
				user.Stats.Volumes[utility.FormatDate(time.Now())] = 0
				usersCacheByTodayVolume[i] = user
			}
		}

		sort.Slice(usersCacheByTodayVolume, func(i, j int) bool {
			return usersCacheByTodayVolume[i].Stats.Volumes[utility.FormatDate(time.Now())] > usersCacheByTodayVolume[j].Stats.Volumes[utility.FormatDate(time.Now())]
		})

		time.Sleep(time.Second * 10)
	}
}

func TotalVolumeUsers() []database.UserProfile {
	maxi := 20
	if maxi > len(usersCacheByTotalVolume) {
		maxi = len(usersCacheByTotalVolume)
	}

	return usersCacheByTotalVolume[:maxi]
}

func TodayVolumeUsers() []database.UserProfile {
	maxi := 20
	if maxi > len(usersCacheByTodayVolume) {
		maxi = len(usersCacheByTodayVolume)
	}

	return usersCacheByTodayVolume[:maxi]
}
