# Agnos Health Backend Assessment

A backend service built with Go (Gin), PostgreSQL, and Nginx reverse proxy for hospital staff authentication and scoped patient search.

## Tech Stack
- **Language:** Go 1.22+ (Gin Web Framework, GORM)
- **Database:** PostgreSQL 15
- **Reverse Proxy:** Nginx
- **Containerization:** Docker & Docker Compose
- **Testing:** Testify

---

## Getting Started

### 1. Prerequisites
- Docker & Docker Compose installed

### 2. Run with Docker Compose
Clone the repository and spin up the full stack:

```bash
docker compose up --build -d

3. Run Unit Tests
To execute all automated unit tests (including positive & negative scenarios):

go test -v .

API Endpoints

1. Create Staff
Method: POST

Path: /staff/create

Body:

JSON
{
  "username": "admin_nurse",
  "password": "pass1234",
  "hospital": "Hospital A"
}

2. Staff Login
Method: POST

Path: /staff/login

Body:

JSON
{
  "username": "admin_nurse",
  "password": "pass1234",
  "hospital": "Hospital A"
}
Response:

JSON
{
  "token": "<JWT_TOKEN>"
}

3. Patient Search (Protected)
Method: GET

Path: /patient/search

Headers: Authorization: Bearer <JWT_TOKEN>

Query Parameters (Optional):

national_id, passport_id, first_name, middle_name, last_name, date_of_birth, phone_number, email