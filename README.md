# PalmYield

A cooperative management platform for smallholder palm-oil (sawit) farmers — a web dashboard for cooperative admins and an **offline-first mobile app** for farmers who often work with poor or no internet connectivity.

Farmers record harvest data in the field without a signal; the app syncs reliably to the backend once connectivity returns, and the backend guarantees no duplicate records even if a sync is retried or interrupted mid-upload. This offline-sync design is the core engineering focus of the project.

## Status

🚧 Early development — see [docs/architecture](docs/architecture) for design notes and the roadmap.

## Structure

```
backend/   Go API (chi, sqlc, PostgreSQL, Asynq background jobs)
web/       Next.js admin dashboard (TypeScript)
mobile/    Flutter farmer app (offline-first, Drift local DB)
docs/      Architecture notes, ADRs, API contract, case study
```

## Local development

Prerequisites: Go 1.24+, Node 22+, Flutter 3.41+, Docker.

```bash
docker compose up -d       # postgres + redis
cd backend && make migrate-up && make run
cd web && npm install && npm run dev
cd mobile && flutter pub get && flutter run
```

## License

Private project — not licensed for reuse yet.
