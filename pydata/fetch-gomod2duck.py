import requests
import json
import time
import duckdb
import datetime
import threading
from queue import Queue, Empty
from datetime import timedelta, date
from concurrent.futures import ThreadPoolExecutor, as_completed
from collections import OrderedDict

# ===================== 配置 =====================
DUCKDB_FILE = "go_index.duckdb"
INDEX_BASE_URL = "https://index.golang.org/index"
MAX_WORKERS = 10
SECOND_STEP = 600
START_YEAR = 2019
BATCH_SIZE = 2000
TIMEOUT = 30
MAX_RETRIES = 3
RATE_LIMIT_DELAY = 0.5  # 请求间隔（秒）
WORKER_TASK_TIMEOUT = 300  # 单个任务超时（秒），超时后标记失败
# =================================================


# ===================== 内置 Bitmap 实现 =====================
class DayBitmap:
    """每天的位图（86400 bits = 10800 bytes）"""
    
    def __init__(self, bytes_data: bytes | None = None):
        self.data = bytearray(bytes_data) if bytes_data else bytearray(86400 // 8)
    
    def set(self, pos: int):
        if 0 <= pos < 86400:
            self.data[pos // 8] |= (1 << (pos % 8))
    
    def get(self, pos: int) -> bool:
        if pos < 0 or pos >= 86400:
            return True
        return (self.data[pos // 8] & (1 << (pos % 8))) != 0
    
    def __getitem__(self, pos: int) -> int:
        return 1 if self.get(pos) else 0
    
    def __setitem__(self, pos: int, value: int):
        if value:
            self.set(pos)
    
    def count(self) -> int:
        return sum(bin(b).count('1') for b in self.data)
    
    def tobytes(self) -> bytes:
        return bytes(self.data)
    
    @staticmethod
    def frombytes(data: bytes) -> 'DayBitmap':
        return DayBitmap(data)


# ===================== DuckDB 连接管理（全局单连接 + 读写锁）=====================
_db_conn: duckdb.DuckDBPyConnection | None = None
_db_lock = threading.Lock()
_write_lock_for_db = threading.Lock()


def get_db_conn() -> duckdb.DuckDBPyConnection:
    """获取全局数据库连接（延迟初始化，线程安全）"""
    global _db_conn
    if _db_conn is None:
        with _write_lock_for_db:
            if _db_conn is None:  # 双重检查锁定
                _db_conn = duckdb.connect(DUCKDB_FILE)
    return _db_conn


def close_db_conn():
    """关闭全局数据库连接"""
    global _db_conn
    with _write_lock_for_db:
        if _db_conn is not None:
            _db_conn.close()
            _db_conn = None


# ===================== DuckDB 写入锁 =====================
_write_lock = threading.Lock()
_batch_buffer: list = []
_buffer_lock = threading.Lock()


# ===================== LRU 位图缓存 =====================
_BITMAP_CACHE_MAX_SIZE = 100  # 最多缓存 100 个日期的位图


class LRUCache:
    """简单的 LRU 缓存"""
    def __init__(self, max_size: int = 100):
        self.max_size = max_size
        self._cache: OrderedDict = OrderedDict()
        self._lock = threading.Lock()
    
    def get(self, key: str) -> DayBitmap | None:
        with self._lock:
            if key in self._cache:
                # 移动到末尾（最常用）
                self._cache.move_to_end(key)
                return self._cache[key]
        return None
    
    def put(self, key: str, value: DayBitmap):
        with self._lock:
            if key in self._cache:
                self._cache.move_to_end(key)
            else:
                if len(self._cache) >= self.max_size:
                    # 移除最旧的
                    self._cache.popitem(last=False)
                self._cache[key] = value
    
    def clear(self):
        with self._lock:
            self._cache.clear()
    
    def remove(self, key: str):
        with self._lock:
            if key in self._cache:
                del self._cache[key]


_bitmap_lru_cache = LRUCache(max_size=_BITMAP_CACHE_MAX_SIZE)


def batch_save_to_duckdb(rows: list):
    """批量写入（线程安全）"""
    if not rows:
        return
    with _buffer_lock:
        _batch_buffer.extend(rows)
        if len(_batch_buffer) >= 1000:
            _flush_buffer()


def _flush_buffer():
    """刷新缓冲区到数据库"""
    global _batch_buffer
    if not _batch_buffer:
        return
    rows_to_write = _batch_buffer.copy()
    _batch_buffer = []
    
    with _write_lock:
        try:
            conn = get_db_conn()
            conn.executemany("""
                INSERT OR IGNORE INTO go_modules (Path, Version, Timestamp)
                VALUES (?, ?, ?)
            """, rows_to_write)
        except Exception as e:
            print(f"写入失败: {e}")
            # 写失败时放回缓冲区
            with _buffer_lock:
                _batch_buffer = rows_to_write + _batch_buffer


# ===================== 数据库初始化 =====================
def init_database():
    """初始化 DuckDB 表结构"""
    # 初始化时直接创建连接，设置全局连接
    global _db_conn
    with _write_lock_for_db:
        if _db_conn is not None:
            _db_conn.close()
        _db_conn = duckdb.connect(DUCKDB_FILE)
        conn = _db_conn
    
    # 创建表结构（在锁外执行，因为初始化时只有主线程）
    conn.execute("""
        CREATE TABLE IF NOT EXISTS go_modules (
            Path        VARCHAR,
            Version     VARCHAR,
            Timestamp   TIMESTAMP,
            PRIMARY KEY (Path, Version, Timestamp)
        )
    """)
    
    conn.execute("CREATE INDEX IF NOT EXISTS idx_go_timestamp ON go_modules(Timestamp)")
    
    conn.execute("""
        CREATE TABLE IF NOT EXISTS sync_bitmap (
            day         VARCHAR PRIMARY KEY,
            bitmap      BLOB,
            total_sec   INTEGER DEFAULT 86400,
            updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    
    conn.execute("""
        CREATE TABLE IF NOT EXISTS sync_tasks (
            day         VARCHAR,
            start_sec   INTEGER,
            end_sec     INTEGER,
            status      VARCHAR DEFAULT 'pending',
            record_count INTEGER DEFAULT 0,
            last_since  VARCHAR,
            updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (day, start_sec)
        )
    """)
    
    conn.execute("""
        CREATE TABLE IF NOT EXISTS sync_progress (
            id          INTEGER PRIMARY KEY,
            last_day    VARCHAR,
            last_sec    INTEGER,
            updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    
    conn.execute("""
        INSERT OR IGNORE INTO sync_progress (id, last_day, last_sec) VALUES (1, NULL, NULL)
    """)
    
    # 初始化后检查并更新 sync_bitmap
    _sync_bitmap_from_modules()


def _sync_bitmap_from_modules():
    """
    从 go_modules 表反向同步 sync_bitmap
    如果 go_modules 中有数据但 sync_bitmap 中没有标记，则更新
    """
    print("🔍 检查 go_modules 数据与 sync_bitmap 一致性...")
    
    # 读取操作使用全局连接
    conn = get_db_conn()
    rows = conn.execute("""
        SELECT "Timestamp" FROM go_modules
    """).fetchall()
    
    if not rows:
        print("✅ go_modules 表为空，无需同步")
        return
    
    # 按日期分组
    day_seconds: dict = {}
    for row in rows:
        ts = row[0]
        # ts 可能是 datetime 对象或字符串
        if isinstance(ts, str):
            dt = parse_timestamp(ts)
        else:
            dt = ts
        
        day_str = dt.strftime("%Y-%m-%d")
        sec = dt.hour * 3600 + dt.minute * 60 + dt.second
        
        if day_str not in day_seconds:
            day_seconds[day_str] = set()
        day_seconds[day_str].add(sec)
    
    # 更新每个日期的 bitmap
    for day, seconds in day_seconds.items():
        ba = get_day_bitmap(day)
        modified = False
        for sec in seconds:
            if not ba.get(sec):
                ba.set(sec)
                modified = True
        
        if modified:
            save_day_bitmap(day, ba)
    
    total_days = len(day_seconds)
    total_secs = sum(len(s) for s in day_seconds.values())
    print(f"✅ 已从 go_modules 同步 {total_secs} 个时间点到 {total_days} 天的 sync_bitmap")


# ===================== 位图操作（线程安全）=====================


def get_day_bitmap(day: str) -> DayBitmap:
    """从 DuckDB 加载某天的位图（LRU 缓存）"""
    bitmap = _bitmap_lru_cache.get(day)
    if bitmap is not None:
        return bitmap
    
    # 使用只读连接查询
    conn = get_db_conn()
    row = conn.execute("SELECT bitmap FROM sync_bitmap WHERE day = ?", [day]).fetchone()
    
    bitmap = DayBitmap() if not row or not row[0] else DayBitmap.frombytes(bytes(row[0]))
    _bitmap_lru_cache.put(day, bitmap)
    
    return bitmap


def save_day_bitmap(day: str, ba: DayBitmap):
    """保存位图到 DuckDB"""
    blob = ba.tobytes()
    with _write_lock:
        conn = get_db_conn()
        conn.execute("""
            INSERT OR REPLACE INTO sync_bitmap (day, bitmap, updated_at)
            VALUES (?, ?, CURRENT_TIMESTAMP)
        """, [day, blob])
    # 更新缓存
    _bitmap_lru_cache.put(day, ba)


def mark_seconds_done(day: str, start_sec: int, end_sec: int):
    """标记一段秒数为已完成"""
    ba = get_day_bitmap(day)
    for sec in range(start_sec, min(end_sec + 1, 86400)):
        ba[sec] = 1
    save_day_bitmap(day, ba)


def is_sec_done(day: str, sec: int) -> bool:
    """检查某秒是否已同步"""
    return get_day_bitmap(day)[sec] == 1


def get_day_coverage(day: str) -> float:
    """获取某天的同步覆盖率"""
    return get_day_bitmap(day).count() / 86400


def get_all_synced_days() -> dict:
    """获取所有已同步天及其覆盖率"""
    conn = get_db_conn()
    rows = conn.execute("SELECT day, bitmap FROM sync_bitmap").fetchall()
    
    result = {}
    for day, blob in rows:
        if blob:
            try:
                ba = DayBitmap.frombytes(bytes(blob))
                result[day] = ba.count() / 86400
            except:
                result[day] = 0.0
    return result


def get_bulk_coverage(start_day: str, end_day: str) -> dict[str, float]:
    """
    批量获取指定日期范围内的覆盖率和位图数据
    用于 generate_all_tasks 避免逐日查询
    """
    conn = get_db_conn()
    rows = conn.execute("""
        SELECT day, bitmap FROM sync_bitmap 
        WHERE day >= ? AND day <= ?
    """, [start_day, end_day]).fetchall()
    
    result = {}
    for day, blob in rows:
        if blob:
            try:
                ba = DayBitmap.frombytes(bytes(blob))
                result[day] = ba.count() / 86400
            except:
                result[day] = 0.0
        else:
            result[day] = 0.0
    return result


def get_bulk_task_status(days: list[str]) -> dict[str, dict[int, str]]:
    """
    批量获取指定日期的所有任务状态
    返回 {day: {start_sec: status}}
    """
    if not days:
        return {}
    
    conn = get_db_conn()
    placeholders = ",".join(["?" for _ in days])
    rows = conn.execute(f"""
        SELECT day, start_sec, status FROM sync_tasks 
        WHERE day IN ({placeholders})
    """, days).fetchall()
    
    result: dict[str, dict[int, str]] = {}
    for day, start_sec, status in rows:
        if day not in result:
            result[day] = {}
        result[day][start_sec] = status
    return result


# ===================== 任务表操作 =====================
def save_task_status(day: str, start_sec: int, end_sec: int, status: str, 
                     record_count: int = 0, last_since: str | None = None):
    """保存任务状态"""
    with _write_lock:
        conn = get_db_conn()
        conn.execute("""
            INSERT OR REPLACE INTO sync_tasks 
            (day, start_sec, end_sec, status, record_count, last_since, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
        """, [day, start_sec, end_sec, status, record_count, last_since])


def get_pending_tasks(limit: int = 100) -> list:
    """获取待处理任务"""
    conn = get_db_conn()
    rows = conn.execute("""
        SELECT day, start_sec, end_sec FROM sync_tasks 
        WHERE status IN ('pending', 'failed')
        ORDER BY day, start_sec LIMIT ?
    """, [limit]).fetchall()
    return rows


def get_task_status(day: str, start_sec: int) -> str | None:
    """获取任务状态"""
    conn = get_db_conn()
    row = conn.execute("""
        SELECT status FROM sync_tasks WHERE day = ? AND start_sec = ?
    """, [day, start_sec]).fetchone()
    return row[0] if row else None


def update_progress(day: str, sec: int):
    """更新全局进度"""
    with _write_lock:
        conn = get_db_conn()
        conn.execute("""
            UPDATE sync_progress SET last_day = ?, last_sec = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = 1
        """, [day, sec])


def get_last_progress() -> tuple | None:
    """获取上次进度"""
    conn = get_db_conn()
    row = conn.execute("SELECT last_day, last_sec FROM sync_progress WHERE id = 1").fetchone()
    return (row[0], row[1]) if row and row[0] else None


# ===================== 时间工具 =====================
def parse_timestamp(ts: str) -> datetime.datetime:
    """安全解析 ISO 格式时间戳"""
    # 处理 Z 后缀
    ts_clean = ts.rstrip("Z")
    if not ts_clean.endswith(("+00:00", "-00:00")):
        ts_clean = ts_clean + "+00:00"
    return datetime.datetime.fromisoformat(ts_clean)


def datetime_to_day(dt: datetime.datetime) -> str:
    return dt.strftime("%Y-%m-%d")


def sec_to_datetime(base_day: str, sec: int) -> datetime.datetime:
    dt = datetime.datetime.strptime(base_day, "%Y-%m-%d")
    return dt + timedelta(seconds=sec)


def day_to_datetime(day: str) -> datetime.datetime:
    return datetime.datetime.strptime(day, "%Y-%m-%d")


# ===================== 核心下载逻辑 =====================
_rate_limiter = threading.Semaphore(1)
_last_request_time = 0.0
_rate_lock = threading.Lock()


def _rate_limit():
    """请求频率控制"""
    global _last_request_time
    with _rate_lock:
        now = time.time()
        elapsed = now - _last_request_time
        if elapsed < RATE_LIMIT_DELAY:
            time.sleep(RATE_LIMIT_DELAY - elapsed)
        _last_request_time = time.time()


def fetch_range(since_dt: datetime.datetime, retry_count: int = 0) -> tuple[list, str | None]:
    """拉取一个时间片数据（带频率控制）"""
    since = since_dt.strftime("%Y-%m-%dT%H:%M:%SZ")
    url = f"{INDEX_BASE_URL}?since={since}&limit={BATCH_SIZE}&include=all"
    
    _rate_limit()
    
    try:
        resp = requests.get(url, headers={"User-Agent": "osv-scalibr/0.4.5"}, timeout=TIMEOUT)
        
        if resp.status_code == 429:
            if retry_count < MAX_RETRIES:
                wait_time = (retry_count + 1) * 2
                print(f"[限流] 等待 {wait_time}s...")
                time.sleep(wait_time)
                return fetch_range(since_dt, retry_count + 1)
            else:
                raise Exception("超过最大重试次数")
        
        resp.raise_for_status()
        lines = resp.text.strip().splitlines()
        
        if not lines:
            return [], None
        
        records = []
        last_ts = None
        
        for line in lines:
            try:
                obj = json.loads(line)
                records.append({
                    "path": obj["Path"],
                    "version": obj["Version"],
                    "timestamp": obj["Timestamp"]
                })
                last_ts = obj["Timestamp"]
            except:
                continue
        
        return records, last_ts
        
    except Exception as e:
        if retry_count < MAX_RETRIES:
            wait_time = (retry_count + 1) * 2
            print(f"[重试 {retry_count+1}] {since} -> {e}, 等待 {wait_time}s")
            time.sleep(wait_time)
            return fetch_range(since_dt, retry_count + 1)
        else:
            print(f"[失败] {since} -> 超过最大重试次数: {e}")
            return [], None


# ===================== Worker =====================
_shutdown_flag = False
_worker_active_count = 0
_active_lock = threading.Lock()


def worker(task_queue: Queue, stats: dict, stats_lock: threading.Lock):
    """并发工作协程（带任务超时）"""
    global _worker_active_count, _shutdown_flag
    
    with _active_lock:
        _worker_active_count += 1
    
    def process_task(task, stats, stats_lock):
        """处理单个任务（供超时机制调用）"""
        day, start_sec = task
        
        if is_sec_done(day, start_sec):
            return True  # 已完成
        
        end_sec = min(start_sec + SECOND_STEP, 86399)
        since_dt = sec_to_datetime(day, start_sec)
        
        save_task_status(day, start_sec, end_sec, 'running', 0, since_dt.isoformat())
        
        try:
            records, last_ts = fetch_range(since_dt)
            
            if records:
                rows = []
                for rec in records:
                    try:
                        dt = parse_timestamp(rec["timestamp"])
                        rows.append((rec["path"], rec["version"], dt))
                    except:
                        continue
                
                if rows:
                    batch_save_to_duckdb(rows)
                    
                    with stats_lock:
                        stats['total_records'] += len(rows)
                        stats['total_tasks'] += 1
                    
                    print(f"✅ {day} {start_sec}-{end_sec} | {len(rows)} | 累计: {stats['total_records']}")
                else:
                    print(f"⚠️  {day} {start_sec}-{end_sec} | 无数据")
                
                mark_seconds_done(day, start_sec, end_sec)
                save_task_status(day, start_sec, end_sec, 'done', len(records), last_ts)
                update_progress(day, end_sec)
            else:
                save_task_status(day, start_sec, end_sec, 'failed')
            
            return True  # 任务完成
            
        except Exception as e:
            print(f"❌ {day} {start_sec}-{end_sec} | {e}")
            save_task_status(day, start_sec, end_sec, 'failed')
            return True  # 任务完成（无论成功失败）
    
    try:
        while not _shutdown_flag:
            task = None
            try:
                task = task_queue.get(timeout=2)
            except Empty:
                # 队列空时检查是否所有任务都完成了
                if task_queue.empty():
                    with _active_lock:
                        active = _worker_active_count
                    # 等待其他 worker 完成
                    time.sleep(1)
                    if task_queue.empty():
                        break
                continue
            
            if task is None:  # 终止信号
                task_queue.task_done()
                break
            
            # 使用定时器实现任务超时
            task_done_flag = threading.Event()
            task_exception = [None]  # type: list[Exception | None]
            
            def task_wrapper():
                try:
                    process_task(task, stats, stats_lock)
                except Exception as e:
                    task_exception[0] = e
                finally:
                    task_done_flag.set()
            
            t = threading.Thread(target=task_wrapper)
            t.start()
            t.join(timeout=WORKER_TASK_TIMEOUT)
            
            if t.is_alive():
                # 超时了，任务还在运行
                print(f"⏰ {task[0]} {task[1]}s | 任务超时 ({WORKER_TASK_TIMEOUT}s)")
                save_task_status(task[0], task[1], task[1] + SECOND_STEP, 'failed')
            elif task_exception[0]:
                print(f"❌ {task[0]} {task[1]}s | 异常: {task_exception[0]}")
                save_task_status(task[0], task[1], task[1] + SECOND_STEP, 'failed')
            
            task_queue.task_done()
    finally:
        with _active_lock:
            _worker_active_count -= 1


# ===================== 增量同步任务生成 =====================
def check_and_generate_tasks() -> Queue:
    """
    增量同步：查询 go_modules 时间戳范围，检查 sync_bitmap 覆盖，
    按时间倒序生成未覆盖的同步任务
    """
    print("📋 增量同步任务生成中...")
    task_queue = Queue()
    
    conn = get_db_conn()
    
    # 1. 查询 go_modules 表的时间戳范围
    print("  - 查询 go_modules 时间戳范围...")
    row = conn.execute("""
        SELECT MIN("Timestamp"), MAX("Timestamp") FROM go_modules
    """).fetchone()
    
    if not row or not row[0]:
        print("  - go_modules 表为空，无需生成任务")
        return task_queue
    
    min_ts, max_ts = row
    print(f"  - go_modules 时间范围: {min_ts} ~ {max_ts}")
    
    # 解析时间戳
    if isinstance(min_ts, str):
        min_dt = parse_timestamp(min_ts)
    else:
        min_dt = min_ts
    if isinstance(max_ts, str):
        max_dt = parse_timestamp(max_ts)
    else:
        max_dt = max_ts
    
    # 2. 获取这个范围内的所有 sync_bitmap 数据
    min_day = min_dt.strftime("%Y-%m-%d")
    max_day = max_dt.strftime("%Y-%m-%d")
    print(f"  - 加载 sync_bitmap ({min_day} ~ {max_day})...")
    coverage_map = get_bulk_coverage(min_day, max_day)
    
    # 3. 批量获取任务状态
    print("  - 加载任务状态...")
    days_in_range = []
    current = min_dt
    while current <= max_dt:
        days_in_range.append(current.strftime("%Y-%m-%d"))
        current += timedelta(days=1)
    
    task_status_map: dict[str, dict[int, str]] = {}
    BATCH_SIZE_DAYS = 100
    for i in range(0, len(days_in_range), BATCH_SIZE_DAYS):
        batch = days_in_range[i:i + BATCH_SIZE_DAYS]
        batch_status = get_bulk_task_status(batch)
        task_status_map.update(batch_status)
    
    # 4. 构建任务队列（倒序：最新时间优先）
    print("  - 构建任务队列...")
    all_tasks = []
    
    # 从最大时间向前遍历
    current = max_dt
    while current >= min_dt:
        day_str = current.strftime("%Y-%m-%d")
        day_coverage = coverage_map.get(day_str, 0.0)
        
        # 跳过已完成的天（覆盖率 >= 99%）
        if day_coverage < 0.99:
            day_tasks = task_status_map.get(day_str, {})
            day_start_sec = 0
            
            # 如果是第一天，从实际时间开始
            if day_str == min_day:
                day_start_sec = min_dt.hour * 3600 + min_dt.minute * 60 + min_dt.second
                # 向下对齐到 SECOND_STEP
                day_start_sec = (day_start_sec // SECOND_STEP) * SECOND_STEP
            
            # 如果是最后一天，到实际时间结束
            day_end_sec = 86399
            if day_str == max_day:
                day_end_sec = max_dt.hour * 3600 + max_dt.minute * 60 + max_dt.second
                # 向上对齐到 SECOND_STEP
                day_end_sec = ((day_end_sec + SECOND_STEP - 1) // SECOND_STEP) * SECOND_STEP - 1
            
            for sec in range(day_start_sec, min(day_end_sec + 1, 86400), SECOND_STEP):
                status = day_tasks.get(sec)
                # 只处理未完成的任务
                if status != 'done':
                    # 检查 sync_tasks 表中是否有 running 或 pending 状态的任务
                    if status in ('running', 'pending'):
                        all_tasks.append((day_str, sec))
                    # 或者检查位图是否真的未覆盖
                    elif not is_sec_done(day_str, sec):
                        all_tasks.append((day_str, sec))
        
        current -= timedelta(days=1)
    
    # 5. 倒序放入队列（确保最新时间先处理）
    print(f"  - 待处理任务数: {len(all_tasks)}")
    for day, sec in reversed(all_tasks):
        task_queue.put((day, sec))
    
    return task_queue


# ===================== 状态显示 =====================
def show_status():
    """显示同步状态"""
    print("=" * 60)
    print("Go Module 同步状态")
    print("=" * 60)
    
    conn = get_db_conn()
    try:
        total = conn.execute("SELECT COUNT(*) FROM go_modules").fetchone()
        unique = conn.execute("SELECT COUNT(DISTINCT Path) FROM go_modules").fetchone()
        total = total[0] if total else 0
        unique = unique[0] if unique else 0
        print(f"📊 总版本数: {total:,}")
        print(f"📊 唯一Go包数: {unique:,}")
    except:
        print("📊 暂无数据")
    
    synced_days = get_all_synced_days()
    if synced_days:
        fully_synced = sum(1 for v in synced_days.values() if v >= 0.99)
        avg = sum(synced_days.values()) / len(synced_days) if synced_days else 0
        print(f"📅 已同步天数: {len(synced_days)}")
        print(f"📅 完全同步: {fully_synced}")
        print(f"📅 平均覆盖率: {avg * 100:.2f}%")
    else:
        print("📅 尚未开始")
    
    try:
        pending = conn.execute("SELECT COUNT(*) FROM sync_tasks WHERE status = 'pending'").fetchone()
        running = conn.execute("SELECT COUNT(*) FROM sync_tasks WHERE status = 'running'").fetchone()
        failed = conn.execute("SELECT COUNT(*) FROM sync_tasks WHERE status = 'failed'").fetchone()
        print(f"📋 待处理: {pending[0] if pending else 0}")
        print(f"📋 运行中: {running[0] if running else 0}")
        print(f"📋 失败: {failed[0] if failed else 0}")
    except:
        pass
    
    row = conn.execute("SELECT last_day, last_sec FROM sync_progress WHERE id = 1").fetchone()
    if row and row[0]:
        print(f"🕐 上次进度: {row[0]} {row[1]}s")
    
    print("=" * 60)


# ===================== 主入口 =====================
def run_concurrent_sync():
    """启动并发同步（增量模式）"""
    global _shutdown_flag, _worker_active_count
    
    print("🚀 启动 Go Module 索引同步（增量模式）")
    print(f"📡 {INDEX_BASE_URL}")
    print(f"👥 并发数: {MAX_WORKERS}")
    print(f"⏱️  步长: {SECOND_STEP}s")
    print(f"🔄 最大重试: {MAX_RETRIES}")
    print(f"⏱️  请求间隔: {RATE_LIMIT_DELAY}s")
    print("-" * 60)
    
    _shutdown_flag = False
    init_database()
    
    # 刷新可能残留的缓冲区
    _flush_buffer()
    
    # 生成增量同步任务
    task_queue = check_and_generate_tasks()

    if task_queue.qsize() == 0:
        print("✅ 全部完成，无需同步")
        show_status()
        return

    print(f"📋 待同步任务: {task_queue.qsize()}")
    print("-" * 60)
    
    stats = {'total_records': 0, 'total_tasks': 0}
    stats_lock = threading.Lock()
    
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        futures = []
        for _ in range(MAX_WORKERS):
            future = executor.submit(worker, task_queue, stats, stats_lock)
            futures.append(future)
        
        # 等待队列清空
        task_queue.join()
        
        # 优雅关闭
        _shutdown_flag = True
        
        # 等待所有 worker 退出
        for f in as_completed(futures):
            try:
                f.result(timeout=5)
            except:
                pass

    # 最后刷新一次缓冲区
    _flush_buffer()
    
    print("-" * 60)
    print(f"🎉 完成! 任务: {stats['total_tasks']}, 记录: {stats['total_records']}")
    show_status()


def run_single_day(day: str):
    """单独同步某一天"""
    global _shutdown_flag
    
    print(f"🔄 同步 {day}")
    _shutdown_flag = False
    init_database()
    
    task_queue = Queue()
    for sec in range(0, 86400, SECOND_STEP):
        if not is_sec_done(day, sec):
            task_queue.put((day, sec))
    
    if task_queue.qsize() == 0:
        print(f"✅ {day} 已完成")
        return
    
    print(f"📋 任务数: {task_queue.qsize()}")
    
    stats = {'total_records': 0, 'total_tasks': 0}
    stats_lock = threading.Lock()
    
    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as executor:
        futures = []
        for _ in range(MAX_WORKERS):
            executor.submit(worker, task_queue, stats, stats_lock)
        
        task_queue.join()
        _shutdown_flag = True
    
    _flush_buffer()
    print(f"✅ {day} 完成, {stats['total_records']} 条")


if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1:
        cmd = sys.argv[1]
        if cmd == "status":
            init_database()
            show_status()
        elif cmd == "sync":
            run_concurrent_sync()
        elif cmd == "day" and len(sys.argv) > 2:
            run_single_day(sys.argv[2])
        else:
            print("用法: python fetch-gomod2duck.py [status|sync|day DATE]")
    else:
        run_concurrent_sync()
