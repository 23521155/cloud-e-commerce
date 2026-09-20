# Architect scalable e-commerce web app

> Đây là nguyên văn bài viết mà URL tham khảo trong
> [`project.md`](../project.md) trỏ tới. Microsoft đã xoá bài này khỏi Azure Architecture Center,
> nên lưu lại ở đây để trích dẫn trong báo cáo.
>
> | | |
> |---|---|
> | Tiêu đề gốc | Architect scalable e-commerce web app |
> | Bị xoá tại commit | [`c41cf98`](https://github.com/MicrosoftDocs/architecture-center/commit/c41cf9855a6b9d6db6bd4e1c8fd145c8e45629ae) — `"delete and redirect articles"` |
> | Nguồn nội dung dưới đây | `docs/web-apps/idea/scalable-ecommerce-web-app-content.md` tại commit [`2e54c95`](https://github.com/MicrosoftDocs/architecture-center/tree/2e54c953b88313030cbc5112a795f9244069afa1) (bản cuối trước khi xoá) |
> | Redirect hiện tại | → [Basic web application](https://learn.microsoft.com/en-us/azure/architecture/web-apps/app-service/architectures/basic-web-app) với `redirect_document_id: false`, tức **không phải bài thay thế tương đương** |
> | Sơ đồ nguồn | [`scalable-ecommerce-web-app.vsdx`](./scalable-ecommerce-web-app.vsdx) — bản Visio tải từ link "Download a Visio file" bên dưới, khớp với ảnh PNG. Dùng làm điểm xuất phát khi cần chỉnh sửa sơ đồ kiến trúc dự án. |

Build scalable e-commerce web apps with Azure Functions and Web Apps, so you can create personalized experiences while Azure takes care of the infrastructure. This solution is ideal for the retail industry.

## Potential use cases

This solution is ideal for retail organizations, but it can apply to other industries that sell goods and services.

## Architecture

![Architecture diagram shows the transaction from users finding products to purchasing them through web apps to third party payment.](./scalable-ecommerce-web-app.png)

*Download a [Visio file](https://arch-center.azureedge.net/scalable-ecommerce-web-app.vsdx) of this architecture.*

### Dataflow

1. User accesses the web app in browser and signs in.
2. Browser pulls static resources such as images from Azure Content Delivery Network.
3. User searches for products and queries SQL database.
4. Web site pulls product catalog from database.
5. Web app pulls product images from Blob Storage.
6. Page output is cached in Azure Cache for Redis for better performance.
7. User submits order and order is placed in the queue.
8. Azure Functions processes order payment.
9. Azure Functions makes payment to third party and records payment in SQL database.

### Components

- [Web Apps](https://learn.microsoft.com/en-us/azure/well-architected/service-guides/app-service-web-apps): An App Service Web App runs in a single region, accessible to web and mobile browsers.
- [Azure SQL Database](https://learn.microsoft.com/en-us/azure/well-architected/service-guides/azure-sql-database): Managed, intelligent SQL in the cloud.
- [Azure Functions](https://learn.microsoft.com/en-us/azure/well-architected/service-guides/azure-functions): Process events with serverless code.
- [Application Insights](https://learn.microsoft.com/en-us/azure/well-architected/service-guides/application-insights): Detect, triage, and diagnose issues in your web apps and services.

## Contributors

*This article is maintained by Microsoft. It was originally written by the following contributors.*

Principal author:

- [Andrew Harvey](https://www.linkedin.com/in/andrewharvey) | Principal Program Manager - Microsoft for Startups

## Next steps

- [Get started easily with Web Apps using the five-minute quick starts](https://learn.microsoft.com/en-us/azure/app-service)
- [Build an ASP.NET app in Azure with SQL Database](https://learn.microsoft.com/en-us/azure/app-service/app-service-web-tutorial-dotnet-sqldatabase)
- [Learn what can you do with Azure Functions](https://learn.microsoft.com/en-us/azure/azure-functions/functions-overview)
- [Application Performance Management with Application Insights](https://learn.microsoft.com/en-us/azure/azure-monitor/app/app-insights-overview)
