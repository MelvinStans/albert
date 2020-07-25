package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/glvr182/appie"
	"github.com/melvin1567/albert/monitor"
	"github.com/microcosm-cc/bluemonday"
)

var (
	// ErrCreatingSession indicates that something went wrong while creating a bot session
	ErrCreatingSession = errors.New("error while creating the discord session")
	// ErrOpeningSocket indicates that something went wrong while opening a bot session
	ErrOpeningSocket = errors.New("error opening socket")
)

// Bot contains all the information the discord bot needs
type Bot struct {
	conn   *discordgo.Session
	mon    *monitor.Monitor
	in     chan appie.Product
	ctx    context.Context
	cancel context.CancelFunc
	subs   map[int][]string
}

const (
	colorErrorMessage       = 16713984
	colorBonusMessage       = 16735744
	colorSubscribeMessage   = 65417
	colorUnSubscribeMessage = 8978687
	colorInfoMessage        = 0
)

// New creates a new bot instance
func New(in chan appie.Product, mon *monitor.Monitor, token string) (*Bot, error) {
	conn, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, ErrCreatingSession
	}

	b := new(Bot)
	b.conn = conn
	b.mon = mon
	b.in = in
	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.subs = make(map[int][]string, 0)

	conn.AddHandler(b.onMessage)
	conn.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	return b, nil
}

// Run starts the discord bot
func (b *Bot) Run() error {
	if err := b.conn.Open(); err != nil {
		return err
	}

	for {
		select {
		case p := <-b.in:
			for _, channel := range b.subs[p.ID] {
				b.conn.ChannelMessageSendEmbed(channel, createProductEmbed(p))
			}
		case <-b.ctx.Done():
			return nil
		}
	}
}

// Stop stops the bot instance
func (b *Bot) Stop() error {
	b.cancel()
	b.conn.Close()
	return nil
}

func (b *Bot) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "" {
		return
	}

	args := strings.Fields(m.Content)
	if args[0] != "!ah" {
		return
	}

	switch args[1] {
	case "info":
		b.onInfo(s, m.ChannelID, args[2])
	case "subscribe":
		b.onSubscribe(s, m.ChannelID, args[2])
	case "unsubscribe":
		b.onUnsubscribe(s, m.ChannelID, args[2])
	}
}

func (b *Bot) onInfo(s *discordgo.Session, channel, pid string) {
	productID, err := strconv.Atoi(pid)

	product, err := appie.ProductByID(productID)
	if err != nil {
		sendEmbedErrorMessage(s, channel, err)
	}

	// Send message to channel
	s.ChannelMessageSendEmbed(channel, createProductEmbed(product))
}

func (b *Bot) onSubscribe(s *discordgo.Session, channel, pid string) {
	prodid, err := strconv.Atoi(pid)
	if err != nil {
		sendEmbedErrorMessage(s, channel, err)
		return
	}

	if err := b.mon.Watch(prodid); err != nil {
		sendEmbedErrorMessage(s, channel, err)
		return
	}

	if b.subs[prodid] == nil {
		b.subs[prodid] = make([]string, 0)
	}
	b.subs[prodid] = append(b.subs[prodid], channel)

	embed := new(discordgo.MessageEmbed)

	product, err := appie.ProductByID(prodid)
	if err != nil {
		sendEmbedErrorMessage(s, channel, err)
	}
	embed.Title = "Subscribed to " + product.Title
	embed.Color = colorSubscribeMessage
	s.ChannelMessageSendEmbed(channel, embed)
}

func (b *Bot) onUnsubscribe(s *discordgo.Session, channel, pid string) {
	prodid, err := strconv.Atoi(pid)
	if err != nil {
		sendEmbedErrorMessage(s, channel, err)
		return
	}

	if err := b.mon.Unwatch(prodid); err != nil {
		sendEmbedErrorMessage(s, channel, err)
		return
	}

	for i := range b.subs[prodid] {
		if i == prodid {
			remove(b.subs[prodid], i)
			break
		}
	}
	embed := new(discordgo.MessageEmbed)

	product, err := appie.ProductByID(prodid)
	if err != nil {
		sendEmbedErrorMessage(s, channel, err)
	}
	embed.Title = "Unsubscribed to " + product.Title
	embed.Color = colorUnSubscribeMessage
	s.ChannelMessageSendEmbed(channel, embed)
}

func createProductEmbed(product appie.Product) *discordgo.MessageEmbed {

	embed := new(discordgo.MessageEmbed)

	embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: product.Images[0].URL}

	// Fields with price and availablity info.
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Brand", Value: product.Brand, Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Price", Value: fmt.Sprintf("€ %.2f", product.Price.Now), Inline: true})
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Available", Value: strings.Title(fmt.Sprintf("%t", product.Orderable)), Inline: true})

	// If product is in sale add more info
	if product.Control.Theme == "bonus" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Bonus", Value: "Yes", Inline: true})
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Bonus Type", Value: product.Shield.Text, Inline: true})

		// Set date in correct format.
		t, _ := time.Parse("2006-01-02", product.Discount.EndDate)
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Bonus end", Value: t.Format("Mon, 02 Jan 2006"), Inline: true})

		timeStartDate, _ := time.Parse("2006-01-02", product.Discount.StartDate)
		fmt.Println(timeStartDate)
		if !timeStartDate.After(time.Now()) {
			embed.Title = "Nu in de bonus!!"
			embed.Color = colorBonusMessage
		} else {
			embed.Title = "Volgende week in de bonus!!"
			embed.Color = colorBonusMessage
		}
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{Name: "Bonus", Value: "No", Inline: true})

		embed.Title = product.Title
		// Remove all html tags from summary and set as description.
		embed.Description = bluemonday.StrictPolicy().Sanitize(product.Summary)
	}
	return embed
}

func sendEmbedErrorMessage(s *discordgo.Session, channel string, err error) {
	embed := new(discordgo.MessageEmbed)
	embed.Title = "Error!"
	embed.Description = err.Error()
	embed.Color = colorErrorMessage
	s.ChannelMessageSendEmbed(channel, embed)
}

// remove removes the channel at index from the list
func remove(slice []string, i int) []string {
	copy(slice[i:], slice[i+1:])
	return slice[:len(slice)-1]
}
