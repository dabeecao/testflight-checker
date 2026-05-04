# TestFlight Checker Bot (Golang Version) 🚀

Bot Telegram giúp theo dõi trạng thái các ứng dụng trên Apple TestFlight và thông báo ngay lập tức khi có slot trống. Phiên bản này được viết bằng **Golang** để đạt hiệu năng tối ưu, tiết kiệm tài nguyên và hoạt động ổn định 24/7.

## ✨ Tính năng nổi bật

- **Theo dõi thời gian thực**: Tự động kiểm tra các link TestFlight theo chu kỳ cấu hình.
- **Thông báo tức thì**: Gửi tin nhắn Telegram kèm nút "Tải ngay" khi phát hiện có slot.
- **Tối ưu hóa Rate Limit**: Cơ chế quét so le (staggered) và tự động dừng khi bị Apple giới hạn (Error 429).
- **Tự động dọn dẹp**: Phát hiện và xóa bỏ các link TestFlight đã chết hoặc bị gỡ (Error 404).
- **Quản lý thông minh**: Người dùng có thể tự thêm/xóa ứng dụng theo dõi thông qua giao diện nút bấm tiện lợi.
- **Database ổn định**: Sử dụng MongoDB với cơ chế đánh chỉ mục (indexing) cho tốc độ truy vấn cực nhanh.

## 🛠 Yêu cầu hệ thống

- **Go**: Phiên bản 1.20 trở lên.
- **MongoDB**: Đã cài đặt và đang chạy (local hoặc Atlas).
- **Bot Token**: Lấy từ [@BotFather](https://t.me/BotFather).

## 🚀 Cài đặt và Sử dụng

1. **Clone dự án**:
   ```bash
   git clone https://github.com/dabeecao/testflight-checker.git
   cd testflight-checker
   ```

2. **Cấu hình môi trường**:
   Sao chép file mẫu và điền thông tin của bạn:
   ```bash
   cp .env.example .env
   ```
   Chỉnh sửa file `.env` với Token và URI MongoDB của bạn.

3. **Cài đặt thư viện**:
   ```bash
   go mod tidy
   ```

4. **Biên dịch và Chạy**:
   Sử dụng script build có sẵn:
   ```bash
   chmod +x build.sh
   ./build.sh
   ./testflight-checker
   ```

## 📖 Danh sách lệnh Bot

- `/start`: Khởi động bot và xem lời chào.
- `/theodoi [URL]`: Thêm một ứng dụng TestFlight vào danh sách theo dõi.
- `/botheodoi`: Xem danh sách ứng dụng đang theo dõi và tùy chọn bỏ theo dõi.
- `/kiemtra`: Kiểm tra thủ công trạng thái của tất cả ứng dụng đang theo dõi.
- `/help`: Xem hướng dẫn sử dụng chi tiết.

## ⚙️ Cơ chế hoạt động (Technical Notes)

- **Watcher**: Chạy trong một goroutine riêng biệt, sử dụng `time.Ticker` để quản lý chu kỳ.
- **Scraper**: Sử dụng `htmlquery` (XPath) để bóc tách dữ liệu từ Apple. Có User-Agent iPhone hiện đại để tránh bị block.
- **Graceful Shutdown**: Bot lắng nghe tín hiệu `SIGINT`, `SIGTERM` để đóng kết nối DB và dừng an toàn.

## 🤝 Hỗ trợ

Nếu bạn thấy bot hữu ích, hãy ủng hộ tác giả tại: [dabeecao.org#donate](https://dabeecao.org#donate) ❤️

---
*Phát triển bởi @dabeecao*
