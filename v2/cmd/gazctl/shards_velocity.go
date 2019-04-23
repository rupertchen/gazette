package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/LiveRamp/gazette/v2/pkg/mainboilerplate"

	"github.com/LiveRamp/gazette/v2/pkg/protocol"

	"github.com/LiveRamp/gazette/v2/pkg/consumer"
	"github.com/olekukonko/tablewriter"
)

// TODO: Add type secondsDurationFlag that can Marshall/Unmarshall
// This would avoid repeated conversion between seconds and nanoseconds

type cmdShardsVelocity struct {
	Selector string `long:"selector" short:"l" description:"Label Selector query to filter on"`
	Interval int    `long:"interval" short:"i" description:"Sample for \"interval\" seconds"`
	Count    int    `long:"count" short:"c" description:"Stop after sampling \"count\" times"`
	rsc      consumer.RoutedShardClient
	rjc      protocol.RoutedJournalClient
}

func init() {
	_ = mustAddCmd(cmdShards, "velocity", "Measure journal read and write rates", `
Measure the read and write rates of each journal read by each shard

TODO: WRITEME
`, newCmdShardsVelocity())
}

func newCmdShardsVelocity() *cmdShardsVelocity {
	var c = &cmdShardsVelocity{}
	var ctx = context.Background()
	c.rsc = shardsCfg.Consumer.RoutedShardClient(ctx)
	c.rjc = shardsCfg.Broker.RoutedJournalClient(ctx)
	return c
}

func (cmd *cmdShardsVelocity) Execute([]string) error {
	startup()

	var resp = listShards(cmd.Selector)

	var res = cmd.measureVelocities(resp.Shards)

	var table = tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"ID", "Journal", "Read bps", "Write bps"})

	for _, j := range res {
		table.Append([]string{
			string(j.ID),
			string(j.Journal),
			fmt.Sprintf("%d", j.ReadRate()),
			fmt.Sprintf("%d", j.WriteRate()),
		})
	}
	table.Render()

	return nil
}

func (cmd *cmdShardsVelocity) measureVelocities(shards []consumer.ListResponse_Shard) []velocityResp {

	// TODO: Use WriteGroup?

	var res []velocityResp
	for _, s := range shards {
		res = append(res, cmd.single(s.Spec)...)
	}

	return []velocityResp{
		{"0000", "my/journal/a", 0, 10, 100, 200, 5},
		{"0000", "my/journal/b", 0, 3255, 1235, 3452, 5},
		{"0001", "my/journal/c", 0, 352, 100, 4532, 5},
		{"0002", "my/journal/d", 0, 34, 100, 5421, 5},
	}
}

func (cmd *cmdShardsVelocity) single(spec consumer.ShardSpec) []velocityResp {
	// TODO: Make stat call against shard
	// TODO: Make read call against journal
	// TODO: build []velocityResp

	var ctx = context.Background()
	var statReq = consumer.StatRequest{Shard: spec.Id}
	var statResp0, statResp1 *consumer.StatResponse
	var err error
	statResp0, err = consumer.StatShard(ctx, cmd.rsc, &statReq)
	mainboilerplate.Must(err, "failed to stat shard", spec.Id)

	time.Sleep(time.Duration(cmd.Interval) * time.Second)

	statResp1, err = consumer.StatShard(ctx, cmd.rsc, &statReq)
	mainboilerplate.Must(err, "failed to stat shard", spec.Id)

	var res []velocityResp
	for _, j := range spec.Sources {
		res = append(res, velocityResp{
			ID:              spec.Id,
			Journal:         j.Journal,
			StartReadOffset: statResp0.Offsets[j.Journal],
			EndReadOffset:   statResp1.Offsets[j.Journal],
			Interval:        5,
		})
	}
	return res
}

type velocityResp struct {
	ID                               consumer.ShardID
	Journal                          protocol.Journal
	StartReadOffset, EndReadOffset   int64
	StartWriteOffset, EndWriteOffset int64
	Interval                         time.Duration
}

func (r velocityResp) ReadRate() int64 {
	return (r.EndReadOffset - r.StartReadOffset) / int64(r.Interval)
}

func (r velocityResp) WriteRate() int64 {
	return (r.EndWriteOffset - r.StartWriteOffset) / int64(r.Interval)
}
