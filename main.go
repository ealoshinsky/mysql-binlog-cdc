package main

import (
	"context"
	"github.com/go-mysql-org/go-mysql/replication"
	"log"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.TODO(), syscall.SIGINT, syscall.SIGHUP, syscall.SIGABRT)
	defer cancel()

	cfg := replication.BinlogSyncerConfig{
		ServerID: 249580531, // for get server ID run in mysql-master: SELECT @@server_id
		Flavor:   "mysql",
		Host:     "127.0.0.1",
		Port:     3336,
		User:     "root",
		Password: "root",
		//RawModeEnabled: true,
	}
	syncer := replication.NewBinlogSyncer(cfg)

	stream, err := syncer.StartSync(syncer.GetNextPosition())
	if err != nil {
		log.Fatalln("Error start sync:", err)
	}

	pipe := make(chan any)
	go func(ctx context.Context, pipe chan any) {
		heartbeat := time.Tick(300 * time.Millisecond)
		for {
			select {
			case <-heartbeat:
				{
					if event, err := stream.GetEvent(ctx); err != nil {
						log.Println("Error on event stream:", err)
					} else {
						pipe <- event
					}
				}
			}
		}
	}(ctx, pipe)

	go eventParser(ctx, pipe)
	<-ctx.Done()
}

func eventParser(ctx context.Context, event chan any) {
	for {
		select {
		case e := <-event:
			{
				log.Println(e)
			}
		}
	}
}
