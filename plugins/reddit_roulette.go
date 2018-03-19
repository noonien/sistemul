package plugins

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/noonien/sistemul/bot"
	"github.com/turnage/graw/reddit"
)

func init() {
	rand.Seed(time.Now().Unix())
}

type RouletteCategory struct {
	Name       string `storm:"id"`
	NSFW       bool
	Subreddits []string
}

type rrPlug struct {
	lurker reddit.Lurker
}

func RegisterRedditRoulette(b *bot.Bot) error {
	var rr rrPlug
	b.RegisterCmd("fun", "rr", rr.roulette)
	b.RegisterCmd("fun", "nsfw", rr.roulette)
	b.RegisterCmd("fun", "rradd", rr.add)
	b.RegisterCmd("fun", "rrdel", rr.remove)

	rs, _ := reddit.NewScript("@Sistemul, a bot by /u/nuunien", 3*time.Second)
	rr.lurker = rs

	var cats []RouletteCategory
	err := b.DB.All(&cats)
	if err != nil {
		return err
	}

	for _, cat := range cats {
		aliasCmd := "rr"
		if cat.NSFW {
			aliasCmd = "nsfw"
		}
		b.RegisterAlias(cat.Name, aliasCmd+" "+cat.Name)
	}

	return nil
}

func (p *rrPlug) add(b *bot.Bot, cmd *bot.Command) error {
	if cmd.Message.Author.Discriminator != "6664" {
		return nil
	}

	args := strings.Fields(cmd.Args)
	if len(args) < 2 || (args[0] == "-nsfw" && len(args) < 3) {
		err := b.Message(cmd.ChannelID, "`usage: !rradd [-nsfw] <category> <subreddits...>`")
		if err != nil {
			return err
		}
		return nil
	}

	var nsfw bool
	if args[0] == "-nsfw" {
		args = args[1:]
		nsfw = true
	}

	category := strings.ToLower(args[0])
	subreddits := args[1:]
	for i := range subreddits {
		subreddits[i] = strings.ToLower(subreddits[i])
	}

	var cat RouletteCategory
	err := b.DB.One("Name", category, &cat)
	if err != nil && err != storm.ErrNotFound {
		return err
	}

	if nsfw {
		cat.NSFW = true
	}

	// new category
	if cat.Name == "" {
		cat.Name = category
		cat.Subreddits = sortedUniqueStrings(subreddits)
		err = b.DB.Save(&cat)
		if err != nil {
			return err
		}

		aliasCmd := "rr"
		if nsfw {
			aliasCmd = "nsfw"
		}
		b.RegisterAlias(category, aliasCmd+" "+category)

		err := b.Messagef(cmd.ChannelID, "`category %s added with subreddits: %s`", category, strings.Join(cat.Subreddits, ", "))
		if err != nil {
			return err
		}

		return nil
	}

	cat.Subreddits = sortedUniqueStrings(append(cat.Subreddits, subreddits...))

	err = b.DB.Save(&cat)
	if err != nil {
		return err
	}

	err = b.Messagef(cmd.ChannelID, "`category %s subreddits: %s`", category, strings.Join(cat.Subreddits, ", "))
	if err != nil {
		return err
	}

	return nil
}

func (p *rrPlug) remove(b *bot.Bot, cmd *bot.Command) error {
	if cmd.Message.Author.Discriminator != "6664" {
		return nil
	}

	args := strings.Fields(cmd.Args)
	if len(args) < 1 {
		err := b.Message(cmd.ChannelID, "`usage: !rrremove <category> [subreddit...]`")
		if err != nil {
			return err
		}
		return nil
	}

	category := strings.ToLower(args[0])
	subreddits := args[1:]
	for i := range subreddits {
		subreddits[i] = strings.ToLower(subreddits[i])
	}

	var cat RouletteCategory
	err := b.DB.One("Name", category, &cat)
	if err != nil {
		if err != storm.ErrNotFound {
			return err
		}

		err := b.Messagef(cmd.ChannelID, "`category %s doesn't exist`", category)
		if err != nil {
			return err
		}

		return nil
	}

	if len(subreddits) == 0 {
		cat.Subreddits = nil
	} else {
		ms := make(map[string]bool)
		for _, sub := range subreddits {
			ms[sub] = true
		}

		for i, sub := range cat.Subreddits {
			if _, ok := ms[sub]; !ok {
				continue
			}

			cat.Subreddits[i] = cat.Subreddits[len(cat.Subreddits)-1]
			cat.Subreddits = cat.Subreddits[:len(cat.Subreddits)-1]
		}

		sort.Strings(cat.Subreddits)
	}

	if len(cat.Subreddits) == 0 {
		err = b.DB.DeleteStruct(&cat)
		if err != nil {
			return err
		}

		b.RemoveAlias(category)

		err := b.Messagef(cmd.ChannelID, "`category %s has been deleted`", category)
		if err != nil {
			return err
		}

		return nil

	}

	err = b.DB.Save(&cat)
	if err != nil {
		return err
	}

	err = b.Messagef(cmd.ChannelID, "`category %s subreddits: %s`", category, strings.Join(cat.Subreddits, ", "))
	if err != nil {
		return err
	}

	return nil

}

func (p *rrPlug) roulette(b *bot.Bot, cmd *bot.Command) error {
	wantNSFW := cmd.Cmd == "nsfw"
	if wantNSFW {
		c, err := b.DG.Channel(cmd.ChannelID)
		if err != nil {
			return err
		}

		if c.Name != "nsfw" {
			return nil
		}
	}

	args := strings.Fields(cmd.Args)
	if len(args) > 1 {
		err := b.Message(cmd.ChannelID, "`usage: !roulette [category]`")
		if err != nil {
			return err
		}
		return nil
	}

	var cat RouletteCategory
	if len(args) == 1 {
		category := strings.ToLower(args[0])
		err := b.DB.One("Name", category, &cat)
		if err != nil {
			if err != storm.ErrNotFound {
				return err
			}

			err := b.Messagef(cmd.ChannelID, "`category %s doesn't exist`", category)
			if err != nil {
				return err
			}

			return nil
		}

		if cat.NSFW != wantNSFW {
			message := fmt.Sprintf("`category %s is NSFW, use !nsfw %s`", cat.Name)
			if wantNSFW {
				message = fmt.Sprintf("`category %s is not SFW, use !rr %s`", cat.Name)
			}

			err := b.Messagef(cmd.ChannelID, message, category)
			if err != nil {
				return err
			}

			return nil
		}
	} else {
		var cats []RouletteCategory
		err := b.DB.Select(q.Eq("NSFW", wantNSFW)).Find(&cats)
		if err != nil {
			return err
		}

		if len(cats) == 0 {
			err := b.Message(cmd.ChannelID, "`no categories found`")
			if err != nil {
				return err
			}
		}

		cat = cats[rand.Intn(len(cats))]
	}

	var post *reddit.Post
	for {
		sub := cat.Subreddits[rand.Intn(len(cat.Subreddits))]

		var err error
		post, err = p.lurker.Thread("/r/" + sub + "/random")
		if err != nil {
			log.Printf("error while getting random post from %s: %v\n", sub, err)
			continue
		}

		if !checkIsImage(post) {
			continue
		}

		break
	}

	nsfwStr := ""
	if wantNSFW {
		nsfwStr = "NSFW "
	}

	err := b.Messagef(cmd.ChannelID, "%s, ia niste %s: %s %s(https://redd.it/%s)", cmd.Message.Author.Mention(), cat.Name, post.URL, nsfwStr, post.ID)
	if err != nil {
		return err
	}

	return nil
}

var imageHosts = []string{"imgur.com", "gfycat.com", "giphy.com", "i.redditmedia.com", "i.redd.it", "media.tumblr.com"}

func checkIsImage(post *reddit.Post) bool {
	linkURL, err := url.Parse(post.URL)
	if err != nil {
		return false
	}

	for _, host := range imageHosts {
		if strings.Contains(linkURL.Host, host) {
			return true
		}
	}

	return false
}
