package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/glvr182/appie"
	"github.com/microcosm-cc/bluemonday"
)

func main() {
	fmt.Println("Hello, world.")

	// Create a new Discord session using the provided bot token.
	token := os.Getenv("BOT_TOKEN")
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!ah info ") {
		args := strings.Fields(m.Content)
		productID, err := strconv.Atoi(args[2])

		product, err := appie.ProductByID(productID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Product not found")
		}

		albertEmbed := new(discordgo.MessageEmbed)

		albertEmbed.Title = product.Title
		albertEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: product.Images[0].URL}

		// Remove all html tags from summary and set as description.
		albertEmbed.Description = bluemonday.StrictPolicy().Sanitize(product.Summary)

		// Fields with price and availablity info.
		albertEmbed.Fields = append(albertEmbed.Fields, &discordgo.MessageEmbedField{Name: "Brand", Value: product.Brand, Inline: true})
		albertEmbed.Fields = append(albertEmbed.Fields, &discordgo.MessageEmbedField{Name: "Price", Value: fmt.Sprintf("€ %.2f", product.Price.Now), Inline: true})
		albertEmbed.Fields = append(albertEmbed.Fields, &discordgo.MessageEmbedField{Name: "Available", Value: strings.Title(fmt.Sprintf("%t", product.Orderable)), Inline: true})

		// If product is in sale add more info
		if product.Control.Theme == "bonus" {
			albertEmbed.Fields = append(albertEmbed.Fields, &discordgo.MessageEmbedField{Name: "Bonus", Value: "Yes", Inline: true})
			albertEmbed.Fields = append(albertEmbed.Fields, &discordgo.MessageEmbedField{Name: "Bonus Type", Value: product.Shield.Text, Inline: true})

			// Set date in correct format.
			t, _ := time.Parse("2006-01-02", product.Discount.EndDate)
			albertEmbed.Fields = append(albertEmbed.Fields, &discordgo.MessageEmbedField{Name: "Bonus end", Value: t.Format("Mon, 02 Jan 2006"), Inline: true})
		} else {
			albertEmbed.Fields = append(albertEmbed.Fields, &discordgo.MessageEmbedField{Name: "Bonus", Value: "No", Inline: true})
		}

		// Send message to channel
		s.ChannelMessageSendEmbed(m.ChannelID, albertEmbed)
	}
}
