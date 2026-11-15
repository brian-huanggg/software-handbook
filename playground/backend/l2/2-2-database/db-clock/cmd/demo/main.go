// demo 印出文章用到的實測結果。先執行 make up。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	dbclock "handbook/playground/backend/db-clock"
)

func main() {
	ctx := context.Background()
	db, err := dbclock.Open(dbclock.DefaultDSN)
	if err != nil {
		log.Fatalf("資料庫未啟動（先執行 make up）：%v", err)
	}
	defer db.Close()
	if err := dbclock.Reset(ctx, db); err != nil {
		log.Fatal(err)
	}

	nodes := []dbclock.Node{{Name: "A", Skew: 0}, {Name: "B", Skew: 90 * time.Second}, {Name: "C", Skew: -90 * time.Second}}

	fmt.Println("== 應用程式時間：A 建立 60 秒的短網址，三台各自判斷")
	must(dbclock.CreateApp(ctx, db, nodes[0], "app", 60*time.Second))
	for _, n := range nodes {
		ok, err := dbclock.IsActiveApp(ctx, db, n, "app")
		must(err)
		fmt.Printf("節點 %s（時鐘偏移 %+v）：有效=%v\n", n.Name, n.Skew, ok)
	}

	fmt.Println("== 資料庫時間：同一條規則，只有一個答案")
	must(dbclock.CreateDB(ctx, db, "db", 60*time.Second))
	ok, err := dbclock.IsActiveDB(ctx, db, "db")
	must(err)
	fmt.Printf("任何節點：有效=%v\n", ok)

	fmt.Println("== 同一個交易裡的三個「現在」（每次間隔 300ms）")
	cs, err := dbclock.ReadClocks(ctx, db, 3, 300*time.Millisecond)
	must(err)
	base := cs[0].TxStart
	for i, c := range cs {
		fmt.Printf("第 %d 次：now()=+%v  statement=+%v  clock_timestamp()=+%v\n", i+1,
			c.TxStart.Sub(base).Round(time.Millisecond),
			c.Statement.Sub(base).Round(time.Millisecond),
			c.Wall.Sub(base).Round(time.Millisecond))
	}

	fmt.Println("== timestamp 與 timestamptz：UTC 寫入、Asia/Taipei 讀出")
	plain, tz, err := dbclock.StoredAs(ctx, db, "UTC", "Asia/Taipei")
	must(err)
	fmt.Printf("兩欄差 %.0f 秒（%.1f 小時）\n", plain-tz, (plain-tz)/3600)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
