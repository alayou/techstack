# Docker 默认挂载路径: /etc/techstack/config.yml

addr: :8775
debug: false

database:
  driver: postgres
  dsn: user=postgres password=changeme dbname=techstack host=postgres port=5432 sslmode=disable TimeZone=Asia/Shanghai

log_level: info
log_file: ""

cert: ""
ca: ""
key: ""

cache_dir: /var/lib/techstack/cache

session_hash_key: "replace-with-16-char-key"
session_cookie_key: "replace-with-16-char-key"

passwordhashing:
  bcryptoptions:
    cost: 10
  argon2options:
    memory: 65536
    iterations: 1
    parallelism: 2
  algo: bcrypt

cors:
  allow_origin:
  allow_methods: POST, GET, OPTIONS,DELETE,PUT
  allow_headers: Content-Type,AccessToken,X-CSRF-Token, Authorization, Token,X-Token,X-User-SourceID
  expose_headers: ""
  allow_credentials: "true"

llm:
  apiKey: "YOUR_API_KEY"
  baseUrl: "https://api.minimaxi.com/v1"
  provider: "openai"
  model: "MiniMax-M2.7"
