package process

// func TestProcessRun(t *testing.T) {
// 	url := "https://pub001.hg.network/rpc"
// 	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancle()
// 	c, err := rpc.DialContext(ctx, url)
// 	if err != nil {
// 		t.Fail()
// 	}
// 	f := fetch.NewFetcher(c, 5, 5, 5, true, false, 2, 3, 0, nil)

// 	p, err := NewProcess(f)
// 	if err != nil {
// 		t.Fatalf("c process err:%v", err)
// 	}
// 	p.Run(9071816, 0)
// 	time.Sleep(5 * time.Second)
// 	p.fetcher.Stop()
// }
