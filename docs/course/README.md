# Course sources

Which document decides what. Only the four documents in the first table are requirements. This file
interprets them; wherever it disagrees with them, they win.

## Requirement-bearing (these decide what we build)

| File | Graded as | What it fixes |
|---|---|---|
| [`project.md`](./project.md) | Đồ án (Bài tập Lớn) | The topic brief: Azure Web Apps, Azure SQL Database, Azure Functions. It also embeds [`architecture.png`](./references/scalable-ecommerce-web-app/architecture.png), which shows services the brief's text never names |
| [`rubric-project.csv`](./rubric-project.csv) | Đồ án (Bài tập Lớn) | **How the project is scored, out of 10.** Adds requirements the topic brief never mentions |
| [`coursework.md`](./coursework.md) | Bài tập quá trình | The Azure lab and the third-party tooling comparison |
| [`rubric-coursework.csv`](./rubric-coursework.csv) | Bài tập quá trình | How the coursework is scored |

`project.md` and `coursework.md` reproduce their course sources word for word, reformatted as
Markdown. Anything we added is set apart as a blockquote labelled *Ghi chú của nhóm — không thuộc đề
bài*, and is not part of the brief.

**The plan is for one web product to answer all four documents.** The two topics were chosen because
they overlap: the coursework would be this application deployed and monitored, rather than a separate
one. Whether that holds up is still open.

### Neither rubric was written specifically for this topic

Each grades every topic in its own pool:

- `rubric-project.csv` — all **20 đồ án topics**, mostly machine learning, MLOps and federated
  learning. Ours is Bài tập 09, *Xây dựng trang thương mại điện tử trên nền tảng điện toán đám mây*.
- `rubric-coursework.csv` — all **37 seminar topics** on the
  [course schedule](https://sites.google.com/uit.edu.vn/cloudcomputing/schedule), mostly Azure and
  OpenStack hands-on labs plus a federated-learning track. Ours is the Week 7 Azure one,
  *Deploying And Monitoring Azure App Service Web Apps — So sánh công cụ của Azure với Third Party*.

The one thing to carry across: Sections 1–3 hold the same criteria in nearly the same words, but only
`rubric-project.csv` annotates them for Web. Read the matching `rubric-coursework.csv` criterion the
same way — ETL is *"data nhận từ client — detect lưu trữ vào trong storage cloud"*, the
processing-flow criterion is *"Web — luồng đi dữ liệu, logic của trang web"*.

One criterion is graded on a ladder rather than pass/fail: `rubric-coursework.csv` 1.2 and
`rubric-project.csv` 1.3 are written as `điểm-đơn vị`, so the score scales with the order of magnitude
of the data processed — byte → kB → MB → GB, with GB the top rung.

### Open questions for the lecturer

- **The technology and platform lists.** `project.md` lists *"Triển khai trên các công nghệ: .NET,
  Java, Node.js, PHP và Python"* and *"Triển khai trên các platform: Windows, Linux"*, and the Điều
  kiện section mentions *"Java & PHP & Script"*. Unresolved whether these are options to pick from or
  lists to satisfy in full.
- **ASP.NET Core in the lab.** `coursework.md`'s lab description deploys *"an ASP.NET Core app"*.
  Unresolved whether that stack is required or only the lab's example app.
- **`>4GB`**, in `rubric-coursework.csv` 1.2 and `rubric-project.csv` 1.3. The criterion is titled
  *"Kích thước dữ liệu - dung lượng bộ nhớ cho xử lý"* — two things — so it is most likely a memory
  spec rather than a rung of the ladder above.
- **The reference article is gone.** Microsoft deleted the article `project.md` cites; its URL now
  redirects to *Basic web application*, a starter App Service + SQL Database + Application Insights
  architecture with neither the Azure Functions nor the Redis that `project.md` requires. The
  original was found in the `MicrosoftDocs/architecture-center` Git history
  ([`references/scalable-ecommerce-web-app/`](./references/scalable-ecommerce-web-app/README.md)).
  Unresolved whether to follow the original or the current page — and if the original, whether the
  services its diagram shows but the brief's text never names (Azure Content Delivery Network,
  multi-factor authentication, Search, Blob Storage, Queue Storage, Application Insights, a
  third-party payment step) are required or only illustrative.

## Reference only

| File | Note |
|---|---|
| [`references/scalable-ecommerce-web-app/`](./references/scalable-ecommerce-web-app/README.md) | Verbatim copy of the Azure article `project.md` cites; Microsoft deleted the original |
| [`references/scalable-ecommerce-web-app/architecture.vsdx`](./references/scalable-ecommerce-web-app/architecture.vsdx) | Editable Visio source of the diagram |

## Usefulness of the external course material (unverified AI assessment)

> **Do not trust this section.** An AI assistant produced every score and note below by reading the
> material; nothing here has been checked by a human.

The course material sits outside the repository because of its size, in
[Slide Sinh Viên](https://drive.google.com/drive/folders/1PkuspkNsv4eObU9f2ASS12fl3yZ9UOtL?usp=drive_link)
and [Tài Liệu hỗ trợ bài tập quá trình/Azure](https://drive.google.com/drive/folders/1jh6wRoTOVG-xwFXNwW6UFWmmRHli_Btc?usp=drive_link).

**Everything below is for reference and lookup only, not a requirement.**

Each document is scored 0–5 against the coursework brief and 0–5 against the project brief, so the
total is out of 10; rows are sorted by it.

| Document | Coursework | Project | Total | What it contains |
|---|---:|---:|---:|---|
| [`Buoi01_GioiThieu_02.pdf`](https://drive.google.com/file/d/1vuI-cYJGQ_BEqD_t8z_lal8qULqIR0uN/view) | 5 | 5 | **10** | Course intent and an earlier version of the rubric; App Service, Azure Monitor, load balancing, Functions, triggers and bindings. |
| [`Dan C. Marinescu - Cloud Computing_ Theory and Practice-Morgan Kaufmann (2022).pdf`](https://drive.google.com/file/d/1QU0xCWD5JjbvfOsGk-5yW0unvCrttcZO/view) | 5 | 4 | **9** | Theory for service models, storage, databases, scaling, latency and cloud applications, plus Elastic Beanstalk and CloudWatch. Its Windows Azure material describes a retired product. |
| [`Buoi02_CloudComputingPlatform_01.pdf`](https://drive.google.com/file/d/1oHqXvsAOxpv1VuwR-OhHvlOpbcU-EJDi/view) | 4 | 4 | **8** | Retail data/ETL architecture, a CI/CD path through staging and production, and a map of Azure App Service, Functions, load balancing and SQL services. |
| [`Lecture 4 - Introduction to PaaS.pdf`](https://drive.google.com/file/d/16zgmljXCZJwe3hNkU_LdW-d6HX6eLIjb/view) | 4 | 4 | **8** | PaaS, dynamic provisioning, load balancing, monitoring, billing, quota, storage and SQL. Its Web Role, Worker Role and SQL Azure material describes an earlier Azure architecture. |
| [`Lecture 4.3 - PaaS Techniques (Database).pdf`](https://drive.google.com/file/d/1Ezu3o8OQEKWrZB4ckgEn9IKOdBwPO6rX/view) | 4 | 4 | **8** | Structured and semi-structured data, SQL/NoSQL, distributed databases, replication, locality, caching, throughput and latency. |
| [`Rajkumar Buyya, James Broberg, Andrzej M. Goscinski - Cloud Computing Principles and Paradigms (Wiley Series on Parallel and Distributed Computing)-Wiley (2011).pdf`](https://drive.google.com/file/d/1RZRxZyPd-5ebFHYgniwe-iazIgCcgNE1/view) | 4 | 4 | **8** | Monitoring and AWS application architecture: CloudWatch, Auto Scaling, Elastic Load Balancing, queues and layered web applications. Published 2011. |
| [`Danh Sách Chủ Đề Thuyết Trình`](https://docs.google.com/spreadsheets/d/1KuM-te-RQk9-LPm7rJhG89eoHdcL0qtVePQPwaZ0ihU/edit) | 4 | 4 | **8** | Azure, GCP and AWS presentation topic outlines from an earlier year, including AWS Elastic Beanstalk. |
| [`Nagaraj Venkatesan, Ahmad Osama - Azure Data Engineering Cookbook_ Get well versed in various data engineering techniques in Azure using this recipe-based guide, 2nd Edition-Packt Publishing (2022).pdf`](https://drive.google.com/file/d/13UmyIYr-3xX4LksWZh2BT251FuxLPEtZ/view) | 4 | 4 | **8** | Azure storage, Azure SQL Database, ADF/ETL, Synapse, pipeline monitoring, read/write performance and optimization. |
| [`Lecture 2 - Introduction to Cloud Computing.pdf`](https://drive.google.com/file/d/1HklsnLfAafrLuEENQZP7bEOQGJQnDLHn/view) | 4 | 3 | **7** | Monitoring metrics, performance, load balancing and the distinction between IaaS, PaaS and SaaS. |
| [`Richard Swinbank - Azure Data Factory by Example_ Practical Implementation for Data Engineers-Apress (2021).pdf`](https://drive.google.com/file/d/149UDz4axJeX1kE1MxhvvPdjha1zjwIfO/view) | 4 | 3 | **7** | Copy Data, parameters, control flow, data flows, publishing, triggers and ADF monitoring. Its monitoring chapters are about ADF. |
| [`Lecture 3.2 - Storage Virtualization.pdf`](https://drive.google.com/file/d/1o5m32PZ3M1VgqgYOkMHVbd5mbmpRj6ao/view) | 3 | 3 | **6** | Storage theory: DAS/NAS/SAN, block and file storage, logical volumes, replication, availability and throughput. |
| [`Lecture 4.1 - PaaS Techniques (File System).pdf`](https://drive.google.com/file/d/1MLeuFQsmH7G6oQ3ONGov62e6m1EDMO5S/view) | 3 | 3 | **6** | Large datasets, distributed storage, replication, concurrent reads/writes, bandwidth and fault tolerance, taught through HDFS. |
| [`Danh sách các Assignment 2022`](https://docs.google.com/spreadsheets/d/14rVcrRa2VGotpYsfjDZg5diMgI-uOZ6IP6YciOs6jgE/edit) | 3 | 3 | **6** | Azure Synapse and the stages of a data pipeline: ingestion, processing, storage, workflow, monitoring/governance and consumption. |
| [`Ron C. L'Esteve - The Definitive Guide to Azure Data Engineering_ Modern ELT, DevOps, and Analytics on the Azure Cloud Platform.pdf`](https://drive.google.com/file/d/1sEMBGzMldEMqmXbD3W7wRt_GuqyNrD0x/view) | 3 | 3 | **6** | Azure SQL, ADF/ELT, Synapse SQL Pool, large-data processing, logging and CI/CD. Its Synapse Link material uses Cosmos DB. |
| [`Lecture 3 - Introduction to IaaS.pdf`](https://drive.google.com/file/d/1bY1YvTC0nVtTSmPJfvrYLzN8qAs3eQoW/view) | 2 | 3 | **5** | Infrastructure classification, resource provisioning, read/write and bandwidth monitoring, scaling and load balancing. |
| [`Lecture 4.2 - PaaS Techniques (Programming Model).pdf`](https://drive.google.com/file/d/1-ZM3AlrhtYJsfzIqQhfeHz-ZraEFEFlC/view) | 3 | 2 | **5** | Large-data processing, MapReduce, data locality, scheduling and worker monitoring. |
| [`Bhadresh Shiyal - Beginning Azure Synapse Analytics_ Transition from Data Warehouse to Data Lakehouse-Apress (2021).pdf`](https://drive.google.com/file/d/11lhzZWqSSVKP7ErGCAgu-gVo6fdcsxD7/view) | 3 | 2 | **5** | Synapse SQL, Spark, pipelines, ETL and architecture. Its Synapse Link chapter is for Cosmos DB. |
| [`Lecture 5 - Introduction to SaaS _ Its Techniques.pdf`](https://drive.google.com/file/d/1ZTxmUZ5jvkzvaNq-KD9tqz29Bqnrfqit/view) | 2 | 2 | **4** | SaaS/PaaS distinction, multitenancy, SLA/QoS, cost and the effect of data-centre distance on latency. |
| [`ChuDeBaoCao_2022.pdf`](https://drive.google.com/file/d/1kjii_Ztn0xKVLITXICnV3r3TZGfHDXUa/view) | 2 | 2 | **4** | Report, video, slides and demo expectations from an earlier year, plus data-processing, monitoring, microservices and Docker topics. |
| [`Christian Coté_ Michelle Gutzait_ Giuseppe Ciaburro - Hands-On Data Warehousing with Azure Data Factory-Packt Publishing (2018).epub`](https://drive.google.com/file/d/1e807W6RVU4H0hBVVZyNC8iruPnFYrIkl/view) | 2 | 2 | **4** | ADF, Blob Storage, Azure SQL, monitoring and pipelines. Published 2018; overlaps the newer ADF titles listed here. |
| [`Richard L. Nuckolls - Azure Storage, Streaming, and Batch Analytics A guide for data engineers-Manning Publications (2020).pdf`](https://drive.google.com/file/d/1teOQE5RK9WYgcAzGLJ_gr_cqY9YgVYVO/view) | 2 | 2 | **4** | Storage, regions, throughput, Azure SQL and ADF. Much of it centres on ADLS Gen1, U-SQL and Data Lake Analytics, all retired. |
| [`Phani Raj, Vinod Jaiswal - Azure Databricks Cookbook_ Accelerate and scale real-time analytics solutions using the Apache Spark-based analytics service-Packt Publishing (2021).pdf`](https://drive.google.com/file/d/12TnjDAXlhA_Ojl-Ci1Zx8mysMHNDRbEN/view) | 2 | 2 | **4** | Large-data reads/writes, Delta Lake, streaming, CI/CD and Log Analytics, on Databricks. |
| [`Lecture 1 - Overview of Distributed Computing.pdf`](https://drive.google.com/file/d/1unZHAh9BnYw4Tz9KtJ5_9Mprlu50pEk_/view) | 1 | 2 | **3** | Introductory utility/grid/cloud concepts, pay-as-you-go and service models, with dated survey figures. |
| [`Lecture 3.1 - Server Virtualization.pdf`](https://drive.google.com/file/d/13ATmcg1PLzobhLuXdR-JuqvKaIjRe2Ib/view) | 1 | 2 | **3** | Hypervisors, live migration, VM scale-up and workload balancing. |
| [`Lecture 3.3 - Network virtualization.pdf`](https://drive.google.com/file/d/1l4o78ztwOu4YuF5fNgsLOnY37ovWKQA0/view) | 1 | 2 | **3** | Network virtualization, bridge, DHCP, NAT and Eucalyptus. |
| [`ChuDeBaoCao_2021.pdf`](https://drive.google.com/file/d/1gXOd1zXVjSvpNtwqMFJjNArjjvccQlda/view) | 1 | 2 | **3** | Report structure and topic list from an earlier year, including commercial clouds and Azure PaaS. |
| [`01_E-Commerce_Merunas Grincalaitis - Mastering Ethereum_ Implement advanced blockchain applications using Ethereum-supported tools, services, and protocols-Packt Publishing (2019).pdf`](https://drive.google.com/file/d/1niX6FxD8Y74yEv78X6LMNKUijLsKBJ2_/view) | 1 | 2 | **3** | An e-commerce marketplace with products and payments, built on Ethereum smart contracts. |
| [`Harsh Chawla, Pankaj Khattar - Data Lake Analytics on Microsoft Azure_ A Practitioner's Guide to Big Data Engineering-Apress (2020).pdf`](https://drive.google.com/file/d/1P5Nw9XPunIFShFAaBSfDZlpS1BC427J1/view) | 2 | 1 | **3** | Data lake, Kafka, Event Hubs, ingestion and ML serving. |
| [`01_Arjuna Sky Kok - Hands-On Blockchain for Python Developers_ Gain blockchain programming skills to build decentralized applications using Python-Packt Publishing (2019).pdf`](https://drive.google.com/file/d/1OdH5EqUz1I2w0FxiScU6NbeyqkBZSQMS/view) | 1 | 1 | **2** | Smart contracts, Web3, Ethereum and IPFS. Deployment, Redis and Azure appear only in passing. |
| [`Lad, Sagar_Kumar, Anjani__ Anjani Kumar - Mastering Databricks Lakehouse Platform-BPB Publications (2022).epub`](https://drive.google.com/file/d/1M9q1Fo4UcCEUUy-nzUp_Z3cuVlm5Dofw/view) | 1 | 1 | **2** | Databricks, Delta Live Tables, MLflow, orchestration and BI. |
| [`Julian Soh, Priyanshi Singh - Data Science Solutions on Azure_ Tools and Techniques Using Databricks and MLOps-Apress (2020).pdf`](https://drive.google.com/file/d/1xXLybsLVO8PyjH0z5_SDLrr3FCiTZ86B/view) | 1 | 1 | **2** | Statistics, Azure Machine Learning, Databricks and MLOps. |
| [`Kiyoshi Nakayama PhD, George Jeno - Federated Learning with Python_ Design and implement a federated learning system and develop applications using existing frameworks-Packt Publishing (2022)(1).epub`](https://drive.google.com/file/d/1KUWzA9usf4GMRaWgEtpleI-Lqo2e4yK-/view) | 0 | 0 | **0** | Federated-learning architecture, client/server training, FedAvg and distributed-ML frameworks. |
