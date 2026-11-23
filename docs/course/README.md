# Course sources

Which document decides what. Only the two documents in the first table are requirements. This file
interprets them; wherever it disagrees with them, they win.

## Requirement-bearing (these decide what we build)

| File | Graded as | What it fixes |
|---|---|---|
| [`project.md`](./project.md) | Đồ án (Bài tập Lớn) | The topic brief: Azure Web Apps, Azure SQL Database, Azure Functions. It also embeds [`architecture.png`](./references/scalable-ecommerce-web-app/architecture.png), which shows services the brief's text never names |
| [`rubric-project.csv`](./rubric-project.csv) | Đồ án (Bài tập Lớn) | **How the project is scored, out of 10.** Adds requirements the topic brief never mentions |

`project.md` reproduces its course source word for word, reformatted as Markdown. Anything we added
is set apart as a blockquote labelled *Ghi chú của nhóm — không thuộc đề bài*, and is not part of the
brief.

### The brief's technology and platform lists are a choice

`project.md` lists *"Triển khai trên các công nghệ: .NET, Java, Node.js, PHP và Python"* and
*"Triển khai trên các platform: Windows, Linux"*. Asked about those two lists, the lecturer ruled
that the technology list is one to pick from and that deploying on a single platform is enough. That
ruling is relayed from memory, not quoted.

The lecturer has also allowed Go, which the list does not name.

### The reference article was deleted

Microsoft deleted the article `project.md` cites. Its URL now redirects to *Basic web application*, a
starter App Service + SQL Database + Application Insights architecture that has neither the Azure
Functions nor the Redis `project.md` requires. The original was found in the
`MicrosoftDocs/architecture-center` Git history and kept at
[`references/scalable-ecommerce-web-app/`](./references/scalable-ecommerce-web-app/README.md). The
lecturer has confirmed that recovered original is the one to follow — relayed from memory, not
quoted.

**Suggestion, not a decision:** build every service that diagram shows — Azure Content Delivery
Network, multi-factor authentication, Search, Blob Storage, Queue Storage, Application Insights and
a third-party payment step — even though the brief's text names none of them. The diagram is part of
the brief, so building them is the safe reading, and they are worth building anyway.

### The rubric was not written specifically for this topic

`rubric-project.csv` grades all **20 đồ án topics**, mostly machine learning, MLOps and federated
learning. Ours is Bài tập 09, *Xây dựng trang thương mại điện tử trên nền tảng điện toán đám mây*.

One criterion is graded on a ladder rather than pass/fail: `rubric-project.csv` 1.3 is written as
`điểm-đơn vị`, so the score scales with the order of magnitude of the data processed — byte → kB →
MB → GB. Asked about it, the lecturer replied:

> Chủ đề của em liên quan đến web, nên so sánh trên database, số lượng bảng, số lượng record trong
> table (crawl data về chứ không phải clone data - vài chục row - 0.25; vài trăm 0.5; vài nghìn
> 0.75; vài chục nghìn 1.0đ)

The question also asked what the `>4GB` printed beside “Kích thước dữ liệu - dung lượng bộ nhớ cho xử lý” means — but the reply did not address it.

## Reference only

| File | Note |
|---|---|
| [`references/scalable-ecommerce-web-app/`](./references/scalable-ecommerce-web-app/README.md) | Verbatim copy of the Azure article `project.md` cites; Microsoft deleted the original |
| [`references/scalable-ecommerce-web-app/architecture.vsdx`](./references/scalable-ecommerce-web-app/architecture.vsdx) | Editable Visio source of the diagram |
