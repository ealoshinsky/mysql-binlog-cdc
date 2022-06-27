package main

import (
	"context"
	"fmt"
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
		ServerID: 1, // for get server ID run in mysql-master: SELECT @@server_id
		Flavor:   "mysql",
		Host:     "127.0.0.1",
		Port:     3370,
		User:     "root",
		Password: "Mastermaster123",
		//RawModeEnabled: true,
	}
	syncer := replication.NewBinlogSyncer(cfg)

	log.Println(syncer.GetNextPosition())

	stream, err := syncer.StartSync(syncer.GetNextPosition())
	if err != nil {
		log.Fatalln("Error start sync:", err)
	}

	pipe := make(chan *replication.BinlogEvent)
	go func(ctx context.Context, pipe chan *replication.BinlogEvent) {
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

	go eventParser(ctx, "mt_catalog", pipe)
	<-ctx.Done()
}

func eventParser(ctx context.Context, target string, event chan *replication.BinlogEvent) {
	for {

		select {
		case e := <-event:
			{
				switch e.Event.(type) {
				//case *replication.RotateEvent:
				//	{
				//		onRotateEvent(e)
				//	}
				case *replication.RowsEvent:
					{
						onRowsEvent(target, e)
					}
					//case *replication.TableMapEvent:
					//	{
					//		onTableEvent(e)
					//	}
					//default:
					//	fmt.Printf("%T\n", ev)
				}
			}
		case <-ctx.Done():
			{
				log.Println("Down any connection")
				return
			}

		}
	}
}

func onRowsEvent(target string, event *replication.BinlogEvent) {
	ev := event.Event.(*replication.RowsEvent)

	// Caveat: table may be altered at runtime.
	schema := string(ev.Table.Schema)
	table := string(ev.Table.Table)
	if table == target {
		log.Println("catch on:", fmt.Sprintf("%s.%s", schema, table))
		switch event.Header.EventType {
		case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
			for _, row := range ev.Rows {
				log.Println(row)
			}
		case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
			log.Println("delete-action", len(ev.Rows))
		case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
			log.Println("update-action", len(ev.Rows))
		default:
			log.Printf("%s not supported now", event.Header.EventType)
		}
	}
}

func onRotateEvent(event *replication.BinlogEvent) {
	ev := event.Event.(*replication.RotateEvent)
	log.Println("Header server-id:", event.Header.ServerID)
	log.Println("Header event type", event.Header.EventType)
	log.Println("Event position:", ev.Position)
}

func onTableEvent(event *replication.BinlogEvent) {
	ev := event.Event.(*replication.TableMapEvent)
	log.Println("On tableEvent:", string(ev.Table), string(ev.Schema))

	for i, name := range ev.ColumnName {
		log.Println(i)
		log.Println(name)
	}
}
