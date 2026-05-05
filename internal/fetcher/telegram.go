package fetcher

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
"pure/entities"

	gonm "golang.org/x/net/html"
)

type TelegramFetcher struct {
	Channel string
	Host    string
	Limit   int
	SinceID string
	Since   time.Time
	Until   time.Time
}

type TelegramMessage struct {
	ID        string
	Content   string
	HTML      string
	DateTime  time.Time
	Tags      []string
	Reactions []entities.Reaction
}

func NewTelegramFetcher(channel, host string) *TelegramFetcher {
	if host == "" {
		host = "t.me"
	}
	return &TelegramFetcher{
		Channel: channel,
		Host:    host,
		Limit:   0,
		SinceID: "",
	}
}

func NewTelegramFetcherWithOptions(channel, host string, limit int, sinceID string, since, until time.Time) *TelegramFetcher {
	if host == "" {
		host = "t.me"
	}
	return &TelegramFetcher{
		Channel: channel,
		Host:    host,
		Limit:   limit,
		SinceID: sinceID,
		Since:   since,
		Until:   until,
	}
}

func (f *TelegramFetcher) FetchNotes() ([]entities.Note, error) {
	var allNotes []entities.Note
	beforeID := ""

	for {
		url := fmt.Sprintf("https://%s/s/%s", f.Host, f.Channel)
		if beforeID != "" {
			url = fmt.Sprintf("https://%s/s/%s?before=%s", f.Host, f.Channel, beforeID)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch channel: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML: %w", err)
		}

		// Find "Load more" link for pagination
		moreLink := doc.Find(".tme_messages_more").First()
		if moreLink.Length() > 0 {
			beforeID, _ = moreLink.Attr("data-before")
		} else {
			beforeID = ""
		}

var pageNotes []entities.Note

		doc.Find(".tgme_widget_message_wrap").Each(func(i int, s *goquery.Selection) {
			msg := f.parseMessage(s)
			if msg.ID != "" && msg.Content != "" {
				// Skip by ID
				if f.SinceID != "" && msg.ID <= f.SinceID {
					beforeID = ""
					return
				}

				// Time-based filtering
				if !f.Since.IsZero() && msg.DateTime.Before(f.Since) {
					beforeID = ""
					return
				}
				if !f.Until.IsZero() && msg.DateTime.After(f.Until) {
					return // Skip this one but continue to find older ones
				}

				note := entities.Note{
					ID:        msg.ID,
					Content:   msg.Content,
					HTML:      msg.HTML,
					Title:     f.extractTitle(msg.Content),
					CreatedAt: msg.DateTime,
					Tags:      msg.Tags,
					Reactions: msg.Reactions,
				}
				pageNotes = append(pageNotes, note)
			}
		})

		allNotes = append(allNotes, pageNotes...)

		// Check limit
		if f.Limit > 0 && len(allNotes) >= f.Limit {
			break
		}

		// Stop if no more pages
		if beforeID == "" {
			break
		}

		// Small delay to be polite to Telegram
		time.Sleep(500 * time.Millisecond)
	}

	// Apply limit if set
	if f.Limit > 0 && len(allNotes) > f.Limit {
		allNotes = allNotes[:f.Limit]
	}

	return allNotes, nil
}

func (f *TelegramFetcher) parseMessage(s *goquery.Selection) TelegramMessage {
	msg := TelegramMessage{}

	// data-post is on the child .tgme_widget_message element
	msgEl := s.Find(".tgme_widget_message")
	postID, exists := msgEl.Attr("data-post")
	if exists {
		re := regexp.MustCompile(`[^/]+$`)
		msg.ID = re.FindString(postID)
	}

	dateTimeStr := s.Find(".tgme_widget_message_date time").AttrOr("datetime", "")
	if dateTimeStr != "" {
		msg.DateTime, _ = time.Parse(time.RFC3339, dateTimeStr)
	}

	// Build HTML content:多媒体 + 文本内容
	var contentParts []string

	// 1. 回复
	replyHTML := f.extractReply(s)
	if replyHTML != "" {
		contentParts = append(contentParts, replyHTML)
	}

	// 2. 图片
	imagesHTML := f.extractImages(s)
	if imagesHTML != "" {
		contentParts = append(contentParts, imagesHTML)
	}

	// 2.5 转发消息
	forwardHTML := f.extractForward(s)
	if forwardHTML != "" {
		contentParts = append([]string{forwardHTML}, contentParts...)
	}

	// 3. 视频
	videoHTML := f.extractVideo(s)
	if videoHTML != "" {
		contentParts = append(contentParts, videoHTML)
	}

	// 4. 语音/音频
	audioHTML := f.extractAudio(s)
	if audioHTML != "" {
		contentParts = append(contentParts, audioHTML)
	}

	// 5. 链接预览
	linkPreviewHTML := f.extractLinkPreview(s)
	if linkPreviewHTML != "" {
		contentParts = append(contentParts, linkPreviewHTML)
	}

	// 6. 文件
	documentHTML := f.extractDocument(s)
	if documentHTML != "" {
		contentParts = append(contentParts, documentHTML)
	}

	// 7. 贴纸
	stickerHTML := f.extractSticker(s)
	if stickerHTML != "" {
		contentParts = append(contentParts, stickerHTML)
	}

	// 8. 文本内容
	contentSel := s.Find(".tgme_widget_message_text")
	if contentSel.Length() > 0 {
		var buf strings.Builder
		contentSel = contentSel.First()
		for _, n := range contentSel.Nodes {
			if n.FirstChild != nil {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					renderNodes(c, &buf)
				}
			}
		}
		rawHTML := buf.String()
		textContent := html.UnescapeString(rawHTML)
		textContent = f.processContent(textContent, s)
		contentParts = append(contentParts, textContent)
	}

	msg.HTML = strings.Join(contentParts, "\n")
	msg.Content = f.stripHTML(msg.HTML)

	msg.Tags = f.extractTags(s)

	msg.Reactions = f.parseReactions(s)

	return msg
}

func renderNodes(n *gonm.Node, w *strings.Builder) {
	if n == nil {
		return
	}
	if n.Type == gonm.TextNode {
		w.WriteString(n.Data)
		return
	}
	if n.Type != gonm.ElementNode {
		return
	}
	w.WriteString("<")
	w.WriteString(n.Data)
	for _, attr := range n.Attr {
		w.WriteString(" ")
		w.WriteString(attr.Key)
		w.WriteString("=\"")
		w.WriteString(gonm.EscapeString(attr.Val))
		w.WriteString("\"")
	}
	w.WriteString(">")
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		renderNodes(c, w)
	}
	w.WriteString("</")
	w.WriteString(n.Data)
	w.WriteString(">")
}

func (f *TelegramFetcher) processContent(html string, s *goquery.Selection) string {
	content := html

	re := regexp.MustCompile(`(url\(["']?)((https?:)?\/\/)`)
	content = re.ReplaceAllString(content, "${1}/static/${2}")

	s.Find("tg-emoji").Each(func(i int, sel *goquery.Selection) {
		emojiID, _ := sel.Attr("emoji-id")
		if emojiID != "" {
			img := fmt.Sprintf(`<img class="tg-emoji" src="https://t.me/i/emoji/%s.webp" alt="" width="20" height="20" />`, emojiID)
			sel.ReplaceWithHtml(img)
		}
	})

s.Find(".tgme_widget_message_photo_wrap").Each(func(i int, sel *goquery.Selection) {
		style, _ := sel.Attr("style")
		urlMatch := regexp.MustCompile(`url\(["']?(https?://[^"']+)["']?\)`).FindStringSubmatch(style)
		if urlMatch != nil {
			fullURL := urlMatch[1]
			path := strings.TrimPrefix(fullURL, "https://")
			path = strings.TrimPrefix(path, "http://")
			img := fmt.Sprintf(`<img src="/static/%s" alt="Image" loading="lazy" />`, path)
			sel.ReplaceWithHtml(img)
		}
	})

	s.Find(".tgme_widget_message_video_wrap video").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		if src != "" {
			sel.SetAttr("src", "/static/"+src)
		}
		sel.SetAttr("controls", "true")
	})

	s.Find(".tgme_widget_message_voice").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		if src != "" {
			sel.SetAttr("src", "/static/"+src)
		}
		sel.SetAttr("controls", "true")
	})

	s.Find(".tgme_widget_message_forward").Each(func(i int, sel *goquery.Selection) {
		fromName := sel.Find(".tgme_widget_message_forwarded_from_name").Text()
		if fromName == "" {
			fromName = sel.Find(".message_forwarded_from").Text()
		}
		if fromName != "" {
			forwardHTML := fmt.Sprintf(`<div class="forwarded-from">↪ %s</div>`, fromName)
			sel.ReplaceWithHtml(forwardHTML)
		}
	})

	s.Find(".tgme_widget_message_link_preview").Each(func(i int, sel *goquery.Selection) {
		title := sel.Find(".link_preview_title").Text()
		description := sel.Find(".link_preview_description").Text()

		linkHref := ""
		sel.Find(".link_preview_url").Each(func(j int, a *goquery.Selection) {
			if href, ok := a.Attr("href"); ok && strings.HasPrefix(href, "http") {
				linkHref = href
			}
		})
		if linkHref == "" {
			sel.Find("a").Each(func(j int, a *goquery.Selection) {
				if href, ok := a.Attr("href"); ok && strings.HasPrefix(href, "http") {
					linkHref = href
				}
			})
		}
		if linkHref == "" {
			return
		}

		image := sel.Find(".link_preview_image")
		imageStyle, _ := image.Attr("style")
		imageURL := regexp.MustCompile(`url\(["']?([^"']+)["']?\)`).FindStringSubmatch(imageStyle)

		var imgHTML string
		if imageURL != nil {
			imgHTML = fmt.Sprintf(`<img class="link_preview_image" src="/static/%s" alt="%s" />`, imageURL[1], title)
		}

		linkHTML := fmt.Sprintf(`
			<a href="%s" class="link-preview" target="_blank" rel="noopener">
				%s
				<div class="link-preview-content">
					<div class="link-preview-title">%s</div>
					<div class="link-preview-description">%s</div>
				</div>
			</a>
		`, linkHref, imgHTML, title, description)

		sel.ReplaceWithHtml(linkHTML)
	})

	s.Find(".tgme_widget_message_reply").Each(func(i int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		if href != "" {
			sel.SetAttr("href", strings.Replace(href, fmt.Sprintf("/%s/", f.Channel), "/posts/", 1))
		}
	})

	return content
}

func (f *TelegramFetcher) extractReply(s *goquery.Selection) string {
	reply := s.Find(".tgme_widget_message_reply")
	if reply.Length() == 0 {
		return ""
	}

	href, _ := reply.Attr("href")
	if href != "" {
		reply.Find("a").SetAttr("href", strings.Replace(href, fmt.Sprintf("/%s/", f.Channel), "/notes/", 1))
	}

	replyHTML, _ := reply.Html()
	if replyHTML != "" {
		return "<blockquote class=\"note-reply\">" + replyHTML + "</blockquote>"
	}
	return ""
}

func (f *TelegramFetcher) extractImages(s *goquery.Selection) string {
	var images []string

	s.Find(".tgme_widget_message_photo_wrap").Each(func(i int, sel *goquery.Selection) {
		style, _ := sel.Attr("style")
		urlMatch := regexp.MustCompile(`url\(["']?(https?://[^"']+)["']?\)`).FindStringSubmatch(style)
		if urlMatch != nil {
			fullURL := urlMatch[1]

			// Extract dimensions from style
			widthMatch := regexp.MustCompile(`width:\s*(\d+)px`).FindStringSubmatch(style)
			heightMatch := regexp.MustCompile(`height:\s*(\d+)px`).FindStringSubmatch(style)
			paddingMatch := regexp.MustCompile(`padding-top:\s*([\d.]+)%`).FindStringSubmatch(style)

			width := 400
			height := 300

			if widthMatch != nil {
				width, _ = strconv.Atoi(widthMatch[1])
			}
			if heightMatch != nil {
				height, _ = strconv.Atoi(heightMatch[1])
			} else if paddingMatch != nil && widthMatch != nil {
				if pct, err := strconv.ParseFloat(paddingMatch[1], 64); err == nil {
					height = int(float64(width) * pct / 100)
				}
			}

			// Use direct URL (no proxy needed - Telegram CDN is accessible)
			img := fmt.Sprintf(`<div class="note-image-wrap"><img src="%s" alt="Image" width="%d" height="%d" loading="lazy" class="note-image" /></div>`, fullURL, width, height)
			images = append(images, img)
		}
	})

	if len(images) == 0 {
		return ""
	}

	if len(images) > 1 {
		layout := "image-list-odd"
		if len(images)%2 == 0 {
			layout = "image-list-even"
		}
		return "<div class=\"note-images " + layout + "\">" + strings.Join(images, "") + "</div>"
	}

	return strings.Join(images, "")
}

func (f *TelegramFetcher) extractVideo(s *goquery.Selection) string {
	var videos []string

	s.Find(".tgme_widget_message_video_wrap video").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		if src != "" {
			sel.SetAttr("src", "/static/"+src)
			sel.SetAttr("controls", "true")
			sel.SetAttr("preload", "metadata")
			sel.SetAttr("playsinline", "true")

			videoHTML, _ := sel.Html()
			if videoHTML == "" {
				// Get the outer HTML
				videoWrap := sel.ParentsFiltered(".tgme_widget_message_video_wrap").First()
				if videoWrap.Length() > 0 {
					html, _ := videoWrap.Html()
					if html != "" {
						html = strings.ReplaceAll(html, "src=\"", "src=\"/static/")
						videos = append(videos, html)
					}
				}
			}
		}
	})

	// Also check for round video
	s.Find(".tgme_widget_message_roundvideo_wrap video").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		if src != "" {
			sel.SetAttr("src", "/static/"+src)
			sel.SetAttr("controls", "true")

			videoWrap := sel.ParentsFiltered(".tgme_widget_message_roundvideo_wrap").First()
			if videoWrap.Length() > 0 {
				html, _ := videoWrap.Html()
				if html != "" {
					videos = append(videos, html)
				}
			}
		}
	})

	return strings.Join(videos, "")
}

func (f *TelegramFetcher) extractAudio(s *goquery.Selection) string {
	var audios []string

	s.Find(".tgme_widget_message_voice").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		if src != "" {
			sel.SetAttr("src", "/static/"+src)
			sel.SetAttr("controls", "true")

			audioHTML, _ := sel.Html()
			if audioHTML == "" {
				audioWrap := sel.ParentsFiltered(".tgme_widget_message_voice_wrap").First()
				if audioWrap.Length() > 0 {
					html, _ := audioWrap.Html()
					if html != "" {
						html = strings.ReplaceAll(html, "src=\"", "src=\"/static/")
						audios = append(audios, html)
					}
				}
			}
		}
	})

	return strings.Join(audios, "")
}

func (f *TelegramFetcher) extractForward(s *goquery.Selection) string {
	forward := s.Find(".tgme_widget_message_forward")
	if forward.Length() == 0 {
		return ""
	}

	fromName := forward.Find(".tgme_widget_message_forwarded_from_name").Text()
	if fromName == "" {
		fromName = forward.Find(".message_forwarded_from").Text()
	}

	if fromName == "" {
		return ""
	}

	return fmt.Sprintf(`<div class="forwarded-from">↪ %s</div>`, html.EscapeString(fromName))
}

func (f *TelegramFetcher) extractLinkPreview(s *goquery.Selection) string {
	linkPreview := s.Find(".tgme_widget_message_link_preview")
	if linkPreview.Length() == 0 {
		return ""
	}

	title := linkPreview.Find(".link_preview_title").Text()
	if title == "" {
		title = linkPreview.Find(".link_preview_site_name").Text()
	}
	description := linkPreview.Find(".link_preview_description").Text()

	linkHref := ""
	if linkPreview.Find(".link_preview_url").Length() > 0 {
		linkHref, _ = linkPreview.Find(".link_preview_url").First().Attr("href")
	}
	if linkHref == "" && linkPreview.Find("a").Length() > 0 {
		linkHref, _ = linkPreview.Find("a").First().Attr("href")
	}

	// Skip if no valid link
	if linkHref == "" || !strings.HasPrefix(linkHref, "http") {
		return ""
	}

	// Get preview image - direct URL
	imageStyle, _ := linkPreview.Find(".link_preview_image").Attr("style")
	imageURL := regexp.MustCompile(`url\(["']?(https?://[^"']+)["']?\)`).FindStringSubmatch(imageStyle)

	var imgHTML string
	if imageURL != nil {
		imgHTML = fmt.Sprintf(`<img class="link-preview-image" src="%s" alt="%s" />`, imageURL[1], html.EscapeString(title))
	}

	linkHTML := fmt.Sprintf(`<a href="%s" class="link-preview" target="_blank" rel="noopener">
		%s
		<div class="link-preview-content">
			<div class="link-preview-title">%s</div>
			<div class="link-preview-description">%s</div>
		</div>
	</a>`, linkHref, imgHTML, html.EscapeString(title), html.EscapeString(description))

	return linkHTML
}

func (f *TelegramFetcher) extractDocument(s *goquery.Selection) string {
	doc := s.Find(".tgme_widget_message_document_wrap")
	if doc.Length() == 0 {
		return ""
	}

	// Get document info
	title := doc.Find(".tgme_message_document_name").Text()
	if title == "" {
		title = "Document"
	}

	html, _ := doc.Html()
	if html != "" {
		html = strings.ReplaceAll(html, "src=\"", "src=\"/static/")
		return html
	}

	return ""
}

func (f *TelegramFetcher) extractSticker(s *goquery.Selection) string {
	var stickers []string

	// Image stickers
	s.Find(".tgme_widget_message_sticker").Each(func(i int, sel *goquery.Selection) {
		webpSrc, _ := sel.Attr("data-webp")
		pngSrc, _ := sel.Attr("data-png")

		src := webpSrc
		if src == "" {
			src = pngSrc
		}

		if src != "" {
			sticker := fmt.Sprintf(`<img class="sticker" src="%s" alt="Sticker" width="200" height="200" />`, src)
			stickers = append(stickers, sticker)
		}
	})

	// Video stickers
	s.Find(".js-videosticker_video").Each(func(i int, sel *goquery.Selection) {
		videoSrc, _ := sel.Attr("src")
		imgSrc := sel.Find("img").AttrOr("src", "")

		sticker := fmt.Sprintf(`<video class="sticker-video" src="%s" width="200" height="200" muted autoplay loop playsinline>`, videoSrc)
		if imgSrc != "" {
			sticker += fmt.Sprintf(`<img src="%s" alt="Sticker" />`, imgSrc)
		}
		sticker += "</video>"
		stickers = append(stickers, sticker)
	})

	return strings.Join(stickers, "")
}

func (f *TelegramFetcher) stripHTML(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(html, "")
}

func (f *TelegramFetcher) extractTitle(content string) string {
	if len(content) == 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			title := strings.Split(line, "。")[0]
			title = strings.Split(title, "\n")[0]
			if len(title) > 50 {
				title = title[:50] + "..."
			}
			return title
		}
	}
	return ""
}

func (f *TelegramFetcher) extractTags(s *goquery.Selection) []string {
	var tags []string
	s.Find("a[href^=\"?q=\"]").Each(func(i int, sel *goquery.Selection) {
		tag := strings.TrimPrefix(sel.Text(), "#")
		if tag != "" {
			tags = append(tags, tag)
		}
	})
	return tags
}

func (f *TelegramFetcher) parseReactions(s *goquery.Selection) []entities.Reaction {
	var reactions []entities.Reaction

	s.Find(".tgme_widget_message_reactions .tgme_reaction").Each(func(i int, sel *goquery.Selection) {
		reaction := entities.Reaction{}

		emoji := sel.Find(".emoji b").Text()
		if emoji != "" {
			reaction.Emoji = emoji
		} else {
			tgEmoji := sel.Find("tg-emoji")
			emojiID, _ := tgEmoji.Attr("emoji-id")
			if emojiID != "" {
				reaction.EmojiID = emojiID
				reaction.EmojiImage = fmt.Sprintf("https://t.me/i/emoji/%s.webp", emojiID)
			}
		}

		clone := sel.Clone()
		clone.Find(".emoji, tg-emoji, i").Remove()
		count := strings.TrimSpace(clone.Text())
		if count != "" {
			reaction.Count = count
			reactions = append(reactions, reaction)
		}
	})

	return reactions
}

func (f *TelegramFetcher) FetchNote(id string) (entities.Note, error) {
	url := fmt.Sprintf("https://%s/%s/%s?embed=1&mode=tme", f.Host, f.Channel, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return entities.Note{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return entities.Note{}, fmt.Errorf("failed to fetch note: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return entities.Note{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return entities.Note{}, fmt.Errorf("failed to parse HTML: %w", err)
	}

	msg := f.parseMessage(doc.Selection)

	return entities.Note{
		ID:        msg.ID,
		Content:   msg.Content,
		HTML:      msg.HTML,
		Title:     f.extractTitle(msg.Content),
		CreatedAt: msg.DateTime,
		Tags:      msg.Tags,
		Reactions: msg.Reactions,
	}, nil
}