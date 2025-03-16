# Keycloak SSO Example

This is a complete example of how to use Keycloak for Single Sign-On (SSO) authentication in a Go application.

## Prerequisites

- Go 1.16 or higher
- Docker and Docker Compose (for running Keycloak)

## Getting Started

1. Start Keycloak using Docker Compose:

```bash
make keycloak
```

2. Access the Keycloak admin console at http://localhost:8080/admin with the following credentials:
   - Username: admin
   - Password: admin

3. Configure Keycloak:
   - Create a new realm called `myrealm`
   - Create a new client:
     - Client ID: `myclient`
     - Client Protocol: `openid-connect`
     - Access Type: `confidential`
     - Valid Redirect URIs: `http://localhost:3000/auth/keycloak/callback`
   - After saving, go to the "Credentials" tab to get the client secret
   - Create roles (e.g., `admin`, `user`)
   - Create users and assign roles

4. Copy the `.env.example` file to `.env` and update the values:

```bash
cp .env.example .env
```

5. Update the `KEYCLOAK_CLIENT_SECRET` in the `.env` file with the client secret from Keycloak.

6. Run the example application:

```bash
make run
```

7. Access the application at http://localhost:3000

## Features

- Login with Keycloak
- Public home page
- Protected dashboard page (requires authentication)
- Admin page (requires admin role)
- Logout

## Makefile Commands

- `make keycloak`: Start Keycloak using Docker Compose
- `make run`: Run the example application
- `make clean`: Stop Keycloak and remove containers
- `make all`: Start both Keycloak and the application

## Screenshots

### Home Page (Not Authenticated)
![Home Page](https://example.com/home.png)

### Login Page
![Login Page](https://example.com/login.png)

### Home Page (Authenticated)
![Home Page Authenticated](https://example.com/home-auth.png)

### Dashboard Page
![Dashboard Page](https://example.com/dashboard.png)

### Admin Page
![Admin Page](https://example.com/admin.png)

## License

This example is licensed under the MIT License. 