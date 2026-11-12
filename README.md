# GameHub Backend

Backend API สำหรับเว็บไซต์ GameHub พัฒนาด้วยภาษา Go และใช้ PostgreSQL บน Supabase เป็นฐานข้อมูล

## Technologies

- Go
- PostgreSQL
- Supabase
- JWT Authentication
- REST API

## Features

- User Registration and Login
- JWT Authentication
- Game Management
- Game Search
- User Profile
- User Game Library
- Game Reviews
- Game Ratings
- Admin Management

## Project Structure

```text
gamehub-backend/
├── database/
├── handlers/
├── middleware/
├── models/
├── routes/
├── services/
├── cmd/
├── main.go
├── schema.sql
├── seed.sql
├── docker-compose.yml
├── go.mod
└── README.md
```

## Requirements

- Go
- PostgreSQL

## Installation

Clone the repository:

```bash
git clone https://github.com/Jinwara670710314/gamehub-backend.git
```

เข้าไปยังโฟลเดอร์โปรเจกต์:

```bash
cd gamehub-backend
```

ติดตั้ง dependencies:

```bash
go mod download
```

## Environment Variables

Create a `.env` file in the project root directory.

```env
DATABASE_URL=your_supabase_database_url
PORT=8080
JWT_SECRET=your_jwt_secret
TWITCH_CLIENT_ID=your_twitch_client_id
TWITCH_CLIENT_SECRET=your_twitch_client_secret
```

Do not upload the `.env` file or expose database credentials and secret keys to GitHub.

## Run the Backend

```bash
go run main.go
```

## Team

- Jinwara Daungchang
- Phanukon

## Deployment

- Frontend: Vercel
- Backend: Render
- Database: Supabase