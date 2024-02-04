package dao

import (
	"container/heap"

	"gofishing-plate/internal/pb"

	"github.com/guogeer/quasar/log"
)

const maxBuildRankItem = 50

type buildRankItem struct {
	Uid        int
	Nickname   string
	Icon       string
	BuildLevel int
	BuildExp   int
}

func (rank *buildRankItem) Score() int {
	return 1000_000*rank.BuildLevel + rank.BuildExp
}

type rankItem interface {
	Score() int
}

type rankHeap []rankItem

func (h rankHeap) Len() int           { return len(h) }
func (h rankHeap) Less(i, j int) bool { return h[i].Score() < h[j].Score() }
func (h rankHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *rankHeap) Push(x any) {
	*h = append(*h, x.(rankItem))
}

func (h *rankHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// 生成建设度排行榜
func GenerateBuildRank() {
	log.Debugf("generate build rank...")

	rs, err := gameDB.Query("select u.uid,u.nickname,u.icon,b.bin from user_info u left join user_bin b on u.uid=b.uid where b.`class`=?", "stat")
	if err != nil {
		log.Error("query build rank error: ", err)
	}
	h := &rankHeap{}
	for rs != nil && rs.Next() {
		item := &buildRankItem{}
		stat := &pb.StatBin{}
		rs.Scan(&item.Uid, &item.Nickname, &item.Icon, PB(stat))
		item.BuildExp = int(stat.BuildExp)
		item.BuildLevel = int(stat.BuildLevel)
		if h.Len() >= maxBuildRankItem && (*h)[0].Score() < item.Score() {
			heap.Pop(h)
		}
		if h.Len() < maxBuildRankItem {
			heap.Push(h, item)
		}
	}
	rank := make([]*buildRankItem, h.Len())
	for i := h.Len() - 1; h.Len() > 0; i-- {
		rank[i] = heap.Pop(h).(*buildRankItem)
		// println(i, rank[i].Score())
	}

	manageDB.Exec("delete from dict where `key`=?", "build_rank")
	manageDB.Exec("insert ignore dict(`key`,`value`) values(?,?)", "build_rank", JSON(rank))
}
