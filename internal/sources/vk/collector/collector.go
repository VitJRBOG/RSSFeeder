package collector

import (
	"fmt"
	"strings"
	"time"

	rss "github.com/VitJRBOG/RSSMaker/internal/rss"
	vkapi "github.com/VitJRBOG/RSSMaker/internal/sources/vk/api"
)

func ComposeRSS(community vkapi.Community, wallPosts []vkapi.WallPost) (rss.RSS, error) {
	var r rss.RSS

	r = parseCommunityInfo(r, community)

	r, err := parseWallPosts(r, wallPosts)
	if err != nil {
		return rss.RSS{}, err
	}

	return r, nil
}

func parseCommunityInfo(r rss.RSS, community vkapi.Community) rss.RSS {
	r.Channel.Title = community.Name
	r.Channel.Link = "https://vk.com/" + community.ScreenName
	r.Channel.Description = strings.ReplaceAll(community.Description, "\n", "<br>")

	return r
}

func parseWallPosts(r rss.RSS, wallPosts []vkapi.WallPost) (rss.RSS, error) {
	for _, wallPost := range wallPosts {
		var rssItem rss.Item

		rssItem.Title = getWallPostTitle(wallPost.Text)
		rssItem.Description = strings.ReplaceAll(wallPost.Text, "\n", "<br>")

		var err error
		rssItem.Date, err = getDateInReadableFormat(int64(wallPost.Date))
		if err != nil {
			return rss.RSS{}, err
		}

		rssItem.Link = fmt.Sprintf("https://vk.com/wall%d_%d", wallPost.OwnerID, wallPost.ID)
		rssItem = parseWallPostAttachments(rssItem, wallPost)

		r.Channel.Items = append(r.Channel.Items, rssItem)
	}

	return r, nil
}

func parseWallPostAttachments(rssItem rss.Item, wallPost vkapi.WallPost) rss.Item {
	if len(wallPost.Attachments) > 0 {
		for _, attachment := range wallPost.Attachments {
			if attachment.Type != "photo" {
				continue
			}

			rssItem.Description = fmt.Sprintf("<img src=\"%s\">\n%s",
				getMaxSizePhotoURL(attachment.Photo), rssItem.Description)
		}
	}

	return rssItem
}

func getWallPostTitle(text string) string {
	runes := []rune(text)

	if len(runes) == 0 {
		return text
	}

	for i, r := range runes {
		if i == 80 {
			for j := i; j > 0; j-- {
				if runes[j] == ' ' {
					return fmt.Sprintf("%s...", string(runes[:j]))
				}
			}

			return fmt.Sprintf("%s...", string(runes[:77]))
		}

		if r == '\n' {
			return string(runes[:i])
		}
	}

	return text
}

func getDateInReadableFormat(ts int64) (string, error) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		return "", err
	}
	t := time.Unix(ts, 0).In(loc)
	dateFormat := "Mon, Jan 2 2006 15:04:05 -0700"
	return t.Format(dateFormat), nil
}

func getMaxSizePhotoURL(photo vkapi.PhotoAttachment) string {
	maxWidth := 0
	maxHeight := 0
	url := ""
	for _, size := range photo.Sizes {
		if size.Width > maxWidth && size.Height > maxHeight {
			url = size.URL
		}
	}
	return url
}