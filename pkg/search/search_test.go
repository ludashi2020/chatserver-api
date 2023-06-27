/*
 * @Author: cloudyi.li
 * @Date: 2023-06-01 08:53:25
 * @LastEditTime: 2023-06-27 13:50:55
 * @LastEditors: cloudyi.li
 * @FilePath: /chatserver-api/pkg/search/search_test.go
 */
package search

import (
	"chatserver-api/pkg/cache"
	"chatserver-api/pkg/config"
	"chatserver-api/pkg/logger"
	"context"
	"fmt"
	"testing"
)

func Test_customSearch(t *testing.T) {
	c := config.Load("../../configs/config.yml")
	logger.InitLogger(&c.LogConfig, c.AppName)
	cache.InitRedis(c.RedisConfig)

	type args struct {
		query    string
		classify string
	}
	tests := []struct {
		name string
		args args
		// wantResultstr string
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				query:    "",
				classify: "News",
			},
		},
		// {
		// 	name: "test1",
		// 	args: args{
		// 		query: "今天有什么新闻",
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResultstr, err := CustomSearch(context.Background(), tt.args.query, tt.args.classify)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(gotResultstr)
		})
	}
}
