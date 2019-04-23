package main

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

type cmdShardsVelocity struct {
	Selector string `long:"selector" short:"l" description:"Label Selector query to filter on"`
	Interval int    `long:"interval" short:"i" description:"Sample for \"interval\" seconds"`
	Count    int    `long:"count" short:"c" description:"Stop after sampling \"count\" times"`
}

func init() {
	_ = mustAddCmd(cmdShards, "velocity", "Measure journal read and write rates", `
Measure the read and write rates of each journal read by each shard

TODO: WRITEME
`, &cmdShardsVelocity{})
}

func (cmd *cmdShardsVelocity) Execute([]string) error {
	startup()

	//var resp = listShards(cmd.Selector)

	var table = tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"ID", "Journal", "Read bps", "Write bps"})

	var res = []velocityResp{
		{"0000", "my/journal/a", 0, 10, 100, 200, 5},
		{"0000", "my/journal/b", 0, 3255, 1235, 3452, 5},
		{"0001", "my/journal/c", 0, 352, 100, 4532, 5},
		{"0002", "my/journal/d", 0, 34, 100, 5421, 5},
	}

	for _, j := range res {
		table.Append([]string{
			j.ID,
			j.Journal,
			fmt.Sprintf("%d", j.ReadRate()),
			fmt.Sprintf("%d", j.WriteRate()),
		})
	}
	table.Render()

	return nil
}

type velocityResp struct {
	ID                               string
	Journal                          string
	StartReadOffset, EndReadOffset   int
	StartWriteOffset, EndWriteOffset int
	Interval                         int
}

func (r velocityResp) ReadRate() int {
	return (r.EndReadOffset - r.StartReadOffset) / r.Interval
}

func (r velocityResp) WriteRate() int {
	return (r.EndWriteOffset - r.StartWriteOffset) / r.Interval
}
