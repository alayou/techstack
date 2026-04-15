import requests
import tarfile
import duckdb
import os
import io
import time
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

# ===================== 配置 =====================
DUCKDB_FILE = "crates_io.duckdb"
TAR_URL = "https://static.crates.io/db-dump.tar.gz"
CHUNK_SIZE = 1024 * 1024 * 10  # 10MB 流式下载
# =================================================


def download_and_import_to_duckdb():
    # 1. 初始化 DuckDB
    conn = duckdb.connect(DUCKDB_FILE)
    print(f"✅ 已打开数据库: {DUCKDB_FILE}")

    # 2. 检查本地是否已存在 tar.gz 文件
    local_tar = "db-dump.tar.gz"
    if os.path.exists(local_tar):
        print(f"📦 本地文件已存在: {local_tar}，跳过下载")
        with open(local_tar, "rb") as f:
            raw_data = f.read()
        raw_stream = io.BytesIO(raw_data)
    else:
        # 流式下载 tar.gz
        print(f"🔽 开始下载: {TAR_URL}")
        session = requests.Session()

        # 配置重试策略
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
        )
        adapter = HTTPAdapter(max_retries=retry_strategy)
        session.mount("https://", adapter)
        session.mount("http://", adapter)

        response = session.get(TAR_URL, stream=True, timeout=1800)
        response.raise_for_status()

        # 内存解压 - 使用 BytesIO 缓冲响应体以支持 seek
        raw_data = response.raw.read()
        raw_stream = io.BytesIO(raw_data)

    with tarfile.open(fileobj=raw_stream, mode="r|gz") as tar:
        for item in tar:
            if not item.isfile() or not item.name.endswith(".csv"):
                continue

            # 只处理 data 目录下的 CSV
            if "/data/" not in item.name:
                continue

            # 表名 = 文件名（去掉 .csv）
            table_name = os.path.basename(item.name).replace(".csv", "")
            print(f"\n📂 导入表: {table_name} → {item.name}")

            # 读取 CSV 内容到内存
            f = tar.extractfile(item)
            if not f:
                continue

            csv_data = f.read()

            # DuckDB 无法直接读取 BytesIO，需要先写入临时文件
            import tempfile
            with tempfile.NamedTemporaryFile(mode='wb', suffix='.csv', delete=False) as tmp:
                tmp.write(csv_data)
                tmp_path = tmp.name

            # 自动创建表并导入
            try:
                conn.execute(f"""
                    CREATE TABLE IF NOT EXISTS {table_name} AS
                    SELECT * FROM read_csv_auto('{tmp_path}', HEADER=TRUE, IGNORE_ERRORS=TRUE)
                """)
            except Exception as e:
                print(f"⚠️ 表 {table_name} 导入失败: {e}")
            finally:
                os.unlink(tmp_path)  # 清理临时文件

            # 统计行数（如果导入失败则跳过统计）
            if table_name in [row[0] for row in conn.execute("SHOW TABLES").fetchall()]:
                result = conn.execute(f"SELECT COUNT(*) FROM {table_name}").fetchone()
                row_count = result[0] if result else 0
                print(f"✅ {table_name} 导入完成，行数: {row_count}")
            continue

    conn.close()
    print("\n🎉 所有 crates.io 数据已完整导入 DuckDB！")


if __name__ == "__main__":
    start = time.time()
    download_and_import_to_duckdb()
    print(f"⏱ 总耗时: {time.time() - start:.2f}s")