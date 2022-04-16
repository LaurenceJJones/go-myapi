package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type redisSetArgs struct {
	Key   string `form:"key" binding:"required"`
	Value string `form:"value" binding:"required"`
	Exp   int32  `form:"exp"`
}

type repoLicense struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Spdx_id string `json:"spdx_id"`
	Url     string `json:"url"`
}

type repo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Link        string    `json:"html_url"`
	Forks       int32     `json:"forks"`
	Stars       int32     `json:"stargazers_count"`
	Watchers    int32     `json:"watchers_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Language    string    `json:"language"`
	License     repoLicense
}

func getClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func setRedis(Args redisSetArgs) *redis.StatusCmd {
	return getClient().Set(ctx, Args.Key, Args.Value, time.Duration(Args.Exp)*time.Second)
}

func getRedis(Key string) *redis.StringCmd {
	return getClient().Get(ctx, Key)
}

func getRepos() []repo {
	repos := []repo{}
	body, err := getRedis("github").Bytes()
	if err != nil {
		res, err := http.Get("https://api.github.com/users/laurencejjones/repos")
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()
		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}
		setRedis(redisSetArgs{
			Key:   "github",
			Value: string(body),
			Exp:   60 * 60,
		})
	}
	if json.Unmarshal(body, &repos) != nil {
		panic(err)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Forks > repos[j].Forks
	})
	return repos
}

func main() {
	r := gin.Default()
	r.GET("/set", func(ctx *gin.Context) {
		var Args redisSetArgs
		if ctx.ShouldBindQuery(&Args) == nil {
			log.Println(Args)
			setRedis(Args)
			ctx.JSON(http.StatusOK, gin.H{
				"messgage": "Set correctly",
			})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Error",
		})
	})
	r.GET("/github", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, getRepos())
	})
	r.Run()
}
