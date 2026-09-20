# Course sources

Which document decides what. Read this before treating anything in `docs/course/` as a requirement.

## Requirement-bearing (these decide what we build)

| File | Deliverable | What it fixes |
|---|---|---|
| [`project.md`](./project.md) | Đồ án (Bài tập Lớn) | The topic brief: Azure Web Apps, Azure SQL Database + Synapse Link + Redis, Azure Functions |
| [`rubric-project.csv`](./rubric-project.csv) | Đồ án (Bài tập Lớn) | **How the đồ án is scored, out of 10.** Adds requirements the topic brief never mentions |
| [`lab-07-deploying-monitoring-app-service.md`](./lab-07-deploying-monitoring-app-service.md) | Bài tập quá trình | Hands-on lab: App Service plan, web app, deployment slots, monitoring, Application Insights, alerts |
| [`rubric-coursework.csv`](./rubric-coursework.csv) | Bài tập quá trình | How the coursework is scored |

The full coursework topic title, from the course schedule page, is
**"Deploying And Monitoring Azure App Service Web Apps — So sánh công cụ của Azure với Third Party"**.
The lab document covers the Azure half; the comparison against third-party tooling comes from the
topic title and is not described in the lab document.

`project.md` and `rubric-project.csv` are **both** authoritative for the đồ án and they do not
overlap much. The brief names Azure services; the rubric scores deliverables. Satisfy the union.

## Reference only (context, never a requirement)

| File | Note |
|---|---|
| [`references/scalable-ecommerce-web-app.md`](./references/scalable-ecommerce-web-app.md) | Verbatim copy of the Azure article `project.md` cites; Microsoft deleted the original |
| [`references/scalable-ecommerce-web-app.png`](./references/scalable-ecommerce-web-app.png) | The architecture diagram. **Embedded in `project.md`, so it is part of the brief** |
| [`references/scalable-ecommerce-web-app.vsdx`](./references/scalable-ecommerce-web-app.vsdx) | Editable Visio source of that diagram |

## Not in this repository

Course teaching material, left where it is because of size and because none of it states a
requirement. Recorded here so nobody has to guess whether it was considered.

| What | Where | Why it is not a requirement |
|---|---|---|
| Course slides, `Lecture 1`–`Lecture 5`, `Buoi01`–`Buoi02` | `D:\uit-hk7\IS402-cloud\Slide Sinh Viên-...\Slide Sinh Viên` | Lecture material on distributed computing, IaaS, PaaS, SaaS |
| Textbooks: Marinescu *Cloud Computing: Theory and Practice*; Buyya et al. *Cloud Computing Principles and Paradigms* | same folder | Background reading |
| 11 data-engineering books: Synapse Analytics, Data Factory, Databricks, Data Lake, Azure Storage & Batch Analytics | `D:\Downloads\Tài Liệu hỗ trợ bài tập quá trình-...\Azure` | Support material for the coursework. Their subject matter does hint at what the rubric's data criteria expect |
| Course schedule page | `D:\Downloads\Cloud Computing - Schedule (...).html` | Index of seminar topics |

## The failure mode this file exists to prevent

The reference article contains Azure CDN, Blob Storage, a Search component and multi-factor
authentication. **None of those appear in the text of `project.md`.** They are in scope only
because `project.md` embeds that article's diagram. Slides and textbooks name dozens more Azure
services; every one of them will look like a requirement to someone skimming.

Decide which document states the deliverable first. Then read the rest.
