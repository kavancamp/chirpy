# Chirpy 🐦

Chirpy is a mini Twitter-style microblogging API written in Go. It allows users to register, log in, post short messages called "chirps", and manage authentication with access and refresh tokens. It also includes premium membership functionality through webhook integration with Polka.

## Features

- ✅ User registration and login
- ✅ JWT-based access token auth
- ✅ Refresh token lifecycle (issue, validate, revoke)
- ✅ Create, retrieve, and delete chirps
- ✅ Filter chirps by author and sort by date
- ✅ Chirpy Red membership via Polka webhook
- ✅ Admin-only endpoints with platform-based restrictions

---

## 📁 API Endpoints

### Authentication

#### `POST /api/users`
Create a new user.
```json
{
  "email": "example@example.com",
  "password": "securepassword"
}
```
PUT /api/users
Update the authenticated user's email and/or password.

Headers:
<pre>Authorization: Bearer access_token</pre>
Body:
```json
{
  "email": "new@example.com",
  "password": "newpassword"
}
```
POST /api/login
Authenticate and receive access & refresh tokens.

```json
{
  "email": "example@example.com",
  "password": "securepassword"
}
```
POST /api/refresh
Get a new access token using a valid refresh token.
<pre>Authorization: Bearer refresh_token</pre>


POST /api/revoke

Revoke the current refresh token.
<pre>Authorization: Bearer refresh_token</pre>

Chirps
POST /api/chirps
Create a new chirp (max 140 characters).

<pre>Authorization: Bearer access_token</pre>
Body:
```json
{
  "body": "Hello, Chirpy!"
}
```
GET /api/chirps
Get all chirps. Optional query parameters:
-author_id: UUID of author to filter
-sort: asc (default) or desc by created_at

Examples:
pgsql
- GET /api/chirps?sort=desc
- GET /api/chirps?author_id=<uuid>
- GET /api/chirps/{id}
- Get a chirp by its ID.

DELETE /api/chirps/{id}
Delete a chirp by ID (only if authenticated user is the author).

<pre>Authorization: Bearer access_token</pre>

Admin
GET /admin/metrics
Returns a simple HTML metrics dashboard for the site.

POST /admin/reset
Resets the DB (only allowed when PLATFORM=dev).

Webhooks
POST /api/polka/webhooks
Handles Polka membership upgrades.

<pre>Authorization: ApiKey POLKA_KEY</pre>

Body:
```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "<uuid>"
  }
}
```
Returns 204 No Content if successful or ignored.

🔐 Environment Variables
Create a .env file with:
<pre>
env
JWT_SECRET=your_secret_key
POLKA_KEY=f271c81ff7084ee5b99a5091b42d486e
PLATFORM=dev
</pre>
🧪 Running the Project
<pre>go run main.go</pre>

🧱 Database
Using sqlc for type-safe SQL queries. Includes tables:

- users
- chirps
- refresh_tokens

✨ Future Improvements
- Pagination support
- Chirp likes and replies
- Follow/follower relationships
- Full frontend SPA

License
MIT © 2025 Keenah VanCampenhout



