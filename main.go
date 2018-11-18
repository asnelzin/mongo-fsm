package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"sync"

	"github.com/asnelzin/mongo-fsm/store"
	"github.com/go-pkgz/mongo"
	"github.com/jessevdk/go-flags"
)

type Opts struct {
	MongoURL string `long:"url" env:"MONGO_URL" description:"mongo url" required:"true"`
	MongoDB  string `long:"db" env:"MONGO_DB" default:"test" description:"mongo database"`
}

func main() {
	var opts Opts
	p := flags.NewParser(&opts, flags.Default)

	if _, err := p.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	log.SetFlags(log.Ldate | log.Ltime)
	log.SetOutput(os.Stdout)

	m, err := mongo.NewServerWithURL(opts.MongoURL, 5*time.Second)
	if err != nil {
		log.Fatalf("could not connect to mongo cluster: %v", err)
	}

	conn := mongo.NewConnection(m, opts.MongoDB, "streams")
	storeEngine := store.NewMongo(conn)

	streamID := "5be9ed414bb89555c1f9facc"

	errors := 0
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		err := storeEngine.SetStateAdmin(streamID, store.Active)
		if err != nil {
			log.Fatalf("could not set initial state: %v", err)
			break
		}

		time.Sleep(100 * time.Millisecond)

		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := storeEngine.SetState(streamID, store.Interrupted)
			if err != nil && err == store.ConcurrentError {
				errors++
			}
		}()

		go func() {
			defer wg.Done()
			_, err := storeEngine.SetState(streamID, store.Finished)
			if err != nil && err == store.ConcurrentError {
				errors++
			}
		}()

		wg.Wait()

		stream, err := storeEngine.GetStream(streamID)
		if err != nil {
			log.Fatalf("[ERROR] stream is not found: %v", err)
		}
		if (stream.State != store.Finished) && (stream.State != store.Interrupted) {
			log.Fatalf("[ERROR] stream is not expected state: %s", stream.State)
		}
	}

	fmt.Printf("Test is passed. Concurrent error rate is: %d / 30\n", errors)
}
