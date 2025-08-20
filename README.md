## Sargantana Go
### What is this?
I needed to build a performant web application but I wanted it to be simple to mantain and provide support for regular real life scenarios like OAuth2, modern frontend or database... So I came up with this.

### Features
- Web server using [Gin](https://github.com/gin-gonic/gin)
- OAuth2 support via [Goth: Multi-Provider Authentication for Go](https://github.com/markbates/goth)
- Out of the box OAuth2 user authentication flow implemented in the backend.
- [React](https://es.react.dev/) frontend via [Vite](https://vite.dev/guide/)
- Extensible and dead simple controller mechanism with session support via Redis or cookies, you choose your preferred flavour.
- Easy configuration for production.
- The full stack can be run locally using Docker Compose. It also has proper secrets management via [Docker Compose secrets](https://docs.docker.com/compose/how-tos/use-secrets/). You need to setup a `.secrets` folder with the secrets you need. This means that the Dockerfile is pretty much ready for production too.

### To do list (in priority order)
- Add more database support (Postgres, MySQL, MongoDB, etc).
