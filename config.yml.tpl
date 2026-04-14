addr: :8775
debug: true
database:
  driver: postgres
  dsn: user=postgres password=123456 dbname=techstack host=127.0.0.1 port=5432 sslmode=disable TimeZone=Asia/Shanghai
log_level: debug
log_file: ""
cert: ""
ca: ""
key: ""
cache_dir: .cache
session_hash_key: "这里必填" # 16位随机字符
session_cookie_key: "这里必填"  # 16位随机字符
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
  "apiKey": "这里必填"
  "baseUrl": "https://api.minimaxi.com/v1"
  "provider": "openai"
  "model": "MiniMax-M2.7"
