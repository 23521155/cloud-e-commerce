# Bài tập 09: Xây dựng trang thương mại điện tử trên nền tảng điện toán đám mây

![Sơ đồ kiến trúc hệ thống](./references/scalable-ecommerce-web-app.png)

## Điều kiện

Có kỹ năng xây dựng trang web sử dụng ngôn ngữ Java & PHP & Script để cấu hình.

## Mô tả

Tạo một trang web bán hàng cho phép xem sản phẩm, tìm kiếm sản phẩm và thanh toán sản phẩm.

## Yêu cầu

Sử dụng các chức năng Azure.

### Web Apps

Dịch vụ chạy trang web trên cloud, hỗ trợ các công cụ build, triển khai và quản lý.

Các dịch vụ cần thực hiện:

- Sử dụng chức năng CD (Continuous Deployment) với Git, GitHub, Bitbucket.
- Triển khai trên các công nghệ: .NET, Java, Node.js, PHP và Python.
- Triển khai trên các platform: Windows, Linux.
- Tự động scale trong môi trường cloud: horizontal (ngang — số lượng VM) & vertical (dọc — sức mạnh của VM).
- Cấu hình cân bằng tải.

### Azure SQL Database

- Implement Azure Synapse Link for SQL.
- Sử dụng Redis để nâng cao hiệu suất.

### Azure Functions

- Phát triển trang web theo event-driven, serverless.
- Tích hợp các dịch vụ sử dụng trigger & binding.

## Tài liệu tham khảo

<https://learn.microsoft.com/en-us/azure/architecture/solution-ideas/articles/scalable-ecommerce-web-app>