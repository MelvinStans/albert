package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/glvr182/appie"
	"github.com/melvin1567/albert/bot"
	"github.com/melvin1567/albert/monitor"
)

func main() {
	var wg sync.WaitGroup
	link := make(chan appie.Product, 0)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	mon, err := monitor.New(link)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		wg.Add(1)
		defer func() {
			wg.Done()
			log.Println("Monitor stopped")
		}()
		if err := mon.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	bot, err := bot.New(link, mon, os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		wg.Add(1)
		defer func() {
			wg.Done()
			log.Println("Discord Bot stopped")
		}()
		if err := bot.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("Everything is now running.  Press CTRL-C to exit.")

	<-sc

	if err := mon.Stop(); err != nil {
		log.Fatal(err)
	}

	if err := bot.Stop(); err != nil {
		log.Fatal(err)
	}

	wg.Wait()
	log.Println("Program stopped")
}
