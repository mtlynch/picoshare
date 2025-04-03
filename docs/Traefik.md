
### why serve Picoshare with Traefik?

In Picoshare, it handles HTTP(S) traffic, and manages SSL certificates with Let's Encrypt, ensuring secure connections. This simplifies deployment by automating routing and SSL setup.

**Create `acme.json` file**:
   - Before starting the services, create an empty `acme.json` file in `/opt/traefik/` with proper permissions (`chmod 600 acme.json`). This file will store the SSL certificates.

### Docker Compose
 - The `docker-compose.yml` file already includes the required services for Traefik and Picoshare, with proper labels for routing and SSL.

```bash
# Docker Compose Command to Run Picoshare with Traefik
 docker-compose up -d
```
### Docker Compose Environment Variables for Traefik and Picoshare

To configure Traefik for Picoshare within Docker Compose, set the following environment variables:

```bash
# PicoShare Configuration 
PS_SHARE_SHARED_SECRET=demigod                 # change the password to secure one 
PS_SHARE_DOMAIN=example.domain.com             # add your domain name 

# Traefik Configuration
TRAEFIK_ACME_EMAIL=youremail@email.com         # Email for Let's Encrypt (replace with your email)
TRAEFIK_ACME_STORAGE_PATH=/acme.json            # Path to store Let's Encrypt certificates
```
