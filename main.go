package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/dabeecao/testflight-checker/internal/config"
	"github.com/dabeecao/testflight-checker/internal/db"
	"github.com/dabeecao/testflight-checker/internal/monitor"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	tele "gopkg.in/telebot.v3"
)

const (
	MsgNoFull = "🎉 Hiện tại ứng dụng <b>%s</b> trên TestFlight <b>còn slot</b>! ✅"
	MsgFull   = "❌ Hiện tại chương trình beta của <b>%s</b> trên TestFlight <b>đã đầy slot</b>. 🔒"
)

func main() {
	fmt.Println(`
    ╔══════════════════════════════════════════════════════════╗
    ║                                                          ║
    ║             TESTFLIGHT CHECKER BOT (GO VERSION)          ║
    ║                Powered by @dabeecao                      ║
    ║                                                          ║
    ╚══════════════════════════════════════════════════════════╝
    `)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ CONFIG ERROR: %v\n", err)
	}

	log.Println("🔌 Connecting to MongoDB...")
	database := db.Connect(cfg)
	database.CreateIndexes()
	log.Println("✅ Database connected and indexed.")

	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatalf("❌ BOT ERROR: %v\n", err)
	}

	log.Printf("🤖 Bot @%s is starting...\n", b.Me.Username)

	// Start Watcher
	stopChan := make(chan struct{})
	go watch(b, database, cfg.SleepTime, stopChan)

	// Handlers
	b.Handle("/start", func(c tele.Context) error {
		userID := c.Sender().ID
		ctx := context.Background()

		var result bson.M
		err := database.Users.FindOne(ctx, bson.M{"user_id": userID}).Decode(&result)
		if err == mongo.ErrNoDocuments {
			database.Users.InsertOne(ctx, bson.M{"user_id": userID, "subscriptions": []interface{}{}})
		}

		return c.Send("👋 Chào mừng bạn đến với TestFlight Checker! Bạn có thể theo dõi ứng dụng bằng cách gửi lệnh /theodoi <url>. 📚 Xem hướng dẫn bằng lệnh /help.")
	})

	b.Handle("/theodoi", func(c tele.Context) error {
		args := c.Args()
		if len(args) < 1 {
			return c.Send("⚠️ Vui lòng nhập URL TestFlight sau lệnh /theodoi.")
		}

		url := args[0]
		re := regexp.MustCompile(`^https://testflight\.apple\.com/join/([\w]+)$`)
		matches := re.FindStringSubmatch(url)
		if len(matches) < 2 {
			return c.Send("❗ Vui lòng nhập một URL TestFlight hợp lệ, ví dụ: https://testflight.apple.com/join/abc123")
		}

		tfID := matches[1]
		userID := c.Sender().ID
		ctx := context.Background()

		// Check if already subscribed
		var sub db.Subscription
		err := database.Subscriptions.FindOne(ctx, bson.M{"user_id": userID, "tf_id": tfID}).Decode(&sub)
		if err == nil {
			return c.Send("✅ Bạn đã theo dõi ứng dụng này rồi!")
		}

		info, err := monitor.GetAppInfo(tfID)
		if err != nil {
			log.Printf("Error fetching app info for %s: %v\n", tfID, err)
			if strings.Contains(err.Error(), "404") {
				return c.Send("❌ Ứng dụng này không tồn tại hoặc link TestFlight đã hết hạn. Vui lòng kiểm tra lại.")
			}
			if strings.Contains(err.Error(), "429") {
				return c.Send("⚠️ Hệ thống đang bị Apple giới hạn (Rate Limit), vui lòng thử lại sau vài phút.")
			}
			// Vẫn cho phép add nếu là lỗi mạng tạm thời khác, nhưng báo lỗi
			c.Send("⚠️ Không thể lấy thông tin ứng dụng ngay lúc này, nhưng bot sẽ tiếp tục theo dõi.")
		}

		_, err = database.Subscriptions.InsertOne(ctx, db.Subscription{
			UserID: userID,
			TFID:   tfID,
			Title:  info.Title,
		})
		if err != nil {
			return c.Send("❌ Lỗi khi lưu vào cơ sở dữ liệu.")
		}

		msg := fmt.Sprintf("✅ Đã theo dõi ứng dụng <b>%s</b> (ID: %s).\n", info.Title, tfID)
		if info.FreeSlots {
			msg += fmt.Sprintf(MsgNoFull, info.Title)
			menu := &tele.ReplyMarkup{}
			btn := menu.URL("🚀 Tải ngay", url)
			menu.Inline(menu.Row(btn))
			return c.Send(msg, menu, tele.ModeHTML)
		} else {
			msg += fmt.Sprintf(MsgFull, info.Title)
			return c.Send(msg, tele.ModeHTML)
		}
	})

	b.Handle("/botheodoi", func(c tele.Context) error {
		userID := c.Sender().ID
		ctx := context.Background()

		cursor, err := database.Subscriptions.Find(ctx, bson.M{"user_id": userID})
		if err != nil {
			return c.Send("❌ Lỗi khi truy vấn dữ liệu.")
		}
		defer cursor.Close(ctx)

		var subs []db.Subscription
		if err = cursor.All(ctx, &subs); err != nil {
			return err
		}

		if len(subs) == 0 {
			return c.Send("ℹ️ Bạn chưa theo dõi ứng dụng nào.")
		}

		menu := &tele.ReplyMarkup{}
		var rows []tele.Row
		for _, s := range subs {
			btn := menu.Data(s.Title, "unfollow", s.TFID)
			rows = append(rows, menu.Row(btn))
		}
		rows = append(rows, menu.Row(menu.Data("❌ Đóng", "close")))
		menu.Inline(rows...)

		return c.Send("❓ Chọn ứng dụng bạn muốn bỏ theo dõi hoặc nhấn 'Đóng' để thoát:", menu)
	})

	b.Handle(tele.OnCallback, func(c tele.Context) error {
		data := c.Callback().Data
		userID := c.Sender().ID
		ctx := context.Background()

		// Telebot uses \f (form feed) as a prefix for callback data from Data buttons
		if data == "\fclose" {
			c.Delete()
			return c.Respond(&tele.CallbackResponse{Text: "🔒 Đóng danh sách."})
		}

		if strings.HasPrefix(data, "\funfollow|") {
			tfID := strings.TrimPrefix(data, "\funfollow|")
			_, err := database.Subscriptions.DeleteOne(ctx, bson.M{"user_id": userID, "tf_id": tfID})
			if err != nil {
				return c.Respond(&tele.CallbackResponse{Text: "❌ Lỗi khi xóa."})
			}
			c.Edit("✅ Đã bỏ theo dõi ứng dụng thành công.")
			return c.Respond(&tele.CallbackResponse{Text: "✅ Đã bỏ theo dõi ứng dụng."})
		}

		return nil
	})

	b.Handle("/kiemtra", func(c tele.Context) error {
		userID := c.Sender().ID
		ctx := context.Background()

		cursor, err := database.Subscriptions.Find(ctx, bson.M{"user_id": userID})
		if err != nil {
			return c.Send("❌ Lỗi khi truy vấn dữ liệu.")
		}
		defer cursor.Close(ctx)

		var subs []db.Subscription
		if err = cursor.All(ctx, &subs); err != nil {
			return err
		}

		if len(subs) == 0 {
			return c.Send("ℹ️ Bạn chưa theo dõi ứng dụng nào.")
		}

		for _, s := range subs {
			info, _ := monitor.GetAppInfo(s.TFID)
			url := fmt.Sprintf(monitor.TestFlightURL, s.TFID)
			if info.FreeSlots {
				menu := &tele.ReplyMarkup{}
				btn := menu.URL("🚀 Tải ngay", url)
				menu.Inline(menu.Row(btn))
				c.Send(fmt.Sprintf(MsgNoFull, info.Title), menu, tele.ModeHTML)
			} else {
				c.Send(fmt.Sprintf(MsgFull, info.Title), tele.ModeHTML)
			}
		}
		return nil
	})

	b.Handle("/help", func(c tele.Context) error {
		helpText := "📖 <b>Hướng dẫn sử dụng TestFlight Checker Bot:</b>\n\n" +
			"• /theodoi + URL - Theo dõi ứng dụng trên TestFlight bằng cách cung cấp URL\n" +
			"• /botheodoi - Hiển thị danh sách ứng dụng đang theo dõi và chọn bỏ theo dõi\n" +
			"• /kiemtra - Kiểm tra thủ công xem có slot trống trong các ứng dụng bạn đã theo dõi không.\n" +
			"• /help - Xem hướng dẫn sử dụng bot\n\n" +
			"🙏 Hãy ủng hộ tác giả @dabeecao nếu bạn thấy bot hữu ích: https://dabeecao.org#donate ❤️"
		return c.Send(helpText, tele.ModeHTML)
	})

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		close(stopChan)
		b.Stop()
		database.Client.Disconnect(context.Background())
		os.Exit(0)
	}()

	log.Println("Bot is starting...")
	b.Start()
}

func watch(b *tele.Bot, database *db.Database, sleepTime int, stopChan chan struct{}) {
	ticker := time.NewTicker(time.Duration(sleepTime) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			ctx := context.Background()
			tfIDs, err := database.Subscriptions.Distinct(ctx, "tf_id", bson.M{})
			if err != nil {
				log.Println("Error getting distinct tf_ids:", err)
				continue
			}

			log.Printf("🔍 Monitoring %d unique TestFlight apps (staggered)...\n", len(tfIDs))
			for _, id := range tfIDs {
				tfID := id.(string)
				hitLimit := checkAndNotify(b, database, tfID)

				if hitLimit {
					log.Println("⏸️ Rate limit hit. Aborting this cycle to let the IP cool down...")
					break
				}

				// Thêm độ trễ 3 giây giữa mỗi ứng dụng để tránh lỗi 429
				time.Sleep(3 * time.Second)
			}
		}
	}
}

func checkAndNotify(b *tele.Bot, database *db.Database, tfID string) bool {
	ctx := context.Background()
	info, err := monitor.GetAppInfo(tfID)
	if err != nil {
		if strings.Contains(err.Error(), "429") {
			log.Printf("⚠️ Rate limited by Apple (429) for ID %s. Staggering needed.\n", tfID)
			return true // Signalling rate limit
		}
		
		if strings.Contains(err.Error(), "404") {
			log.Printf("🗑️ App %s has been removed (404). Cleaning up...\n", tfID)
			
			// Lấy danh sách người dùng đang theo dõi để thông báo
			cursor, err := database.Subscriptions.Find(ctx, bson.M{"tf_id": tfID})
			if err == nil {
				defer cursor.Close(ctx)
				var subs []db.Subscription
				cursor.All(ctx, &subs)
				
				appTitle := tfID
				if len(subs) > 0 {
					appTitle = subs[0].Title
				}

				msg := fmt.Sprintf("⚠️ Ứng dụng <b>%s</b> đã bị gỡ bỏ khỏi TestFlight hoặc link không còn tồn tại. Bot sẽ tự động ngừng theo dõi ứng dụng này.", appTitle)
				
				for _, sub := range subs {
					b.Send(tele.ChatID(sub.UserID), msg, tele.ModeHTML)
				}
			}

			// Xóa khỏi database
			database.Subscriptions.DeleteMany(ctx, bson.M{"tf_id": tfID})
			database.AppStatus.DeleteOne(ctx, bson.M{"tf_id": tfID})
			
			return false
		}

		log.Printf("⚠️ Lỗi khi kiểm tra tf_id %s: %v\n", tfID, err)
		return false
	}

	var status db.AppStatus
	err = database.AppStatus.FindOne(ctx, bson.M{"tf_id": tfID}).Decode(&status)

	statusChanged := false
	if err == mongo.ErrNoDocuments {
		statusChanged = true
	} else if status.LastFreeSlots != info.FreeSlots {
		statusChanged = true
	}

	if statusChanged {
		// Send notifications
		cursor, err := database.Subscriptions.Find(ctx, bson.M{"tf_id": tfID})
		if err == nil {
			defer cursor.Close(ctx)
			var subs []db.Subscription
			cursor.All(ctx, &subs)
			for _, sub := range subs {
				// Cập nhật tên nếu trước đó chưa rõ
				if (sub.Title == "Ứng dụng chưa rõ" || sub.Title == "") && info.Title != "Ứng dụng chưa rõ" {
					database.Subscriptions.UpdateMany(ctx, bson.M{"tf_id": tfID}, bson.M{"$set": bson.M{"title": info.Title}})
				}

				msg := ""
				if info.FreeSlots {
					msg = fmt.Sprintf(MsgNoFull, info.Title)
					menu := &tele.ReplyMarkup{}
					btn := menu.URL("🚀 Tải ngay", fmt.Sprintf(monitor.TestFlightURL, tfID))
					menu.Inline(menu.Row(btn))
					b.Send(tele.ChatID(sub.UserID), msg, menu, tele.ModeHTML)
				} else {
					msg = fmt.Sprintf(MsgFull, info.Title)
					b.Send(tele.ChatID(sub.UserID), msg, tele.ModeHTML)
				}
				// Small delay to avoid Telegram rate limits if many users are subscribed
				time.Sleep(50 * time.Millisecond)
			}
		}

		// Update status
		database.AppStatus.UpdateOne(
			ctx,
			bson.M{"tf_id": tfID},
			bson.M{"$set": bson.M{"last_free_slots": info.FreeSlots}},
			options.Update().SetUpsert(true),
		)
	}

	return false
}
