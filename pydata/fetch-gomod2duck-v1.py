import requests
import json
import time
from datetime import datetime
import duckdb

# ===================== 配置 =====================
START_SINCE = "2019-01-01T00:00:00Z"  # 首次从这里开始
DUCKDB_FILE = "go_index.duckdb"        # 输出数据库
BATCH_SIZE = 2000                      # 固定
TIMEOUT = 60
# =================================================

# 1. 初始化 DuckDB 表
def init_db():
    conn = duckdb.connect(DUCKDB_FILE)
    conn.execute("""
        CREATE TABLE IF NOT EXISTS go_modules (
            Path VARCHAR,
            Version VARCHAR,
            Timestamp TIMESTAMP,
            PRIMARY KEY (Path, Version)
        )
    """)

    # 创建进度记录表
    conn.execute("""
        CREATE TABLE IF NOT EXISTS progress (
            last_since VARCHAR PRIMARY KEY
        )
    """)
    conn.execute("CREATE INDEX IF NOT EXISTS idx_go_timestamp ON go_modules(Timestamp)")
    conn.close()

# 2. 获取上次进度
def get_last_since():
    try:
        conn = duckdb.connect(DUCKDB_FILE)
        row = conn.execute("SELECT last_since FROM progress").fetchone()
        conn.close()
        return row[0] if row else START_SINCE
    except:
        return START_SINCE

# 3. 保存进度
def save_progress(since):
    conn = duckdb.connect(DUCKDB_FILE)
    conn.execute("DELETE FROM progress")
    conn.execute("INSERT INTO progress VALUES (?)", [since])
    conn.close()

# 4. 批量写入 DuckDB
def insert_batch(rows):
    if not rows:
        return
    conn = duckdb.connect(DUCKDB_FILE)
    conn.executemany(
        "INSERT OR IGNORE INTO go_modules VALUES (?, ?, ?)",
        rows
    )
    conn.close()

# 5. 拉取一页数据
def fetch_page(since):
    url = f"https://index.golang.org/index?since={since}&limit={BATCH_SIZE}&include=all"
    headers = {"User-Agent": "osv-scalibr/0.4.5"}

    try:
        resp = requests.get(url, headers=headers, timeout=TIMEOUT)
        resp.raise_for_status()
        lines = resp.text.strip().splitlines()
        return lines
    except Exception as e:
        print(f"请求失败: {e}, 重试中...")
        time.sleep(5)
        return fetch_page(since)

# 6. 主同步逻辑
def sync():
    init_db()
    since = get_last_since()
    print(f"从 {since} 开始同步...")

    total_inserted = 0

    while True:
        lines = fetch_page(since)
        if not lines:
            print("✅ 同步完成！")
            break

        batch = []
        last_ts = None

        for line in lines:
            try:
                data = json.loads(line)
                path = data["Path"]
                ver = data["Version"]
                ts = data["Timestamp"]
                dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
                batch.append((path, ver, dt))
                last_ts = ts
            except:
                continue

        insert_batch(batch)
        total_inserted += len(batch)

        # 更新进度
        since = last_ts
        save_progress(since)

        print(f"已写入: {len(batch)} | 总计: {total_inserted} | since: {since}")
        time.sleep(0.5)  # 防限流

if __name__ == "__main__":
    sync()